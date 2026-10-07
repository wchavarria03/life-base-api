package services

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"

	"life-base-api/app/internal/auth"
	"life-base-api/app/internal/models"
)

// ── Repository interfaces ───────────────────────────────────────────────────

type ChildProfileRepository interface {
	List(ctx context.Context) ([]*models.ChildProfile, error)
	FindByID(ctx context.Context, id string) (*models.ChildProfile, error)
	FindByEmail(ctx context.Context, email string) (*models.ChildProfile, error)
	Create(ctx context.Context, input models.ChildProfileInput) (*models.ChildProfile, error)
	Update(ctx context.Context, id string, fields map[string]any) (*models.ChildProfile, error)
	Delete(ctx context.Context, id string) error
}

type WalletTransactionRepository interface {
	ListByChildID(ctx context.Context, childID string) ([]*models.WalletTransaction, error)
	ExistsForTask(ctx context.Context, taskID, reason string) (bool, error)
	Create(ctx context.Context, input models.WalletTransaction) (*models.WalletTransaction, error)
}

type ShopItemRepository interface {
	List(ctx context.Context) ([]*models.ShopItem, error)
	FindByID(ctx context.Context, id string) (*models.ShopItem, error)
	Create(ctx context.Context, input models.ShopItemInput) (*models.ShopItem, error)
	Update(ctx context.Context, id string, fields map[string]any) (*models.ShopItem, error)
	Delete(ctx context.Context, id string) error
}

type ShopOrderRepository interface {
	ListByChildID(ctx context.Context, childID string) ([]*models.ShopOrder, error)
	ListPending(ctx context.Context) ([]*models.ShopOrder, error)
	FindByID(ctx context.Context, id string) (*models.ShopOrder, error)
	Create(ctx context.Context, order models.ShopOrder) (*models.ShopOrder, error)
	Update(ctx context.Context, id string, fields map[string]any) (*models.ShopOrder, error)
}

// ── Child profiles ───────────────────────────────────────────────────────────

// ChildProfileService manages child profiles — plain parent-owned CRUD,
// same shape as every other owned resource in this app, plus PIN handling
// for the Kid Mode (PIN) login path.
type ChildProfileService struct {
	profiles ChildProfileRepository
}

// NewChildProfileService constructs a ChildProfileService.
func NewChildProfileService(profiles ChildProfileRepository) *ChildProfileService {
	return &ChildProfileService{profiles: profiles}
}

func (s *ChildProfileService) List(ctx context.Context) ([]*models.ChildProfile, error) {
	return s.profiles.List(ctx)
}

func (s *ChildProfileService) FindByID(ctx context.Context, id string) (*models.ChildProfile, error) {
	return s.profiles.FindByID(ctx, id)
}

// FindMyProfile returns the caller's own linked child profile, if any
// (nil, nil if the caller isn't a linked child) — used by GET /v1/me.
func (s *ChildProfileService) FindMyProfile(ctx context.Context, email string) (*models.ChildProfile, error) {
	if email == "" {
		return nil, nil
	}
	return s.profiles.FindByEmail(ctx, email)
}

func (s *ChildProfileService) Create(ctx context.Context, input models.ChildProfileInput) (*models.ChildProfile, error) {
	userID := auth.UserIDFromContext(ctx)
	if userID == "" {
		return nil, fmt.Errorf("no authenticated user")
	}
	if input.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	input.OwnerUserID = userID
	if input.PIN != nil && *input.PIN != "" {
		hash, err := hashPIN(*input.PIN)
		if err != nil {
			return nil, err
		}
		input.PinHash = &hash
	}
	input.PIN = nil
	return s.profiles.Create(ctx, input)
}

func (s *ChildProfileService) Update(ctx context.Context, id string, fields map[string]any) (*models.ChildProfile, error) {
	if len(fields) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}
	if pin, ok := fields["pin"]; ok {
		delete(fields, "pin")
		pinStr, _ := pin.(string)
		if pinStr != "" {
			hash, err := hashPIN(pinStr)
			if err != nil {
				return nil, err
			}
			fields["pin_hash"] = hash
		}
	}
	return s.profiles.Update(ctx, id, fields)
}

func (s *ChildProfileService) Delete(ctx context.Context, id string) error {
	return s.profiles.Delete(ctx, id)
}

// VerifyPIN checks pin against childID's stored hash — used by the Kid Mode
// (PIN) entry flow. Runs under the parent's own ctx: RLS already scopes
// FindByID to profiles the caller owns, so a parent can only verify a PIN
// for their own kid.
func (s *ChildProfileService) VerifyPIN(ctx context.Context, childID, pin string) (bool, error) {
	profile, err := s.profiles.FindByID(ctx, childID)
	if err != nil {
		return false, err
	}
	if profile == nil || profile.PinHash == nil {
		return false, nil
	}
	return bcrypt.CompareHashAndPassword([]byte(*profile.PinHash), []byte(pin)) == nil, nil
}

func hashPIN(pin string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(pin), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash pin: %w", err)
	}
	return string(hash), nil
}

// ── Wallet ───────────────────────────────────────────────────────────────────

// WalletService reads a child's coin ledger and — the one cross-domain
// piece — completes a kid-assigned task (awarding coins) and applies daily
// penalties. Every write here runs under auth.WithServiceRole, after an
// explicit Go-level authorization check; see auth.WithServiceRole's doc
// comment for why.
type WalletService struct {
	tasks         *TaskService
	childProfiles ChildProfileRepository
	walletTx      WalletTransactionRepository
}

// NewWalletService constructs a WalletService.
func NewWalletService(tasks *TaskService, childProfiles ChildProfileRepository, walletTx WalletTransactionRepository) *WalletService {
	return &WalletService{tasks: tasks, childProfiles: childProfiles, walletTx: walletTx}
}

// Wallet returns childID's balance (sum of their ledger) and transaction
// history. Runs under the caller's own ctx — RLS's "linked child can view
// own ledger" / "owner views own kids' ledger" policies decide visibility.
func (s *WalletService) Wallet(ctx context.Context, childID string) (*models.Wallet, error) {
	txns, err := s.walletTx.ListByChildID(ctx, childID)
	if err != nil {
		return nil, fmt.Errorf("list wallet transactions: %w", err)
	}
	balance := 0
	for _, t := range txns {
		balance += t.Amount
	}
	return &models.Wallet{Balance: balance, Transactions: txns}, nil
}

// CompleteTask is the single entry point task completion should go
// through — ordinary tasks (no assigned child) delegate straight to
// TaskService.Complete under the caller's own ctx, unchanged. A
// kid-assigned task additionally authorizes the caller (parent or the
// linked child — checked by attempting to read the child's profile under
// the caller's own ctx/RLS, which only a parent-owner or the linked child
// themself can see) and, on a genuine incomplete->complete transition,
// awards CoinValue coins exactly once (guarded by ExistsForTask).
func (s *WalletService) CompleteTask(ctx context.Context, taskID string) (*models.Task, error) {
	task, err := s.tasks.FindByID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("find task: %w", err)
	}
	if task == nil {
		return nil, fmt.Errorf("task not found")
	}
	if task.AssignedChildID == nil {
		return s.tasks.Complete(ctx, taskID)
	}

	child, err := s.childProfiles.FindByID(ctx, *task.AssignedChildID)
	if err != nil {
		return nil, fmt.Errorf("find child profile: %w", err)
	}
	if child == nil {
		return nil, fmt.Errorf("not authorized to complete this task")
	}

	wasIncomplete := task.CompletedAt == nil
	svcCtx := auth.WithServiceRole(ctx)
	updated, err := s.tasks.Complete(svcCtx, taskID)
	if err != nil {
		return nil, err
	}

	if wasIncomplete && task.CoinValue != nil && *task.CoinValue != 0 {
		exists, err := s.walletTx.ExistsForTask(svcCtx, taskID, "task_complete")
		if err == nil && !exists {
			_, _ = s.walletTx.Create(svcCtx, models.WalletTransaction{
				ChildID: *task.AssignedChildID,
				Amount:  *task.CoinValue,
				Reason:  "task_complete",
				TaskID:  &taskID,
			})
		}
	}
	return updated, nil
}

// ── Shop ─────────────────────────────────────────────────────────────────────

// ShopService manages parent-defined shop items and child purchases.
type ShopService struct {
	items         ShopItemRepository
	orders        ShopOrderRepository
	childProfiles ChildProfileRepository
	walletTx      WalletTransactionRepository
}

// NewShopService constructs a ShopService.
func NewShopService(items ShopItemRepository, orders ShopOrderRepository, childProfiles ChildProfileRepository, walletTx WalletTransactionRepository) *ShopService {
	return &ShopService{items: items, orders: orders, childProfiles: childProfiles, walletTx: walletTx}
}

func (s *ShopService) ListItems(ctx context.Context) ([]*models.ShopItem, error) {
	return s.items.List(ctx)
}

func (s *ShopService) CreateItem(ctx context.Context, input models.ShopItemInput) (*models.ShopItem, error) {
	userID := auth.UserIDFromContext(ctx)
	if userID == "" {
		return nil, fmt.Errorf("no authenticated user")
	}
	if input.Name == "" || input.CoinCost == nil {
		return nil, fmt.Errorf("name and coin_cost are required")
	}
	input.OwnerUserID = userID
	return s.items.Create(ctx, input)
}

func (s *ShopService) UpdateItem(ctx context.Context, id string, fields map[string]any) (*models.ShopItem, error) {
	if len(fields) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}
	return s.items.Update(ctx, id, fields)
}

func (s *ShopService) DeleteItem(ctx context.Context, id string) error {
	return s.items.Delete(ctx, id)
}

func (s *ShopService) ListOrdersByChild(ctx context.Context, childID string) ([]*models.ShopOrder, error) {
	return s.orders.ListByChildID(ctx, childID)
}

func (s *ShopService) ListPendingOrders(ctx context.Context) ([]*models.ShopOrder, error) {
	return s.orders.ListPending(ctx)
}

// Purchase buys itemID for childID: authorizes the caller the same way
// CompleteTask does (can they even read this child's profile?), confirms
// the item is active and belongs to that child's own parent, confirms
// sufficient balance, then inserts the order and a debit transaction — both
// under auth.WithServiceRole, since a child legitimately spending their own
// coins is still, mechanically, a write to rows owned by their parent.
func (s *ShopService) Purchase(ctx context.Context, childID, itemID string) (*models.ShopOrder, error) {
	child, err := s.childProfiles.FindByID(ctx, childID)
	if err != nil {
		return nil, fmt.Errorf("find child profile: %w", err)
	}
	if child == nil {
		return nil, fmt.Errorf("not authorized to purchase for this child")
	}

	item, err := s.items.FindByID(ctx, itemID)
	if err != nil {
		return nil, fmt.Errorf("find shop item: %w", err)
	}
	if item == nil || !item.Active || item.OwnerUserID != child.OwnerUserID {
		return nil, fmt.Errorf("shop item not available")
	}

	svcCtx := auth.WithServiceRole(ctx)
	txns, err := s.walletTx.ListByChildID(svcCtx, childID)
	if err != nil {
		return nil, fmt.Errorf("list wallet transactions: %w", err)
	}
	balance := 0
	for _, t := range txns {
		balance += t.Amount
	}
	if balance < item.CoinCost {
		return nil, fmt.Errorf("not enough coins: have %d, need %d", balance, item.CoinCost)
	}

	order, err := s.orders.Create(svcCtx, models.ShopOrder{
		ChildID:            childID,
		ShopItemID:         itemID,
		CoinCostAtPurchase: item.CoinCost,
		Status:             "pending",
	})
	if err != nil {
		return nil, fmt.Errorf("create order: %w", err)
	}
	_, _ = s.walletTx.Create(svcCtx, models.WalletTransaction{
		ChildID:     childID,
		Amount:      -item.CoinCost,
		Reason:      "shop_purchase",
		ShopOrderID: &order.ID,
	})
	return order, nil
}

// MarkFulfilled sets a pending order to fulfilled — parent-only, runs under
// the caller's own ctx: shop_orders' owner RLS policy already scopes this
// to orders belonging to the caller's own kids.
func (s *ShopService) MarkFulfilled(ctx context.Context, orderID string) (*models.ShopOrder, error) {
	return s.orders.Update(ctx, orderID, map[string]any{
		"status":       "fulfilled",
		"fulfilled_at": time.Now().UTC().Format(time.RFC3339),
	})
}

// ── Daily penalty job ────────────────────────────────────────────────────────

// RunDailyPenalties is the daily cron entry point (cmd/kid_penalties.go,
// run with no user token so every repository call already resolves to the
// service key — see databases.resolveKeys). For every todo-category task
// assigned to a child, with a coin value, past its due date, still
// incomplete, and not already penalized, deducts the child's configured
// penalty fee and marks the task so a re-run or retry can't double-penalize.
func (s *WalletService) RunDailyPenalties(ctx context.Context) (int, error) {
	category := models.TaskTodo
	tasks, err := s.tasks.List(ctx, &category)
	if err != nil {
		return 0, fmt.Errorf("list todo tasks: %w", err)
	}

	children, err := s.childProfiles.List(ctx)
	if err != nil {
		return 0, fmt.Errorf("list child profiles: %w", err)
	}
	childByID := make(map[string]*models.ChildProfile, len(children))
	for _, c := range children {
		childByID[c.ID] = c
	}

	today := time.Now().Format("2006-01-02")
	svcCtx := auth.WithServiceRole(ctx)
	applied := 0
	for _, t := range tasks {
		task := t.Task
		if task.AssignedChildID == nil || task.CoinValue == nil || task.CompletedAt != nil || task.PenaltyAppliedAt != nil {
			continue
		}
		if task.DueDate == nil || *task.DueDate > today {
			continue
		}
		child := childByID[*task.AssignedChildID]
		if child == nil {
			continue
		}
		if _, err := s.walletTx.Create(svcCtx, models.WalletTransaction{
			ChildID: *task.AssignedChildID,
			Amount:  -child.PenaltyFee,
			Reason:  "task_penalty",
			TaskID:  &task.ID,
		}); err != nil {
			return applied, fmt.Errorf("penalize task %s: %w", task.ID, err)
		}
		if _, err := s.tasks.Update(svcCtx, task.ID, map[string]any{"penalty_applied_at": time.Now().UTC().Format(time.RFC3339)}); err != nil {
			return applied, fmt.Errorf("mark task %s penalized: %w", task.ID, err)
		}
		applied++
	}
	return applied, nil
}
