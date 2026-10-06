package models

import "time"

// Note is the stored shape from notes.notes.
type Note struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id,omitempty"`
	Title     string    `json:"title"`
	Content   *string   `json:"content,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

// NoteInput is the write shape for create/update.
type NoteInput struct {
	Title   string  `json:"title,omitempty"`
	Content *string `json:"content,omitempty"`
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
