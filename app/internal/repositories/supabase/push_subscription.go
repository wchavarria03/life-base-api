package supabase

import (
	"context"
	"net/url"

	"life-base-api/app/internal/databases"
	"life-base-api/app/internal/models"
)

type PushSubscriptionRepository struct {
	client *databases.SupabaseClient
}

func NewPushSubscriptionRepository(client *databases.SupabaseClient) *PushSubscriptionRepository {
	return &PushSubscriptionRepository{client: client}
}

// Upsert registers or refreshes a subscription — endpoint is unique, so
// re-subscribing the same device replaces its keys instead of duplicating.
func (r *PushSubscriptionRepository) Upsert(ctx context.Context, s *models.PushSubscription) (*models.PushSubscription, error) {
	rows, err := databases.Post[[]*models.PushSubscription](ctx, r.client,
		"/rest/v1/push_subscriptions?on_conflict=endpoint", s,
		"resolution=merge-duplicates,return=representation")
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

func (r *PushSubscriptionRepository) DeleteByEndpoint(ctx context.Context, endpoint string) error {
	return databases.Delete(ctx, r.client, "/rest/v1/push_subscriptions",
		url.Values{"endpoint": []string{"eq." + endpoint}})
}

func (r *PushSubscriptionRepository) ListByUserID(ctx context.Context, userID string) ([]*models.PushSubscription, error) {
	return databases.Get[[]*models.PushSubscription](ctx, r.client, "/rest/v1/push_subscriptions",
		url.Values{"user_id": []string{"eq." + userID}})
}

// Delete removes a subscription by id — used to drop one that a push send
// reports as gone (410/404 from the push service).
func (r *PushSubscriptionRepository) Delete(ctx context.Context, id string) error {
	return databases.Delete(ctx, r.client, "/rest/v1/push_subscriptions", databases.EqID(id))
}
