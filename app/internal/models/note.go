package models

import "time"

// Note is the stored shape from notes.notes.
type Note struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id,omitempty"`
	Title     string    `json:"title"`
	Content   *string   `json:"content,omitempty"`
	Pinned    bool      `json:"pinned"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`

	// Vault sync fields (notes/004_vault_sync.sql). Git is the source of
	// truth for source == "vault" notes; see docs/NOTES_SYNC.md.
	Source        string     `json:"source"`
	VaultPath     *string    `json:"vault_path,omitempty"`
	OrgID         *string    `json:"org_id,omitempty"`
	ContentFormat string     `json:"content_format"`
	VaultCommit   *string    `json:"vault_commit,omitempty"`
	PendingWrite  bool       `json:"pending_write"`
	Conflict      bool       `json:"conflict"`
	DeletedAt     *time.Time `json:"deleted_at,omitempty"`
	Sensitive     bool       `json:"sensitive"`
	// SensitiveOverride is the user's in-app choice (nil = follow rules).
	SensitiveOverride *bool `json:"sensitive_override,omitempty"`
	// RenderedHTML is filled in by NoteService.FindByID for org notes —
	// sanitized, never stored. Not a column.
	RenderedHTML string `json:"rendered_html,omitempty"`
}

// NoteInput is the write shape for create/update.
type NoteInput struct {
	Title   string  `json:"title,omitempty"`
	Content *string `json:"content,omitempty"`
	Pinned  *bool   `json:"pinned,omitempty"`
}

// NoteVersion is an archived prior title/content of a note, snapshotted
// right before an update overwrites it.
type NoteVersion struct {
	ID            string    `json:"id,omitempty"`
	NoteID        string    `json:"note_id,omitempty"`
	Title         string    `json:"title"`
	Content       *string   `json:"content,omitempty"`
	VersionNumber int       `json:"version_number"`
	CreatedAt     time.Time `json:"created_at,omitempty"`
}
