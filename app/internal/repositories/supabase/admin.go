package supabase

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"life-base-api/app/internal/databases"
	"life-base-api/app/internal/models"
)

// AdminRepository backs household/role administration.
type AdminRepository struct {
	client *databases.SupabaseClient
}

// NewAdminRepository constructs an AdminRepository.
func NewAdminRepository(client *databases.SupabaseClient) *AdminRepository {
	return &AdminRepository{client: client}
}

// authUser is the subset of Supabase GoTrue's admin user object we need.
type authUser struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

type listUsersResponse struct {
	Users []authUser `json:"users"`
}

// ListAuthEmails returns userID -> email for every Supabase auth user, via
// GoTrue's admin API. This always uses the service-role key directly
// (never the caller's own JWT) since it's a project-admin-only endpoint
// unrelated to PostgREST/RLS.
func (r *AdminRepository) ListAuthEmails(ctx context.Context) (map[string]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.client.BaseURL+"/auth/v1/admin/users", nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("apikey", r.client.APIKey)
	req.Header.Set("Authorization", "Bearer "+r.client.APIKey)

	resp, err := r.client.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("list auth users: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("list auth users: http %d", resp.StatusCode)
	}

	var out listUsersResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode auth users: %w", err)
	}

	emails := make(map[string]string, len(out.Users))
	for _, u := range out.Users {
		emails[u.ID] = u.Email
	}
	return emails, nil
}

// ListMembers returns every household_members row.
func (r *AdminRepository) ListMembers(ctx context.Context) ([]*models.HouseholdMember, error) {
	return databases.Get[[]*models.HouseholdMember](ctx, r.client, "/rest/v1/household_members",
		url.Values{"select": []string{"user_id,role"}})
}

// DefaultHouseholdID returns the single household row this app assumes for
// v1 (personal/family use — one household), creating it if none exists yet.
func (r *AdminRepository) DefaultHouseholdID(ctx context.Context) (string, error) {
	type household struct {
		ID string `json:"id"`
	}

	existing, err := databases.Get[[]household](ctx, r.client, "/rest/v1/households",
		url.Values{"select": []string{"id"}, "limit": []string{"1"}})
	if err != nil {
		return "", err
	}
	if len(existing) > 0 {
		return existing[0].ID, nil
	}

	created, err := databases.Post[[]household](ctx, r.client, "/rest/v1/households",
		map[string]string{"name": "Household"}, "return=representation")
	if err != nil {
		return "", err
	}
	if len(created) == 0 {
		return "", fmt.Errorf("create default household: empty response")
	}
	return created[0].ID, nil
}

// SetMemberRole upserts a household_members row for userID, creating the
// default household membership if the user has none yet.
func (r *AdminRepository) SetMemberRole(ctx context.Context, userID, role string) error {
	householdID, err := r.DefaultHouseholdID(ctx)
	if err != nil {
		return fmt.Errorf("resolve household: %w", err)
	}

	body := map[string]string{
		"household_id": householdID,
		"user_id":      userID,
		"role":         role,
	}
	_, err = databases.Post[[]map[string]any](ctx, r.client,
		"/rest/v1/household_members?on_conflict=user_id", body,
		"resolution=merge-duplicates,return=representation")
	return err
}

// ListPageAccess returns the full role/page access matrix.
func (r *AdminRepository) ListPageAccess(ctx context.Context) ([]*models.PageAccessEntry, error) {
	return databases.Get[[]*models.PageAccessEntry](ctx, r.client, "/rest/v1/page_access", nil)
}

// SetPageAccess upserts one (role, page_key) access entry.
func (r *AdminRepository) SetPageAccess(ctx context.Context, entry *models.PageAccessEntry) error {
	_, err := databases.Post[[]*models.PageAccessEntry](ctx, r.client,
		"/rest/v1/page_access?on_conflict=role,page_key", entry,
		"resolution=merge-duplicates,return=representation")
	return err
}
