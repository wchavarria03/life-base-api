package services

import (
	"context"
	"sort"
	"sync"
	"time"

	supabaserepo "life-base-api/app/internal/repositories/supabase"

	"life-base-api/app/internal/models"
)

// AdminService manages household roles and the page access matrix.
type AdminService struct {
	repo *supabaserepo.AdminRepository

	mu    sync.Mutex
	cache map[string]cachedPageKeys
}

// pageKeyCacheTTL bounds how long a per-request feature check may lag behind
// an admin toggle; admin writes also clear the cache immediately.
const pageKeyCacheTTL = 30 * time.Second

type cachedPageKeys struct {
	keys    map[string]bool
	expires time.Time
}

// NewAdminService constructs an AdminService.
func NewAdminService(repo *supabaserepo.AdminRepository) *AdminService {
	return &AdminService{repo: repo, cache: make(map[string]cachedPageKeys)}
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
	defer s.invalidate()
	return s.repo.SetPageAccess(ctx, entry)
}

// ListUserPageAccess returns the per-account overrides for userID.
func (s *AdminService) ListUserPageAccess(ctx context.Context, userID string) ([]*models.UserPageAccessEntry, error) {
	return s.repo.ListUserPageAccess(ctx, userID)
}

// SetUserPageAccess forces one page on or off for one account. A nil allowed
// removes the override so the account follows its role default again.
func (s *AdminService) SetUserPageAccess(ctx context.Context, userID, pageKey string, allowed *bool) error {
	defer s.invalidate()
	if allowed == nil {
		return s.repo.DeleteUserPageAccess(ctx, userID, pageKey)
	}
	return s.repo.SetUserPageAccess(ctx, &models.UserPageAccessEntry{UserID: userID, PageKey: pageKey, Allowed: *allowed})
}

// AllowedPageKeys returns the page_key values the account may use: the role
// matrix, overridden per account by user_page_access.
func (s *AdminService) AllowedPageKeys(ctx context.Context, userID, role string) ([]string, error) {
	set, err := s.effectiveKeys(ctx, userID, role)
	if err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(set))
	for k := range set {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys, nil
}

// PageAllowedAny reports whether the account may use at least one of keys.
// It backs middleware.FeatureGate, so the API refuses disabled features
// even when called directly.
func (s *AdminService) PageAllowedAny(ctx context.Context, userID, role string, keys []string) (bool, error) {
	set, err := s.effectiveKeys(ctx, userID, role)
	if err != nil {
		return false, err
	}
	for _, k := range keys {
		if set[k] {
			return true, nil
		}
	}
	return false, nil
}

func (s *AdminService) effectiveKeys(ctx context.Context, userID, role string) (map[string]bool, error) {
	cacheKey := role + "|" + userID
	now := time.Now()
	s.mu.Lock()
	if c, ok := s.cache[cacheKey]; ok && now.Before(c.expires) {
		s.mu.Unlock()
		return c.keys, nil
	}
	s.mu.Unlock()

	entries, err := s.repo.ListPageAccess(ctx)
	if err != nil {
		return nil, err
	}
	set := make(map[string]bool, len(entries))
	for _, e := range entries {
		if e.Role == role && e.Allowed {
			set[e.PageKey] = true
		}
	}
	if userID != "" {
		overrides, err := s.repo.ListUserPageAccess(ctx, userID)
		if err != nil {
			return nil, err
		}
		for _, o := range overrides {
			if o.Allowed {
				set[o.PageKey] = true
			} else {
				delete(set, o.PageKey)
			}
		}
	}

	s.mu.Lock()
	s.cache[cacheKey] = cachedPageKeys{keys: set, expires: now.Add(pageKeyCacheTTL)}
	s.mu.Unlock()
	return set, nil
}

func (s *AdminService) invalidate() {
	s.mu.Lock()
	s.cache = make(map[string]cachedPageKeys)
	s.mu.Unlock()
}
