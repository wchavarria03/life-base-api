package supabase

import (
	"context"
	"net/url"

	"life-base-api/app/internal/databases"
	"life-base-api/app/internal/models"
)

const kidsSchema = "kids"

// ── Child profiles ───────────────────────────────────────────────────────────

// ChildProfileRepository persists child profiles.
type ChildProfileRepository struct {
	client *databases.SupabaseClient
}

// NewChildProfileRepository constructs a ChildProfileRepository.
func NewChildProfileRepository(client *databases.SupabaseClient) *ChildProfileRepository {
	return &ChildProfileRepository{client: client}
}

// List returns every child profile the caller owns (or, under RLS, their
// own single linked profile).
func (r *ChildProfileRepository) List(ctx context.Context) ([]*models.ChildProfile, error) {
	return databases.Get[[]*models.ChildProfile](ctx, r.client, "/rest/v1/child_profiles",
		url.Values{"order": []string{"name.asc"}}, kidsSchema)
}

// FindByID returns a child profile by id, or nil if not found/visible.
func (r *ChildProfileRepository) FindByID(ctx context.Context, id string) (*models.ChildProfile, error) {
	return databases.First(databases.Get[[]*models.ChildProfile](ctx, r.client, "/rest/v1/child_profiles",
		url.Values{"id": []string{"eq." + id}, "limit": []string{"1"}}, kidsSchema))
}

// FindByEmail returns the child profile linked to email (the caller's own
// JWT email, typically), or nil. Relies on RLS's "linked child can view own
// row" policy, so this only ever returns a row the caller is entitled to.
func (r *ChildProfileRepository) FindByEmail(ctx context.Context, email string) (*models.ChildProfile, error) {
	return databases.First(databases.Get[[]*models.ChildProfile](ctx, r.client, "/rest/v1/child_profiles",
		url.Values{"email": []string{"eq." + email}, "limit": []string{"1"}}, kidsSchema))
}

// Create inserts a new child profile.
func (r *ChildProfileRepository) Create(ctx context.Context, input models.ChildProfileInput) (*models.ChildProfile, error) {
	return databases.First(databases.Post[[]*models.ChildProfile](ctx, r.client, "/rest/v1/child_profiles", input, "return=representation", kidsSchema))
}

// Update patches a child profile's fields.
func (r *ChildProfileRepository) Update(ctx context.Context, id string, fields map[string]any) (*models.ChildProfile, error) {
	return databases.First(databases.Patch[[]*models.ChildProfile](ctx, r.client, "/rest/v1/child_profiles", databases.EqID(id), fields, "return=representation", kidsSchema))
}

// Delete removes a child profile.
func (r *ChildProfileRepository) Delete(ctx context.Context, id string) error {
	return databases.Delete(ctx, r.client, "/rest/v1/child_profiles", databases.EqID(id), kidsSchema)
}

// ── Wallet transactions ──────────────────────────────────────────────────────

// WalletTransactionRepository persists wallet ledger entries. Writes are
// only ever issued by WalletService under auth.WithServiceRole — see
// kids.sql's RLS comment for why there's no insert/update/delete policy at
// all for a normal caller.
type WalletTransactionRepository struct {
	client *databases.SupabaseClient
}

// NewWalletTransactionRepository constructs a WalletTransactionRepository.
func NewWalletTransactionRepository(client *databases.SupabaseClient) *WalletTransactionRepository {
	return &WalletTransactionRepository{client: client}
}

// ListByChildID returns a child's full ledger, newest first.
func (r *WalletTransactionRepository) ListByChildID(ctx context.Context, childID string) ([]*models.WalletTransaction, error) {
	return databases.Get[[]*models.WalletTransaction](ctx, r.client, "/rest/v1/wallet_transactions",
		url.Values{"child_id": []string{"eq." + childID}, "order": []string{"created_at.desc"}}, kidsSchema)
}

// ExistsForTask reports whether a ledger entry with this reason already
// exists for taskID — the idempotency check callers use before awarding or
// penalizing, so re-running Complete or the daily penalty job can't
// double-pay.
func (r *WalletTransactionRepository) ExistsForTask(ctx context.Context, taskID, reason string) (bool, error) {
	rows, err := databases.Get[[]*models.WalletTransaction](ctx, r.client, "/rest/v1/wallet_transactions",
		url.Values{"task_id": []string{"eq." + taskID}, "reason": []string{"eq." + reason}, "limit": []string{"1"}}, kidsSchema)
	if err != nil {
		return false, err
	}
	return len(rows) > 0, nil
}

// Create inserts a new ledger entry.
func (r *WalletTransactionRepository) Create(ctx context.Context, input models.WalletTransaction) (*models.WalletTransaction, error) {
	return databases.First(databases.Post[[]*models.WalletTransaction](ctx, r.client, "/rest/v1/wallet_transactions", input, "return=representation", kidsSchema))
}

// ── Shop items ────────────────────────────────────────────────────────────────

// ShopItemRepository persists parent-defined shop rewards.
type ShopItemRepository struct {
	client *databases.SupabaseClient
}

// NewShopItemRepository constructs a ShopItemRepository.
func NewShopItemRepository(client *databases.SupabaseClient) *ShopItemRepository {
	return &ShopItemRepository{client: client}
}

// List returns every shop item visible to the caller (their own, if a
// parent; their parent's active items, if a linked child — per RLS).
func (r *ShopItemRepository) List(ctx context.Context) ([]*models.ShopItem, error) {
	return databases.Get[[]*models.ShopItem](ctx, r.client, "/rest/v1/shop_items",
		url.Values{"order": []string{"coin_cost.asc"}}, kidsSchema)
}

// FindByID returns a shop item by id, or nil if not found/visible.
func (r *ShopItemRepository) FindByID(ctx context.Context, id string) (*models.ShopItem, error) {
	return databases.First(databases.Get[[]*models.ShopItem](ctx, r.client, "/rest/v1/shop_items",
		url.Values{"id": []string{"eq." + id}, "limit": []string{"1"}}, kidsSchema))
}

// Create inserts a new shop item.
func (r *ShopItemRepository) Create(ctx context.Context, input models.ShopItemInput) (*models.ShopItem, error) {
	return databases.First(databases.Post[[]*models.ShopItem](ctx, r.client, "/rest/v1/shop_items", input, "return=representation", kidsSchema))
}

// Update patches a shop item's fields.
func (r *ShopItemRepository) Update(ctx context.Context, id string, fields map[string]any) (*models.ShopItem, error) {
	return databases.First(databases.Patch[[]*models.ShopItem](ctx, r.client, "/rest/v1/shop_items", databases.EqID(id), fields, "return=representation", kidsSchema))
}

// Delete removes a shop item.
func (r *ShopItemRepository) Delete(ctx context.Context, id string) error {
	return databases.Delete(ctx, r.client, "/rest/v1/shop_items", databases.EqID(id), kidsSchema)
}

// ── Shop orders ───────────────────────────────────────────────────────────────

// ShopOrderRepository persists shop purchases. Inserts happen only via
// ShopService.Purchase under auth.WithServiceRole; FindByID/List work under
// the caller's own ctx (both parent and child have a select policy),
// Update (marking fulfilled) is parent-only under their own ctx.
type ShopOrderRepository struct {
	client *databases.SupabaseClient
}

// NewShopOrderRepository constructs a ShopOrderRepository.
func NewShopOrderRepository(client *databases.SupabaseClient) *ShopOrderRepository {
	return &ShopOrderRepository{client: client}
}

// ListByChildID returns a child's orders, newest first.
func (r *ShopOrderRepository) ListByChildID(ctx context.Context, childID string) ([]*models.ShopOrder, error) {
	return databases.Get[[]*models.ShopOrder](ctx, r.client, "/rest/v1/shop_orders",
		url.Values{"child_id": []string{"eq." + childID}, "order": []string{"created_at.desc"}}, kidsSchema)
}

// ListPending returns every pending order the caller (a parent) owns, for
// the "review and fulfill" list in Kids Settings.
func (r *ShopOrderRepository) ListPending(ctx context.Context) ([]*models.ShopOrder, error) {
	return databases.Get[[]*models.ShopOrder](ctx, r.client, "/rest/v1/shop_orders",
		url.Values{"status": []string{"eq.pending"}, "order": []string{"created_at.asc"}}, kidsSchema)
}

// FindByID returns a shop order by id, or nil if not found/visible.
func (r *ShopOrderRepository) FindByID(ctx context.Context, id string) (*models.ShopOrder, error) {
	return databases.First(databases.Get[[]*models.ShopOrder](ctx, r.client, "/rest/v1/shop_orders",
		url.Values{"id": []string{"eq." + id}, "limit": []string{"1"}}, kidsSchema))
}

// Create inserts a new shop order.
func (r *ShopOrderRepository) Create(ctx context.Context, order models.ShopOrder) (*models.ShopOrder, error) {
	return databases.First(databases.Post[[]*models.ShopOrder](ctx, r.client, "/rest/v1/shop_orders", order, "return=representation", kidsSchema))
}

// Update patches a shop order's fields (used to mark it fulfilled).
func (r *ShopOrderRepository) Update(ctx context.Context, id string, fields map[string]any) (*models.ShopOrder, error) {
	return databases.First(databases.Patch[[]*models.ShopOrder](ctx, r.client, "/rest/v1/shop_orders", databases.EqID(id), fields, "return=representation", kidsSchema))
}
