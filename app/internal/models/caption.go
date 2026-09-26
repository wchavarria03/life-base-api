package models

import "time"

type CaptionCategory struct {
	ID     string `json:"id"`
	UserID string `json:"user_id,omitempty"`
	Name   string `json:"name"`
}

// CaptionTemplate is the API shape: CurrentBody and CategoryIDs are filled
// in by the service layer, not columns on caption_templates directly.
type CaptionTemplate struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id,omitempty"`
	Title       string    `json:"title"`
	CurrentBody string    `json:"current_body"`
	CategoryIDs []string  `json:"category_ids"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
	UpdatedAt   time.Time `json:"updated_at,omitempty"`
}

type CaptionTemplateVersion struct {
	ID            string    `json:"id"`
	TemplateID    string    `json:"template_id,omitempty"`
	Body          string    `json:"body"`
	VersionNumber int       `json:"version_number"`
	CreatedAt     time.Time `json:"created_at,omitempty"`
}
