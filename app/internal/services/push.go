package services

import (
	"context"

	supabaserepo "life-base-api/app/internal/repositories/supabase"

	"life-base-api/app/internal/models"
)

type PushService struct {
	repo *supabaserepo.PushSubscriptionRepository
}

func NewPushService(repo *supabaserepo.PushSubscriptionRepository) *PushService {
	return &PushService{repo: repo}
}

func (s *PushService) Subscribe(ctx context.Context, userID, endpoint, p256dh, authKey string) error {
	_, err := s.repo.Upsert(ctx, &models.PushSubscription{
		UserID:   userID,
		Endpoint: endpoint,
		P256dh:   p256dh,
		AuthKey:  authKey,
	})
	return err
}

func (s *PushService) Unsubscribe(ctx context.Context, endpoint string) error {
	return s.repo.DeleteByEndpoint(ctx, endpoint)
}

func (s *PushService) ListByUserID(ctx context.Context, userID string) ([]*models.PushSubscription, error) {
	return s.repo.ListByUserID(ctx, userID)
}

func (s *PushService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}
