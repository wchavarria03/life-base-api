package services

import (
	"context"

	supabaserepo "life-base-api/app/internal/repositories/supabase"

	"life-base-api/app/internal/models"
)

// PushService manages web push subscriptions.
type PushService struct {
	repo *supabaserepo.PushSubscriptionRepository
}

// NewPushService constructs a PushService.
func NewPushService(repo *supabaserepo.PushSubscriptionRepository) *PushService {
	return &PushService{repo: repo}
}

// Subscribe registers or refreshes a push subscription for a user.
func (s *PushService) Subscribe(ctx context.Context, userID, endpoint, p256dh, authKey string) error {
	_, err := s.repo.Upsert(ctx, &models.PushSubscription{
		UserID:   userID,
		Endpoint: endpoint,
		P256dh:   p256dh,
		AuthKey:  authKey,
	})
	return err
}

// Unsubscribe removes a push subscription by endpoint.
func (s *PushService) Unsubscribe(ctx context.Context, endpoint string) error {
	return s.repo.DeleteByEndpoint(ctx, endpoint)
}

// ListByUserID returns every push subscription for a user.
func (s *PushService) ListByUserID(ctx context.Context, userID string) ([]*models.PushSubscription, error) {
	return s.repo.ListByUserID(ctx, userID)
}

// Delete removes a push subscription by id.
func (s *PushService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}
