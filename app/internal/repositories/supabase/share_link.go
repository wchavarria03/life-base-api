package supabase

import (
	"context"
	"net/url"

	"life-base-api/app/internal/databases"
	"life-base-api/app/internal/models"
)

// ShareLinkRepository backs both the owner-facing /v1/share-links routes
// and the public, unauthenticated /public/shared/:token resolver.
type ShareLinkRepository struct {
	client *databases.SupabaseClient
}

// NewShareLinkRepository constructs a ShareLinkRepository.
func NewShareLinkRepository(client *databases.SupabaseClient) *ShareLinkRepository {
	return &ShareLinkRepository{client: client}
}

// List returns the caller's own share links (owner-scoped by RLS — always
// runs with a user JWT).
func (r *ShareLinkRepository) List(ctx context.Context) ([]*models.ShareLink, error) {
	return databases.Get[[]*models.ShareLink](ctx, r.client, "/rest/v1/share_links",
		url.Values{"order": []string{"created_at.desc"}})
}

// Create inserts a new share link row.
func (r *ShareLinkRepository) Create(ctx context.Context, input models.ShareLinkInput) (*models.ShareLink, error) {
	return databases.First(databases.Post[[]*models.ShareLink](ctx, r.client, "/rest/v1/share_links", input, "return=representation"))
}

// FindByToken looks up a share link by its raw token. Called from the
// public /public/shared/:token route, which carries no user JWT — this
// always runs with the service-role key (see databases.resolveKeys), so it
// deliberately bypasses RLS; the token itself is the access check. Safe
// because resource_id was only ever stored after an RLS-scoped ownership
// check at creation time, never taken from an unauthenticated caller.
func (r *ShareLinkRepository) FindByToken(ctx context.Context, token string) (*models.ShareLink, error) {
	return databases.First(databases.Get[[]*models.ShareLink](ctx, r.client, "/rest/v1/share_links", url.Values{
		"token": []string{"eq." + token},
		"limit": []string{"1"},
	}))
}

// Update patches arbitrary fields (owner-scoped by RLS when called with a
// user JWT — revoke; unscoped via service-role when called from the public
// resolver — view-count/last-viewed bookkeeping, which is why it isn't
// folded into a narrower "Revoke" method).
func (r *ShareLinkRepository) Update(ctx context.Context, id string, fields map[string]any) error {
	_, err := databases.Patch[[]*models.ShareLink](ctx, r.client, "/rest/v1/share_links",
		databases.EqID(id), fields, "return=representation")
	return err
}
