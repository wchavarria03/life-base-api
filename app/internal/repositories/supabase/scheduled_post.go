package supabase

import (
	"context"
	"net/url"
	"time"

	"life-base-api/app/internal/databases"
	"life-base-api/app/internal/models"
)

// ScheduledPostRepository persists scheduled social posts.
type ScheduledPostRepository struct {
	client *databases.SupabaseClient
}

// NewScheduledPostRepository constructs a ScheduledPostRepository.
func NewScheduledPostRepository(client *databases.SupabaseClient) *ScheduledPostRepository {
	return &ScheduledPostRepository{client: client}
}

// List returns the caller's scheduled posts, soonest first.
func (r *ScheduledPostRepository) List(ctx context.Context) ([]*models.ScheduledPost, error) {
	return databases.Get[[]*models.ScheduledPost](ctx, r.client, "/rest/v1/scheduled_posts",
		url.Values{"order": []string{"scheduled_at.asc"}})
}

// FindByID returns a scheduled post by id, or nil if not found.
func (r *ScheduledPostRepository) FindByID(ctx context.Context, id string) (*models.ScheduledPost, error) {
	rows, err := databases.Get[[]*models.ScheduledPost](ctx, r.client, "/rest/v1/scheduled_posts",
		url.Values{"id": []string{"eq." + id}, "limit": []string{"1"}})
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

// Create inserts a new scheduled post.
func (r *ScheduledPostRepository) Create(ctx context.Context, input models.ScheduledPostInput) (*models.ScheduledPost, error) {
	rows, err := databases.Post[[]*models.ScheduledPost](ctx, r.client, "/rest/v1/scheduled_posts", input, "return=representation")
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

// Update patches a scheduled post's fields.
func (r *ScheduledPostRepository) Update(ctx context.Context, id string, fields map[string]any) error {
	fields["updated_at"] = time.Now().UTC().Format(time.RFC3339)
	_, err := databases.Patch[[]*models.ScheduledPost](ctx, r.client, "/rest/v1/scheduled_posts",
		databases.EqID(id), fields, "")
	return err
}

// Delete removes a scheduled post.
func (r *ScheduledPostRepository) Delete(ctx context.Context, id string) error {
	return databases.Delete(ctx, r.client, "/rest/v1/scheduled_posts", databases.EqID(id))
}

// ListDue returns every pending scheduled post whose scheduled_at has
// passed, across all users — used by the cron sender (service-role key,
// bypasses RLS by design). When userID is non-empty, scopes to that user
// only — used by the manual "check now" button.
func (r *ScheduledPostRepository) ListDue(ctx context.Context, userID string) ([]*models.ScheduledPost, error) {
	params := url.Values{
		"status":       []string{"eq." + string(models.ScheduledPostPending)},
		"scheduled_at": []string{"lte." + time.Now().UTC().Format(time.RFC3339)},
	}
	if userID != "" {
		params.Set("user_id", "eq."+userID)
	}
	return databases.Get[[]*models.ScheduledPost](ctx, r.client, "/rest/v1/scheduled_posts", params)
}
