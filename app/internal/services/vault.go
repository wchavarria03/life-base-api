package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"life-base-api/app/internal/auth"
	"life-base-api/app/internal/models"
	supabaserepo "life-base-api/app/internal/repositories/supabase"
	"life-base-api/app/internal/vault"
)

const (
	vaultMaxUpsertItems  = 100
	vaultMaxContentBytes = 1 << 20 // 1 MiB per note
	vaultMaxSearchLen    = 200
	// vaultConflictPrefix marks the archived incoming vault version of a
	// conflicted note in notes.note_versions.
	vaultConflictPrefix = "[vault] "
)

var (
	// ErrVaultDisabled means VAULT_SYNC_USER_ID is not configured.
	ErrVaultDisabled = errors.New("vault sync is not configured")
	// ErrSensitiveDeclared is returned when the app tries to un-flag a note
	// whose file declares itself sensitive.
	ErrSensitiveDeclared = errors.New("sensitive flag is declared inside the file; remove it there first")

	commitRe = regexp.MustCompile(`^[0-9a-f]{7,64}$`)
	uuidRe   = regexp.MustCompile(`^[0-9a-fA-F-]{8,40}$`)
)

// VaultService implements the git <-> app sync described in
// docs/NOTES_SYNC.md. Internal (worker) methods run without a user JWT, i.e.
// with the service-role key, scoped to the single configured vault user.
type VaultService struct {
	repo     *supabaserepo.VaultRepository
	versions *supabaserepo.NoteVersionRepository
	notes    *NoteService
	userID   string // VAULT_SYNC_USER_ID — the vault is single-user
}

// NewVaultService constructs a VaultService. userID may be empty, which
// disables the internal endpoints.
func NewVaultService(repo *supabaserepo.VaultRepository, versions *supabaserepo.NoteVersionRepository, notes *NoteService, userID string) *VaultService {
	return &VaultService{repo: repo, versions: versions, notes: notes, userID: strings.TrimSpace(userID)}
}

func (s *VaultService) nextVersion(ctx context.Context, noteID string) (int, error) {
	existing, err := s.versions.ListByNote(ctx, noteID)
	if err != nil {
		return 0, fmt.Errorf("list note versions: %w", err)
	}
	if len(existing) > 0 {
		return existing[0].VersionNumber + 1, nil
	}
	return 1, nil
}

func (s *VaultService) archive(ctx context.Context, noteID, title string, content *string) error {
	n, err := s.nextVersion(ctx, noteID)
	if err != nil {
		return err
	}
	_, err = s.versions.Create(ctx, &models.NoteVersion{NoteID: noteID, Title: title, Content: content, VersionNumber: n})
	return err
}

// ---------------------------------------------------------------------
// Internal (worker) operations
// ---------------------------------------------------------------------

// Upsert applies a batch of file changes from the worker. Per-item failures
// are reported in the result slice rather than failing the batch.
func (s *VaultService) Upsert(ctx context.Context, req models.VaultUpsertRequest) ([]models.VaultUpsertResult, error) {
	if s.userID == "" {
		return nil, ErrVaultDisabled
	}
	if len(req.Items) > vaultMaxUpsertItems {
		return nil, fmt.Errorf("too many items (max %d)", vaultMaxUpsertItems)
	}
	results := make([]models.VaultUpsertResult, 0, len(req.Items))
	for _, item := range req.Items {
		res := models.VaultUpsertResult{VaultPath: item.VaultPath}
		status, noteID, err := s.upsertOne(ctx, item)
		if err != nil {
			res.Status = "error"
			var ve *vaultItemError
			if errors.As(err, &ve) {
				res.Error = ve.msg
			} else {
				log.Printf("vault upsert %q: %v", item.VaultPath, err)
				res.Error = "internal error"
			}
		} else {
			res.Status = status
			res.NoteID = noteID
		}
		results = append(results, res)
	}
	return results, nil
}

// vaultItemError is a validation failure that is safe to show the worker.
type vaultItemError struct{ msg string }

func (e *vaultItemError) Error() string { return e.msg }

// vaultIncoming is one validated, parsed upsert item.
type vaultIncoming struct {
	item      models.VaultUpsertItem
	format    string
	meta      vault.Meta
	orgID     string
	sensitive bool
}

func (s *VaultService) upsertOne(ctx context.Context, item models.VaultUpsertItem) (string, string, error) {
	if err := vault.ValidateNotePath(item.VaultPath); err != nil {
		return "", "", &vaultItemError{msg: "invalid vault_path"}
	}
	if item.Commit != "" && !commitRe.MatchString(item.Commit) {
		return "", "", &vaultItemError{msg: "invalid commit"}
	}
	existing, err := s.repo.FindByVaultPath(ctx, s.userID, item.VaultPath)
	if err != nil {
		return "", "", fmt.Errorf("find by path: %w", err)
	}

	if item.Deleted {
		return s.applyDelete(ctx, existing, item)
	}
	if len(item.Content) > vaultMaxContentBytes {
		return "", "", &vaultItemError{msg: "content too large"}
	}

	in := newVaultIncoming(item, existing)
	switch {
	case existing == nil:
		return s.createFromVault(ctx, in)
	case existing.PendingWrite && existing.DeletedAt == nil:
		// The app has an edit the worker has not written yet.
		return s.reconcilePending(ctx, existing, in)
	case existing.Content != nil && *existing.Content == item.Content && existing.DeletedAt == nil && existing.Sensitive == in.sensitive:
		return s.touchUnchanged(ctx, existing, item.Commit)
	default:
		return s.overwriteFromVault(ctx, existing, in)
	}
}

// newVaultIncoming parses an upsert item and resolves its org id and
// sensitive flag (the user's in-app override wins over the path default).
func newVaultIncoming(item models.VaultUpsertItem, existing *models.Note) vaultIncoming {
	format := vault.FormatForPath(item.VaultPath)
	meta := vault.ParseNote(format, item.Content, item.VaultPath)
	orgID := meta.OrgID
	if orgID == "" && vault.ValidOrgID(item.OrgID) {
		orgID = item.OrgID
	}
	var override *bool
	if existing != nil {
		override = existing.SensitiveOverride
	}
	return vaultIncoming{
		item:      item,
		format:    format,
		meta:      meta,
		orgID:     orgID,
		sensitive: vault.ResolveSensitive(meta.Sensitive, vault.IsSensitivePath(item.VaultPath), override),
	}
}

// createFromVault inserts a note that exists in the vault but not yet in the app.
func (s *VaultService) createFromVault(ctx context.Context, in vaultIncoming) (string, string, error) {
	row := map[string]any{
		"user_id":        s.userID,
		"title":          in.meta.Title,
		"content":        in.item.Content,
		"source":         "vault",
		"vault_path":     in.item.VaultPath,
		"content_format": in.format,
		"sensitive":      in.sensitive,
	}
	if in.orgID != "" {
		row["org_id"] = in.orgID
	}
	if in.item.Commit != "" {
		row["vault_commit"] = in.item.Commit
	}
	created, err := s.repo.CreateNote(ctx, row)
	if err != nil || created == nil {
		return "", "", fmt.Errorf("create note: %w", err)
	}
	if err := s.saveLinks(ctx, created.ID, in.orgID, in.meta.Links); err != nil {
		return "", "", err
	}
	return "created", created.ID, nil
}

// reconcilePending handles an incoming vault version of a note that has an
// unwritten app edit: identical content clears the pending flag, anything else
// keeps both versions and flags a conflict (the app edit is never overwritten).
func (s *VaultService) reconcilePending(ctx context.Context, existing *models.Note, in vaultIncoming) (string, string, error) {
	if existing.Content != nil && *existing.Content == in.item.Content {
		// The vault already has exactly our version (e.g. the echo of a
		// commit the worker just pushed): nothing to reconcile.
		fields := map[string]any{"pending_write": false, "conflict": false}
		if in.item.Commit != "" {
			fields["vault_commit"] = in.item.Commit
		}
		if _, err := s.repo.UpdateNote(ctx, existing.ID, fields); err != nil {
			return "", "", fmt.Errorf("clear pending: %w", err)
		}
		return "unchanged", existing.ID, nil
	}
	incoming := in.item.Content
	if err := s.archive(ctx, existing.ID, vaultConflictPrefix+in.meta.Title, &incoming); err != nil {
		return "", "", fmt.Errorf("archive vault version: %w", err)
	}
	if _, err := s.repo.UpdateNote(ctx, existing.ID, map[string]any{"conflict": true}); err != nil {
		return "", "", fmt.Errorf("flag conflict: %w", err)
	}
	return "conflict", existing.ID, nil
}

// touchUnchanged records the commit for a note whose content did not change.
func (s *VaultService) touchUnchanged(ctx context.Context, existing *models.Note, commit string) (string, string, error) {
	if commit != "" && (existing.VaultCommit == nil || *existing.VaultCommit != commit) {
		if _, err := s.repo.UpdateNote(ctx, existing.ID, map[string]any{"vault_commit": commit}); err != nil {
			return "", "", fmt.Errorf("update commit: %w", err)
		}
	}
	return "unchanged", existing.ID, nil
}

// overwriteFromVault replaces an existing app note with the vault version.
func (s *VaultService) overwriteFromVault(ctx context.Context, existing *models.Note, in vaultIncoming) (string, string, error) {
	fields := map[string]any{
		"title":          in.meta.Title,
		"content":        in.item.Content,
		"source":         "vault",
		"content_format": in.format,
		"sensitive":      in.sensitive,
		"deleted_at":     nil,
		"pending_write":  false,
		"conflict":       false,
	}
	if in.orgID != "" {
		fields["org_id"] = in.orgID
	} else {
		fields["org_id"] = nil
	}
	if in.item.Commit != "" {
		fields["vault_commit"] = in.item.Commit
	}
	if _, err := s.repo.UpdateNote(ctx, existing.ID, fields); err != nil {
		return "", "", fmt.Errorf("update note: %w", err)
	}
	if err := s.saveLinks(ctx, existing.ID, in.orgID, in.meta.Links); err != nil {
		return "", "", err
	}
	return "updated", existing.ID, nil
}

func (s *VaultService) applyDelete(ctx context.Context, existing *models.Note, item models.VaultUpsertItem) (string, string, error) {
	if existing == nil || existing.DeletedAt != nil {
		return "skipped", "", nil
	}
	if existing.PendingWrite {
		// Deleted in git while edited in the app: keep the app version and
		// let the user decide. A nil-content "[vault]" version means "deleted".
		if err := s.archive(ctx, existing.ID, vaultConflictPrefix+existing.Title, nil); err != nil {
			return "", "", fmt.Errorf("archive deletion: %w", err)
		}
		if _, err := s.repo.UpdateNote(ctx, existing.ID, map[string]any{"conflict": true}); err != nil {
			return "", "", fmt.Errorf("flag conflict: %w", err)
		}
		return "conflict", existing.ID, nil
	}
	fields := map[string]any{
		"deleted_at":    time.Now().UTC().Format(time.RFC3339),
		"pending_write": false,
	}
	if item.Commit != "" {
		fields["vault_commit"] = item.Commit
	}
	if _, err := s.repo.UpdateNote(ctx, existing.ID, fields); err != nil {
		return "", "", fmt.Errorf("soft delete: %w", err)
	}
	return "deleted", existing.ID, nil
}

// saveLinks re-indexes a note's outgoing links and resolves links from other
// notes that were waiting for this note's org id.
func (s *VaultService) saveLinks(ctx context.Context, noteID, orgID string, links []vault.Link) error {
	rows := make([]models.NoteLinkRow, 0, len(links))
	for _, l := range links {
		row := models.NoteLinkRow{FromNoteID: noteID, ToOrgID: l.Target}
		if l.Label != "" {
			label := l.Label
			row.Label = &label
		}
		if ref, err := s.repo.FindByOrgID(ctx, s.userID, l.Target); err == nil && ref != nil {
			id := ref.ID
			row.ToNoteID = &id
		}
		rows = append(rows, row)
	}
	if err := s.repo.ReplaceLinks(ctx, noteID, rows); err != nil {
		return fmt.Errorf("replace links: %w", err)
	}
	if orgID != "" {
		if err := s.repo.ResolveLinksTo(ctx, orgID, noteID); err != nil {
			return fmt.Errorf("resolve links: %w", err)
		}
	}
	return nil
}

// Pending lists app-edited notes the worker should write to git, plus
// whether the user pressed Sync.
func (s *VaultService) Pending(ctx context.Context) (*models.VaultPendingResponse, error) {
	if s.userID == "" {
		return nil, ErrVaultDisabled
	}
	notes, err := s.repo.ListPending(ctx, s.userID)
	if err != nil {
		return nil, fmt.Errorf("list pending: %w", err)
	}
	out := &models.VaultPendingResponse{Notes: []models.VaultPendingNote{}}
	for _, n := range notes {
		// Defense in depth: never hand the worker a path it should not write.
		if n.VaultPath == nil || vault.ValidateNotePath(*n.VaultPath) != nil {
			continue
		}
		content := ""
		if n.Content != nil {
			content = *n.Content
		}
		out.Notes = append(out.Notes, models.VaultPendingNote{
			ID: n.ID, VaultPath: *n.VaultPath, Content: content, ContentFormat: n.ContentFormat,
		})
	}
	state, err := s.repo.GetSyncState(ctx, s.userID)
	if err != nil {
		return nil, fmt.Errorf("get sync state: %w", err)
	}
	out.SyncRequested = state != nil && state.RequestedAt != nil &&
		(state.HandledAt == nil || state.RequestedAt.After(*state.HandledAt))
	return out, nil
}

// Ack records that the worker committed and pushed a pending note.
func (s *VaultService) Ack(ctx context.Context, req models.VaultAckRequest) error {
	if s.userID == "" {
		return ErrVaultDisabled
	}
	if !uuidRe.MatchString(req.ID) {
		return fmt.Errorf("invalid id")
	}
	if !commitRe.MatchString(req.Commit) {
		return fmt.Errorf("invalid commit")
	}
	note, err := s.repo.FindOwned(ctx, s.userID, req.ID)
	if err != nil {
		return fmt.Errorf("find note: %w", err)
	}
	if note == nil {
		return ErrNoteNotFound
	}
	fields := map[string]any{"vault_commit": req.Commit}
	stillCurrent := true
	if req.ContentSHA256 != "" {
		content := ""
		if note.Content != nil {
			content = *note.Content
		}
		sum := sha256.Sum256([]byte(content))
		stillCurrent = strings.EqualFold(hex.EncodeToString(sum[:]), req.ContentSHA256)
	}
	if stillCurrent {
		fields["pending_write"] = false
	}
	_, err = s.repo.UpdateNote(ctx, req.ID, fields)
	return err
}

// SyncDone records the commit the worker last fully synchronized.
func (s *VaultService) SyncDone(ctx context.Context, commit string) error {
	if s.userID == "" {
		return ErrVaultDisabled
	}
	if !commitRe.MatchString(commit) {
		return fmt.Errorf("invalid commit")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.repo.UpsertSyncState(ctx, map[string]any{
		"user_id": s.userID, "handled_at": now, "last_commit": commit, "last_synced_at": now,
	})
	return err
}

// ---------------------------------------------------------------------
// User-facing operations (run under the caller's JWT, RLS applies)
// ---------------------------------------------------------------------

// RequestSync sets the flag the worker polls.
func (s *VaultService) RequestSync(ctx context.Context) (*models.VaultSyncState, error) {
	uid := auth.UserIDFromContext(ctx)
	if uid == "" {
		return nil, fmt.Errorf("no authenticated user")
	}
	return s.repo.UpsertSyncState(ctx, map[string]any{
		"user_id": uid, "requested_at": time.Now().UTC().Format(time.RFC3339),
	})
}

// SyncStatus returns the caller's sync state (empty if never synced).
func (s *VaultService) SyncStatus(ctx context.Context) (*models.VaultSyncState, error) {
	state, err := s.repo.GetSyncState(ctx, "")
	if err != nil {
		return nil, err
	}
	if state == nil {
		state = &models.VaultSyncState{}
	}
	return state, nil
}

// SetSensitive toggles the in-app sensitive flag. A flag declared inside the
// file itself cannot be switched off from the app.
func (s *VaultService) SetSensitive(ctx context.Context, id string, value bool) (*models.Note, error) {
	note, err := s.notes.notes.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("find note: %w", err)
	}
	if note == nil {
		return nil, ErrNoteNotFound
	}
	if !value && note.Content != nil {
		format := note.ContentFormat
		if format == "" {
			format = "md"
		}
		path := ""
		if note.VaultPath != nil {
			path = *note.VaultPath
		}
		if vault.ParseNote(format, *note.Content, path).Sensitive {
			return nil, ErrSensitiveDeclared
		}
	}
	return s.notes.notes.Update(ctx, id, map[string]any{"sensitive": value, "sensitive_override": value})
}

// Backlinks lists notes linking to the given note.
func (s *VaultService) Backlinks(ctx context.Context, id string) ([]models.Backlink, error) {
	if !uuidRe.MatchString(id) {
		return nil, ErrNoteNotFound
	}
	note, err := s.notes.notes.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("find note: %w", err)
	}
	if note == nil {
		return nil, ErrNoteNotFound
	}
	orgID := ""
	if note.OrgID != nil && vault.ValidOrgID(*note.OrgID) {
		orgID = *note.OrgID
	}
	rows, err := s.repo.ListBacklinks(ctx, id, orgID)
	if err != nil {
		return nil, err
	}
	out := make([]models.Backlink, 0, len(rows))
	seen := map[string]bool{}
	for _, row := range rows {
		if row.Notes == nil || row.Notes.ID == id || seen[row.Notes.ID] {
			continue
		}
		seen[row.Notes.ID] = true
		bl := models.Backlink{NoteID: row.Notes.ID, Title: row.Notes.Title, Sensitive: row.Notes.Sensitive}
		if row.Label != nil {
			bl.Label = *row.Label
		}
		out = append(out, bl)
	}
	return out, nil
}

// Graph returns all live notes and the resolved links between them. Links
// whose target is not (yet) a known note are dropped, as are self-links and
// duplicates.
func (s *VaultService) Graph(ctx context.Context) (*models.NoteGraph, error) {
	nodes, err := s.repo.ListGraphNodes(ctx)
	if err != nil {
		return nil, err
	}
	links, err := s.repo.ListGraphLinks(ctx)
	if err != nil {
		return nil, err
	}
	g := &models.NoteGraph{Nodes: make([]models.GraphNode, 0, len(nodes)), Edges: []models.GraphEdge{}}
	byID := make(map[string]bool, len(nodes))
	byOrg := make(map[string]string, len(nodes))
	for _, n := range nodes {
		g.Nodes = append(g.Nodes, *n)
		byID[n.ID] = true
		if n.OrgID != nil && *n.OrgID != "" {
			byOrg[*n.OrgID] = n.ID
		}
	}
	g.Edges = graphEdges(links, byID, byOrg)
	return g, nil
}

// graphEdges resolves link rows to deduplicated note-to-note edges.
func graphEdges(links []*models.GraphLinkRow, byID map[string]bool, byOrg map[string]string) []models.GraphEdge {
	edges := []models.GraphEdge{}
	seen := map[string]bool{}
	for _, l := range links {
		var target string
		if l.ToNoteID != nil {
			target = *l.ToNoteID
		} else {
			target = byOrg[l.ToOrgID]
		}
		key := l.FromNoteID + ">" + target
		if target == "" || target == l.FromNoteID || !byID[l.FromNoteID] || !byID[target] || seen[key] {
			continue
		}
		seen[key] = true
		edges = append(edges, models.GraphEdge{Source: l.FromNoteID, Target: target})
	}
	return edges
}

// Search runs full-text search over the caller's notes.
func (s *VaultService) Search(ctx context.Context, q string, limit int) ([]*models.NoteSearchHit, error) {
	q = strings.TrimSpace(q)
	if q == "" {
		return []*models.NoteSearchHit{}, nil
	}
	if len(q) > vaultMaxSearchLen {
		q = q[:vaultMaxSearchLen]
	}
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	hits, err := s.repo.Search(ctx, q, limit)
	if err != nil {
		return nil, err
	}
	if hits == nil {
		hits = []*models.NoteSearchHit{}
	}
	return hits, nil
}

// LookupByOrgID resolves an org-roam id to a note (for in-note id: links).
func (s *VaultService) LookupByOrgID(ctx context.Context, orgID string) (*models.NoteRef, error) {
	if !vault.ValidOrgID(orgID) {
		return nil, ErrNoteNotFound
	}
	ref, err := s.repo.FindByOrgID(ctx, "", orgID)
	if err != nil {
		return nil, err
	}
	if ref == nil {
		return nil, ErrNoteNotFound
	}
	return ref, nil
}

// ResolveConflict settles a conflicted note: keep == "mine" keeps the app
// version (it will be written to git over the vault one, as a normal
// commit — never a force push); keep == "vault" adopts the archived
// incoming vault version.
func (s *VaultService) ResolveConflict(ctx context.Context, id, keep string) (*models.Note, error) {
	note, err := s.notes.notes.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("find note: %w", err)
	}
	if note == nil {
		return nil, ErrNoteNotFound
	}
	if !note.Conflict {
		return nil, fmt.Errorf("note is not in conflict")
	}
	switch keep {
	case "mine":
		return s.notes.notes.Update(ctx, id, map[string]any{"conflict": false, "pending_write": true})
	case "vault":
		vv, err := s.latestVaultVersion(ctx, id)
		if err != nil {
			return nil, err
		}
		if vv == nil {
			return nil, fmt.Errorf("no archived vault version found")
		}
		// Keep the app version in history before replacing it.
		if err := s.archive(ctx, id, note.Title, note.Content); err != nil {
			return nil, fmt.Errorf("archive app version: %w", err)
		}
		if vv.Content == nil {
			return s.notes.notes.Update(ctx, id, map[string]any{
				"conflict": false, "pending_write": false, "deleted_at": time.Now().UTC().Format(time.RFC3339),
			})
		}
		path := ""
		if note.VaultPath != nil {
			path = *note.VaultPath
		}
		format := note.ContentFormat
		if format == "" {
			format = "md"
		}
		meta := vault.ParseNote(format, *vv.Content, path)
		sensitive := vault.ResolveSensitive(meta.Sensitive, vault.IsSensitivePath(path), note.SensitiveOverride)
		fields := map[string]any{
			"title": meta.Title, "content": *vv.Content, "conflict": false, "pending_write": false,
			"sensitive": sensitive,
		}
		if meta.OrgID != "" {
			fields["org_id"] = meta.OrgID
		}
		updated, err := s.notes.notes.Update(ctx, id, fields)
		if err != nil {
			return nil, err
		}
		if err := s.saveLinksUser(ctx, id, meta.Links); err != nil {
			log.Printf("resolve conflict: re-index links: %v", err)
		}
		return updated, nil
	}
	return nil, fmt.Errorf("keep must be \"mine\" or \"vault\"")
}

func (s *VaultService) latestVaultVersion(ctx context.Context, noteID string) (*models.NoteVersion, error) {
	versions, err := s.versions.ListByNote(ctx, noteID)
	if err != nil {
		return nil, fmt.Errorf("list versions: %w", err)
	}
	for _, v := range versions { // newest first
		if strings.HasPrefix(v.Title, vaultConflictPrefix) {
			cp := *v
			cp.Title = strings.TrimPrefix(v.Title, vaultConflictPrefix)
			return &cp, nil
		}
	}
	return nil, nil
}

// saveLinksUser re-indexes links under the caller's JWT (RLS), without the
// single-user org-id resolution the worker path does.
func (s *VaultService) saveLinksUser(ctx context.Context, noteID string, links []vault.Link) error {
	rows := make([]models.NoteLinkRow, 0, len(links))
	for _, l := range links {
		row := models.NoteLinkRow{FromNoteID: noteID, ToOrgID: l.Target}
		if l.Label != "" {
			label := l.Label
			row.Label = &label
		}
		if ref, err := s.repo.FindByOrgID(ctx, "", l.Target); err == nil && ref != nil {
			id := ref.ID
			row.ToNoteID = &id
		}
		rows = append(rows, row)
	}
	return s.repo.ReplaceLinks(ctx, noteID, rows)
}
