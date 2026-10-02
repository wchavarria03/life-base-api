package models

import "time"

// ShareResourceType is which kind of resource a share link points at.
type ShareResourceType string

const (
	// ShareResourceNote shares a single note.
	ShareResourceNote ShareResourceType = "note"
	// ShareResourceBike shares a single bike (plus its components).
	ShareResourceBike ShareResourceType = "bike"
)

// ShareLink is the stored shape from share_links.
type ShareLink struct {
	ID           string            `json:"id"`
	UserID       string            `json:"user_id,omitempty"`
	ResourceType ShareResourceType `json:"resource_type"`
	ResourceID   string            `json:"resource_id"`
	Token        string            `json:"token"`
	ExpiresAt    *time.Time        `json:"expires_at,omitempty"`
	Revoked      bool              `json:"revoked"`
	ViewCount    int               `json:"view_count"`
	LastViewedAt *time.Time        `json:"last_viewed_at,omitempty"`
	CreatedAt    time.Time         `json:"created_at,omitempty"`
	// ResourceTitle is filled in by the service layer for the "my shared
	// links" list — not a column, never read from or written to the DB.
	ResourceTitle string `json:"resource_title,omitempty"`
}

// ShareLinkInput is the write shape for Create.
type ShareLinkInput struct {
	UserID       string            `json:"user_id,omitempty"`
	ResourceType ShareResourceType `json:"resource_type,omitempty"`
	ResourceID   string            `json:"resource_id,omitempty"`
	Token        string            `json:"token,omitempty"`
	ExpiresAt    *time.Time        `json:"expires_at,omitempty"`
}

// SharedComponentView is the public, cost-free projection of a bike
// component.
type SharedComponentView struct {
	Name     string  `json:"name"`
	Brand    *string `json:"brand,omitempty"`
	Model    *string `json:"model,omitempty"`
	IsActive bool    `json:"is_active"`
}

// SharedBikeView is the public, read-only projection of a bike — no serial
// number, purchase date/location, or any cost field.
type SharedBikeView struct {
	Name       string                `json:"name"`
	Type       string                `json:"type"`
	Model      string                `json:"model"`
	Mileage    float64               `json:"mileage"`
	Components []SharedComponentView `json:"components"`
}

// SharedNoteView is the public, read-only projection of a note.
type SharedNoteView struct {
	Title   string  `json:"title"`
	Content *string `json:"content,omitempty"`
}

// SharedResource is the public resolver's response envelope — exactly one
// of Note/Bike is set, matching ResourceType.
type SharedResource struct {
	ResourceType ShareResourceType `json:"resource_type"`
	Note         *SharedNoteView   `json:"note,omitempty"`
	Bike         *SharedBikeView   `json:"bike,omitempty"`
}
