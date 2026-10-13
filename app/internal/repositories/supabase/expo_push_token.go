//nolint:dupl // structurally mirrors PushSubscriptionRepository; different table/shape, not worth a generic for 5 methods
package supabase

import (
	"context"
	"net/url"

	"life-base-api/app/internal/databases"
	"life-base-api/app/internal/models"
)

// ExpoPushTokenRepository persists mobile (Expo) push tokens.
type ExpoPushTokenRepository struct {
	client *databases.SupabaseClient
}

// NewExpoPushTokenRepository constructs an ExpoPushTokenRepository.
func NewExpoPushTokenRepository(client *databases.SupabaseClient) *ExpoPushTokenRepository {
	return &ExpoPushTokenRepository{client: client}
}

// Upsert registers or refreshes a token — token is unique, so re-registering
// the same device replaces its platform instead of duplicating.
func (r *ExpoPushTokenRepository) Upsert(ctx context.Context, t *models.ExpoPushToken) (*models.ExpoPushToken, error) {
	return databases.First(databases.Post[[]*models.ExpoPushToken](ctx, r.client,
		"/rest/v1/expo_push_tokens?on_conflict=token", t,
		"resolution=merge-duplicates,return=representation"))
}

// DeleteByToken removes a token by its value.
func (r *ExpoPushTokenRepository) DeleteByToken(ctx context.Context, token string) error {
	return databases.Delete(ctx, r.client, "/rest/v1/expo_push_tokens",
		url.Values{"token": []string{"eq." + token}})
}

// ListByUserID returns every token for a user.
func (r *ExpoPushTokenRepository) ListByUserID(ctx context.Context, userID string) ([]*models.ExpoPushToken, error) {
	return databases.Get[[]*models.ExpoPushToken](ctx, r.client, "/rest/v1/expo_push_tokens",
		url.Values{"user_id": []string{"eq." + userID}})
}

// Delete removes a token by id — used to drop one Expo reports as no
// longer registered.
func (r *ExpoPushTokenRepository) Delete(ctx context.Context, id string) error {
	return databases.Delete(ctx, r.client, "/rest/v1/expo_push_tokens", databases.EqID(id))
}
