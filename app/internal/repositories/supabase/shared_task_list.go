package supabase

import (
	"context"
	"net/url"

	"life-base-api/app/internal/databases"
	"life-base-api/app/internal/models"
)

// SharedTaskListRepository backs public/unauthenticated task-list share
// links (public schema — no schema profile needed).
type SharedTaskListRepository struct {
	client *databases.SupabaseClient
}

// NewSharedTaskListRepository constructs a SharedTaskListRepository.
func NewSharedTaskListRepository(client *databases.SupabaseClient) *SharedTaskListRepository {
	return &SharedTaskListRepository{client: client}
}

// List returns the caller's own share rows (owner-scoped by RLS — this
// always runs with a user JWT, the authenticated /v1/shared-lists route).
func (r *SharedTaskListRepository) List(ctx context.Context) ([]*models.SharedTaskList, error) {
	return databases.Get[[]*models.SharedTaskList](ctx, r.client, "/rest/v1/shared_task_lists",
		url.Values{"order": []string{"created_at.desc"}})
}

// Create inserts a new share row.
func (r *SharedTaskListRepository) Create(ctx context.Context, input models.SharedTaskListInput) (*models.SharedTaskList, error) {
	return databases.First(databases.Post[[]*models.SharedTaskList](ctx, r.client, "/rest/v1/shared_task_lists", input, "return=representation"))
}

// FindByToken looks up a share row by its raw token. Called from the public
// /public/shared/:token routes, which carry no user JWT — this always runs
// with the service-role key (see databases.resolveKeys), so it deliberately
// bypasses RLS; the token itself is the access check.
func (r *SharedTaskListRepository) FindByToken(ctx context.Context, token string) (*models.SharedTaskList, error) {
	return databases.First(databases.Get[[]*models.SharedTaskList](ctx, r.client, "/rest/v1/shared_task_lists", url.Values{
		"token": []string{"eq." + token},
		"limit": []string{"1"},
	}))
}

// Revoke soft-deletes a share row by id (owner-scoped by RLS).
func (r *SharedTaskListRepository) Revoke(ctx context.Context, id string) error {
	_, err := databases.Patch[[]*models.SharedTaskList](ctx, r.client, "/rest/v1/shared_task_lists",
		databases.EqID(id), map[string]any{"revoked": true}, "return=representation")
	return err
}
