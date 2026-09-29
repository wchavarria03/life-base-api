package services

import (
	"context"
	"fmt"

	supabaserepo "life-base-api/app/internal/repositories/supabase"

	"life-base-api/app/internal/models"
)

// MenuPreferencesService manages per-user hidden-nav-page preferences.
type MenuPreferencesService struct {
	repo *supabaserepo.MenuPreferencesRepository
}

// NewMenuPreferencesService constructs a MenuPreferencesService.
func NewMenuPreferencesService(repo *supabaserepo.MenuPreferencesRepository) *MenuPreferencesService {
	return &MenuPreferencesService{repo: repo}
}

// Get returns the page_keys the caller has hidden from their own nav,
// defaulting to an empty list (nothing hidden) when no rows exist yet.
func (s *MenuPreferencesService) Get(ctx context.Context, userID string) ([]string, error) {
	rows, err := s.repo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get menu preferences: %w", err)
	}
	hidden := make([]string, 0, len(rows))
	for _, row := range rows {
		if row.Hidden {
			hidden = append(hidden, row.PageKey)
		}
	}
	return hidden, nil
}

// Set upserts whether one page_key is hidden from the caller's own nav. The
// "settings" page can never be hidden, so a caller must always be able to
// get back in to re-enable something they hid.
func (s *MenuPreferencesService) Set(ctx context.Context, userID, pageKey string, hidden bool) ([]string, error) {
	if pageKey == "settings" {
		return nil, fmt.Errorf("the settings page cannot be hidden")
	}
	_, err := s.repo.Upsert(ctx, &models.UserMenuPreference{
		UserID:  userID,
		PageKey: pageKey,
		Hidden:  hidden,
	})
	if err != nil {
		return nil, fmt.Errorf("set menu preference: %w", err)
	}
	return s.Get(ctx, userID)
}
