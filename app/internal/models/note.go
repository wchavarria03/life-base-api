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
