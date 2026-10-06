package supabase

import (
	"context"
	"net/url"

	"life-base-api/app/internal/databases"
	"life-base-api/app/internal/models"
)

// MenuPreferencesRepository persists per-user hidden-nav-page preferences.
type MenuPreferencesRepository struct {
	client *databases.SupabaseClient
}

// NewMenuPreferencesRepository constructs a MenuPreferencesRepository.
func NewMenuPreferencesRepository(client *databases.SupabaseClient) *MenuPreferencesRepository {
	return &MenuPreferencesRepository{client: client}
}

// ListByUserID returns every menu preference row for a user.
func (r *MenuPreferencesRepository) ListByUserID(ctx context.Context, userID string) ([]*models.UserMenuPreference, error) {
	return databases.Get[[]*models.UserMenuPreference](ctx, r.client, "/rest/v1/user_menu_preferences",
		url.Values{"user_id": []string{"eq." + userID}})
}

// Upsert creates or replaces a user's preference row for one page_key.
func (r *MenuPreferencesRepository) Upsert(ctx context.Context, p *models.UserMenuPreference) (*models.UserMenuPreference, error) {
	return databases.First(databases.Post[[]*models.UserMenuPreference](ctx, r.client,
		"/rest/v1/user_menu_preferences?on_conflict=user_id,page_key", p,
		"resolution=merge-duplicates,return=representation"))
}
