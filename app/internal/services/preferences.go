package services

import (
	"context"
	"fmt"

	supabaserepo "life-base-api/app/internal/repositories/supabase"

	"life-base-api/app/internal/models"
)

type PreferencesService struct {
	repo *supabaserepo.PreferencesRepository
}

func NewPreferencesService(repo *supabaserepo.PreferencesRepository) *PreferencesService {
	return &PreferencesService{repo: repo}
}

// Get returns the caller's preferences, defaulting both toggles to false
// when no row exists yet (first visit to Settings).
func (s *PreferencesService) Get(ctx context.Context, userID string) (*models.UserPreferences, error) {
	prefs, err := s.repo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get preferences: %w", err)
	}
	if prefs == nil {
		return &models.UserPreferences{UserID: userID}, nil
	}
	return prefs, nil
}

func (s *PreferencesService) Set(ctx context.Context, userID string, pushEnabled, emailDigestEnabled bool) (*models.UserPreferences, error) {
	return s.repo.Upsert(ctx, &models.UserPreferences{
		UserID:             userID,
		PushEnabled:        pushEnabled,
		EmailDigestEnabled: emailDigestEnabled,
	})
}

func (s *PreferencesService) ListEnabledForPush(ctx context.Context) ([]*models.UserPreferences, error) {
	return s.repo.ListEnabledForPush(ctx)
}

func (s *PreferencesService) ListEnabledForEmailDigest(ctx context.Context) ([]*models.UserPreferences, error) {
	return s.repo.ListEnabledForEmailDigest(ctx)
}
