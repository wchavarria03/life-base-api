package services

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"life-base-api/app/internal/auth"
	"life-base-api/app/internal/models"
)

// SharedTaskListRepository is the storage interface SharedTaskListService
// needs.
type SharedTaskListRepository interface {
	List(ctx context.Context) ([]*models.SharedTaskList, error)
	Create(ctx context.Context, input models.SharedTaskListInput) (*models.SharedTaskList, error)
	FindByToken(ctx context.Context, token string) (*models.SharedTaskList, error)
	Revoke(ctx context.Context, id string) error
}

// SharedTaskListConfig holds the Resend email settings (reusing the same
// account DigestConfig sends the email digest through) and the deployed
// frontend's base URL, used to build the "{FRONTEND_URL}/shared/{token}"
// link emailed to the invitee.
type SharedTaskListConfig struct {
	ResendAPIKey string
	ResendFrom   string
	FrontendURL  string // e.g. "https://life-base.example.com", no trailing slash
}

// SharedTaskListService creates, lists, and revokes tokenized public share
// links for a task list, and serves the public (unauthenticated) view/complete
// actions those links grant.
type SharedTaskListService struct {
	shares SharedTaskListRepository
	tasks  *TaskService
	cfg    SharedTaskListConfig
	http   *http.Client
}

// NewSharedTaskListService constructs a SharedTaskListService.
func NewSharedTaskListService(shares SharedTaskListRepository, tasks *TaskService, cfg SharedTaskListConfig) *SharedTaskListService {
	return &SharedTaskListService{shares: shares, tasks: tasks, cfg: cfg, http: &http.Client{Timeout: 15 * time.Second}}
}

// List returns the caller's own shares. Never includes the raw token —
// Token is only ever returned by Create, right after the link is minted.
func (s *SharedTaskListService) List(ctx context.Context) ([]*models.SharedTaskList, error) {
	shares, err := s.shares.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list shares: %w", err)
	}
	for _, sh := range shares {
		sh.Token = ""
	}
	return shares, nil
}

// Create mints a new token, stores it, and emails the link to "email" via
// Resend. Returns the row with its raw token populated — the only time it's
// ever exposed after creation.
func (s *SharedTaskListService) Create(ctx context.Context, category models.TaskCategory, email string) (*models.SharedTaskList, error) {
	userID := auth.UserIDFromContext(ctx)
	if userID == "" {
		return nil, fmt.Errorf("no authenticated user")
	}
	switch category {
	case models.TaskHousehold, models.TaskHouse, models.TaskTodo:
	default:
		return nil, fmt.Errorf("invalid category: %s", category)
	}
	if email == "" {
		return nil, fmt.Errorf("email is required")
	}

	token, err := generateShareToken()
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	share, err := s.shares.Create(ctx, models.SharedTaskListInput{
		UserID:   userID,
		Category: category,
		Token:    token,
	})
	if err != nil {
		return nil, fmt.Errorf("create share: %w", err)
	}
	if share == nil {
		return nil, fmt.Errorf("create share: empty response")
	}

	if err := s.sendInviteEmail(ctx, email, share); err != nil {
		// Best-effort: the link is created and usable even if the email
		// fails to send (e.g. Resend not configured in dev) — the owner can
		// still copy it manually. Surface the error so the caller knows.
		return share, fmt.Errorf("share created but invite email failed: %w", err)
	}
	return share, nil
}

// Revoke soft-deletes a share row (RLS scopes this to the caller's own rows).
func (s *SharedTaskListService) Revoke(ctx context.Context, id string) error {
	return s.shares.Revoke(ctx, id)
}

// ResolveToken validates a public share token and returns the row it
// points at, or nil if the token doesn't exist or has been revoked.
func (s *SharedTaskListService) ResolveToken(ctx context.Context, token string) (*models.SharedTaskList, error) {
	share, err := s.shares.FindByToken(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("find share: %w", err)
	}
	if share == nil || share.Revoked {
		return nil, nil
	}
	return share, nil
}

// PublicListTasks returns the shared owner's tasks for the share's category —
// the payload behind GET /public/shared/:token.
func (s *SharedTaskListService) PublicListTasks(ctx context.Context, token string) ([]models.TaskWithStatus, error) {
	share, err := s.ResolveToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if share == nil {
		return nil, fmt.Errorf("share not found")
	}
	return s.tasks.ListByUserAndCategory(ctx, share.UserID, share.Category)
}

// PublicCompleteTask marks taskID complete through a share link, after
// verifying taskID actually belongs to the share's owner and category —
// otherwise a guessed task id from a different owner/category could be
// completed through someone else's link.
func (s *SharedTaskListService) PublicCompleteTask(ctx context.Context, token, taskID string) (*models.Task, error) {
	share, err := s.ResolveToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if share == nil {
		return nil, fmt.Errorf("share not found")
	}

	task, err := s.tasks.FindByID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("find task: %w", err)
	}
	if task == nil || task.UserID != share.UserID || task.Category != share.Category {
		return nil, fmt.Errorf("task not found")
	}

	return s.tasks.Complete(ctx, taskID)
}

func (s *SharedTaskListService) sendInviteEmail(ctx context.Context, to string, share *models.SharedTaskList) error {
	if s.cfg.ResendAPIKey == "" || s.cfg.ResendFrom == "" {
		return fmt.Errorf("resend not configured")
	}
	if s.cfg.FrontendURL == "" {
		return fmt.Errorf("frontend URL not configured")
	}

	link := fmt.Sprintf("%s/shared/%s", s.cfg.FrontendURL, share.Token)
	body := fmt.Sprintf("You've been invited to view and check off a %s task list: %s", share.Category, link)
	payload, err := json.Marshal(map[string]any{
		"from":    s.cfg.ResendFrom,
		"to":      []string{to},
		"subject": "Life-Base: a task list was shared with you",
		"text":    body,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, resendAPIURL, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+s.cfg.ResendAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.http.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("resend: http %d", resp.StatusCode)
	}
	return nil
}

// generateShareToken mints a long, url-safe, opaque token to serve as the
// bearer credential for a public share link — treat it like a secret
// (never logged, never re-displayed after creation; see
// SharedTaskListService.List).
func generateShareToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
