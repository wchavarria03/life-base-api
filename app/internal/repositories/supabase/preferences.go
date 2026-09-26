package supabase

import (
	"context"
	"net/url"

	"life-base-api/app/internal/databases"
	"life-base-api/app/internal/models"
)

type PreferencesRepository struct {
	client *databases.SupabaseClient
}

func NewPreferencesRepository(client *databases.SupabaseClient) *PreferencesRepository {
	return &PreferencesRepository{client: client}
}

func (r *PreferencesRepository) FindByUserID(ctx context.Context, userID string) (*models.UserPreferences, error) {
	rows, err := databases.Get[[]*models.UserPreferences](ctx, r.client, "/rest/v1/user_preferences",
		url.Values{"user_id": []string{"eq." + userID}, "limit": []string{"1"}})
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

func (r *PreferencesRepository) Upsert(ctx context.Context, p *models.UserPreferences) (*models.UserPreferences, error) {
	rows, err := databases.Post[[]*models.UserPreferences](ctx, r.client,
		"/rest/v1/user_preferences?on_conflict=user_id", p,
		"resolution=merge-duplicates,return=representation")
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

// ListEnabledForPush returns every user_id with push_enabled = true.
func (r *PreferencesRepository) ListEnabledForPush(ctx context.Context) ([]*models.UserPreferences, error) {
	return databases.Get[[]*models.UserPreferences](ctx, r.client, "/rest/v1/user_preferences",
		url.Values{"push_enabled": []string{"eq.true"}})
}

// ListEnabledForEmailDigest returns every user_id with email_digest_enabled = true.
func (r *PreferencesRepository) ListEnabledForEmailDigest(ctx context.Context) ([]*models.UserPreferences, error) {
	return databases.Get[[]*models.UserPreferences](ctx, r.client, "/rest/v1/user_preferences",
		url.Values{"email_digest_enabled": []string{"eq.true"}})
}
