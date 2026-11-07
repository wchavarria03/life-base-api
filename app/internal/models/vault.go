package models

import "time"

// VaultUpsertItem is one file change reported by the homelab sync worker.
type VaultUpsertItem struct {
	VaultPath     string `json:"vault_path"`
	OrgID         string `json:"org_id,omitempty"`
	ContentFormat string `json:"content_format,omitempty"`
	Content       string `json:"content"`
	Commit        string `json:"commit"`
	Deleted       bool   `json:"deleted,omitempty"`
}

// VaultUpsertRequest is the body of POST /v1/internal/notes/upsert.
type VaultUpsertRequest struct {
	Items []VaultUpsertItem `json:"items"`
}

// VaultUpsertResult reports what happened to one item.
// Status is one of: created, updated, unchanged, deleted, skipped, conflict, error.
type VaultUpsertResult struct {
	VaultPath string `json:"vault_path"`
	Status    string `json:"status"`
	NoteID    string `json:"note_id,omitempty"`
	Error     string `json:"error,omitempty"`
}

// VaultPendingNote is an app-edited note waiting to be written to git.
type VaultPendingNote struct {
	ID            string `json:"id"`
	VaultPath     string `json:"vault_path"`
	Content       string `json:"content"`
	ContentFormat string `json:"content_format"`
}

// VaultPendingResponse is the body of GET /v1/internal/notes/pending.
type VaultPendingResponse struct {
	SyncRequested bool               `json:"sync_requested"`
	Notes         []VaultPendingNote `json:"notes"`
}

// VaultAckRequest is the body of POST /v1/internal/notes/ack. ContentSHA256
// (optional) is the hex digest of the content the worker wrote; if the note
// changed again in the meantime, pending_write stays true.
type VaultAckRequest struct {
	ID            string `json:"id"`
	Commit        string `json:"commit"`
	ContentSHA256 string `json:"content_sha256,omitempty"`
}

// VaultSyncDoneRequest is the body of POST /v1/internal/notes/sync-done.
type VaultSyncDoneRequest struct {
	Commit string `json:"commit"`
}

// VaultSyncState is one row of notes.vault_sync_state.
type VaultSyncState struct {
	UserID       string     `json:"user_id,omitempty"`
	RequestedAt  *time.Time `json:"requested_at,omitempty"`
	HandledAt    *time.Time `json:"handled_at,omitempty"`
	LastCommit   *string    `json:"last_commit,omitempty"`
	LastSyncedAt *time.Time `json:"last_synced_at,omitempty"`
}

// NoteLinkRow is a row of notes.note_links (write shape).
type NoteLinkRow struct {
	FromNoteID string  `json:"from_note_id"`
	ToOrgID    string  `json:"to_org_id"`
	ToNoteID   *string `json:"to_note_id,omitempty"`
	Label      *string `json:"label,omitempty"`
}

// BacklinkSource is the embedded source note of a note_links row.
type BacklinkSource struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Sensitive bool   `json:"sensitive"`
}

// BacklinkRow is the PostgREST shape of a backlink query.
type BacklinkRow struct {
	FromNoteID string          `json:"from_note_id"`
	Label      *string         `json:"label"`
	Notes      *BacklinkSource `json:"notes"`
}

// Backlink is what the API returns for GET /v1/notes/:id/backlinks.
type Backlink struct {
	NoteID    string `json:"note_id"`
	Title     string `json:"title"`
	Label     string `json:"label,omitempty"`
	Sensitive bool   `json:"sensitive"`
}

// NoteSearchHit is one full-text search result. Snippet marks matches with
// "<<" and ">>" (plain text, not HTML).
type NoteSearchHit struct {
	ID        string  `json:"id"`
	Title     string  `json:"title"`
	Snippet   string  `json:"snippet"`
	Source    string  `json:"source"`
	Sensitive bool    `json:"sensitive"`
	VaultPath *string `json:"vault_path,omitempty"`
	Rank      float64 `json:"rank"`
}

// NoteRef is a minimal note reference (org-id lookup).
type NoteRef struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// GraphNode is one note in the notes graph (no content).
type GraphNode struct {
	ID        string  `json:"id"`
	Title     string  `json:"title"`
	Source    string  `json:"source"`
	VaultPath *string `json:"vault_path,omitempty"`
	OrgID     *string `json:"org_id,omitempty"`
	Pinned    bool    `json:"pinned"`
	Sensitive bool    `json:"sensitive"`
}

// GraphLinkRow is the PostgREST shape of a note_links row for the graph.
type GraphLinkRow struct {
	FromNoteID string  `json:"from_note_id"`
	ToNoteID   *string `json:"to_note_id"`
	ToOrgID    string  `json:"to_org_id"`
}

// GraphEdge connects two notes by id.
type GraphEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

// NoteGraph is what GET /v1/notes/graph returns.
type NoteGraph struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}
