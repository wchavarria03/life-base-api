package services

import (
	"context"

	supabaserepo "life-base-api/app/internal/repositories/supabase"

	"life-base-api/app/internal/models"
)

// AdminService manages household roles and the page access matrix.
type AdminService struct {
	repo *supabaserepo.AdminRepository
}

// NewAdminService constructs an AdminService.
func NewAdminService(repo *supabaserepo.AdminRepository) *AdminService {
	return &AdminService{repo: repo}
}

// ListMembers returns every household member, with email filled in from
// Supabase auth for display.
func (s *AdminService) ListMembers(ctx context.Context) ([]*models.HouseholdMember, error) {
	members, err := s.repo.ListMembers(ctx)
	if err != nil {
		return nil, err
	}

	emails, err := s.repo.ListAuthEmails(ctx)
	if err != nil {
		return nil, err
	}
	for _, m := range members {
		m.Email = emails[m.UserID]
	}
	return members, nil
}

// SetMemberRole assigns a role to a household member.
func (s *AdminService) SetMemberRole(ctx context.Context, userID, role string) error {
	return s.repo.SetMemberRole(ctx, userID, role)
}

// ListPageAccess returns the full role/page access matrix.
func (s *AdminService) ListPageAccess(ctx context.Context) ([]*models.PageAccessEntry, error) {
	return s.repo.ListPageAccess(ctx)
}

// SetPageAccess upserts one (role, page_key) access entry.
func (s *AdminService) SetPageAccess(ctx context.Context, entry *models.PageAccessEntry) error {
	return s.repo.SetPageAccess(ctx, entry)
}

// AllowedPageKeys returns the page_key values allowed for role.
func (s *AdminService) AllowedPageKeys(ctx context.Context, role string) ([]string, error) {
	entries, err := s.repo.ListPageAccess(ctx)
	if err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.Role == role && e.Allowed {
			keys = append(keys, e.PageKey)
		}
	}
	return keys, nil
}
