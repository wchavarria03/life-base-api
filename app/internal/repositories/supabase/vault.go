package supabase

import (
	"context"
	"net/url"

	"life-base-api/app/internal/databases"
	"life-base-api/app/internal/models"
)

// VaultRepository backs vault sync: the internal worker endpoints (called
// without a user JWT, so they run with the service-role key and must always
// scope by user_id explicitly) and the user-facing backlinks/search/sync
// calls (called with the user's JWT, so RLS applies).
type VaultRepository struct {
	client *databases.SupabaseClient
}

// NewVaultRepository constructs a VaultRepository.
func NewVaultRepository(client *databases.SupabaseClient) *VaultRepository {
	return &VaultRepository{client: client}
}

// FindByVaultPath returns the note for a vault path, including soft-deleted
// ones. userID may be empty only when running under a user JWT (RLS scopes).
func (r *VaultRepository) FindByVaultPath(ctx context.Context, userID, vaultPath string) (*models.Note, error) {
	params := url.Values{
		"vault_path": []string{"eq." + vaultPath},
		"limit":      []string{"1"},
	}
	if userID != "" {
		params.Set("user_id", "eq."+userID)
	}
	return databases.First(databases.Get[[]*models.Note](ctx, r.client, "/rest/v1/notes", params, notesSchema))
}

// FindOwned returns a note by id, scoped to userID (service-role callers).
func (r *VaultRepository) FindOwned(ctx context.Context, userID, id string) (*models.Note, error) {
	return databases.First(databases.Get[[]*models.Note](ctx, r.client, "/rest/v1/notes", url.Values{
		"id":      []string{"eq." + id},
		"user_id": []string{"eq." + userID},
		"limit":   []string{"1"},
	}, notesSchema))
}

// FindByOrgID returns a live note by its org-roam id. userID may be empty
// under a user JWT.
func (r *VaultRepository) FindByOrgID(ctx context.Context, userID, orgID string) (*models.NoteRef, error) {
	params := url.Values{
		"select":     []string{"id,title"},
		"org_id":     []string{"eq." + orgID},
		"deleted_at": []string{"is.null"},
		"limit":      []string{"1"},
	}
	if userID != "" {
		params.Set("user_id", "eq."+userID)
	}
	return databases.First(databases.Get[[]*models.NoteRef](ctx, r.client, "/rest/v1/notes", params, notesSchema))
}

// CreateNote inserts a note row (row must include user_id under service role).
func (r *VaultRepository) CreateNote(ctx context.Context, row map[string]any) (*models.Note, error) {
	return databases.First(databases.Post[[]*models.Note](ctx, r.client, "/rest/v1/notes", row, "return=representation", notesSchema))
}

// UpdateNote patches a note row by id.
func (r *VaultRepository) UpdateNote(ctx context.Context, id string, fields map[string]any) (*models.Note, error) {
	return databases.First(databases.Patch[[]*models.Note](ctx, r.client, "/rest/v1/notes", databases.EqID(id), fields, "return=representation", notesSchema))
}

// ListPending returns the user's notes waiting to be written to git:
// pending_write, not in conflict, not deleted, with a vault path.
func (r *VaultRepository) ListPending(ctx context.Context, userID string) ([]*models.Note, error) {
	return databases.Get[[]*models.Note](ctx, r.client, "/rest/v1/notes", url.Values{
		"user_id":       []string{"eq." + userID},
		"pending_write": []string{"eq.true"},
		"conflict":      []string{"eq.false"},
		"deleted_at":    []string{"is.null"},
		"vault_path":    []string{"not.is.null"},
		"order":         []string{"updated_at.asc"},
		"limit":         []string{"200"},
	}, notesSchema)
}

// ReplaceLinks deletes a note's outgoing links and inserts the new set.
func (r *VaultRepository) ReplaceLinks(ctx context.Context, noteID string, rows []models.NoteLinkRow) error {
	if err := databases.Delete(ctx, r.client, "/rest/v1/note_links", url.Values{
		"from_note_id": []string{"eq." + noteID},
	}, notesSchema); err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}
	_, err := databases.Post[[]*models.NoteLinkRow](ctx, r.client, "/rest/v1/note_links", rows, "return=minimal", notesSchema)
	return err
}

// ResolveLinksTo points every still-unresolved link with this org id at noteID.
func (r *VaultRepository) ResolveLinksTo(ctx context.Context, orgID, noteID string) error {
	_, err := databases.Patch[[]*models.NoteLinkRow](ctx, r.client, "/rest/v1/note_links", url.Values{
		"to_org_id":  []string{"eq." + orgID},
		"to_note_id": []string{"is.null"},
	}, map[string]any{"to_note_id": noteID}, "return=minimal", notesSchema)
	return err
}

// ListBacklinks returns links pointing at noteID (resolved) or at orgID
// (unresolved), with the source note embedded. orgID must be pre-validated
// (it is interpolated into an `or` filter); so must noteID.
func (r *VaultRepository) ListBacklinks(ctx context.Context, noteID, orgID string) ([]*models.BacklinkRow, error) {
	params := url.Values{
		"select": []string{"from_note_id,label,notes!from_note_id(id,title,sensitive)"},
		"limit":  []string{"200"},
	}
	if orgID != "" {
		params.Set("or", "(to_note_id.eq."+noteID+",to_org_id.eq."+orgID+")")
	} else {
		params.Set("to_note_id", "eq."+noteID)
	}
	return databases.Get[[]*models.BacklinkRow](ctx, r.client, "/rest/v1/note_links", params, notesSchema)
}

// Search runs notes.search_notes under the caller's JWT.
func (r *VaultRepository) Search(ctx context.Context, q string, limit int) ([]*models.NoteSearchHit, error) {
	return databases.Post[[]*models.NoteSearchHit](ctx, r.client, "/rest/v1/rpc/search_notes",
		map[string]any{"q": q, "lim": limit}, "", notesSchema)
}

// GetSyncState returns the user's sync row, or nil.
func (r *VaultRepository) GetSyncState(ctx context.Context, userID string) (*models.VaultSyncState, error) {
	params := url.Values{"limit": []string{"1"}}
	if userID != "" {
		params.Set("user_id", "eq."+userID)
	}
	return databases.First(databases.Get[[]*models.VaultSyncState](ctx, r.client, "/rest/v1/vault_sync_state", params, notesSchema))
}

// UpsertSyncState merges fields (must include user_id) into the user's row.
func (r *VaultRepository) UpsertSyncState(ctx context.Context, row map[string]any) (*models.VaultSyncState, error) {
	return databases.First(databases.Post[[]*models.VaultSyncState](ctx, r.client, "/rest/v1/vault_sync_state",
		row, "resolution=merge-duplicates,return=representation", notesSchema))
}
