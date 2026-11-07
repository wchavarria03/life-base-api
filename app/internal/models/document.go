package models

import "time"

// Document is the stored shape from documents — a private file upload
// (insurance, receipts, warranty cards) with no public URL, unlike the
// social-images bucket.
type Document struct {
	ID          string    `json:"id,omitempty"`
	UserID      string    `json:"user_id,omitempty"`
	Title       string    `json:"title"`
	Category    *string   `json:"category,omitempty"`
	FileName    string    `json:"file_name"`
	ContentType string    `json:"content_type"`
	FileSize    int64     `json:"file_size"`
	StoragePath string    `json:"storage_path"`
	// Sensitive documents are never exposed through share links and get a
	// shorter signed-URL lifetime.
	Sensitive bool      `json:"sensitive"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
}

// DocumentInput is the write shape for Create.
type DocumentInput struct {
	UserID      string  `json:"user_id,omitempty"`
	Title       string  `json:"title"`
	Category    *string `json:"category,omitempty"`
	FileName    string  `json:"file_name"`
	ContentType string  `json:"content_type"`
	FileSize    int64   `json:"file_size"`
	StoragePath string  `json:"storage_path"`
	Sensitive   bool    `json:"sensitive"`
}

// Document link entity types.
const (
	DocumentEntityNote          = "note"
	DocumentEntityTaskLog       = "task_log"
	DocumentEntityBike          = "bike"
	DocumentEntityComponent     = "component"
	DocumentEntityMedicalRecord = "medical_record"
)

// DocumentLink attaches a document to one entity.
type DocumentLink struct {
	ID         string    `json:"id,omitempty"`
	UserID     string    `json:"user_id,omitempty"`
	DocumentID string    `json:"document_id"`
	EntityType string    `json:"entity_type"`
	EntityID   string    `json:"entity_id"`
	CreatedAt  time.Time `json:"created_at,omitempty"`
}

// DocumentLinkInput is the write shape for creating a link.
type DocumentLinkInput struct {
	UserID     string `json:"user_id,omitempty"`
	DocumentID string `json:"document_id"`
	EntityType string `json:"entity_type"`
	EntityID   string `json:"entity_id"`
}

// SignedDocumentURL is a short-lived URL for fetching a document without a
// JWT (for example an <img> tag).
type SignedDocumentURL struct {
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expires_at"`
}

// SharedAttachmentView is a non-sensitive attachment exposed in a share link.
type SharedAttachmentView struct {
	Title       string `json:"title"`
	ContentType string `json:"content_type"`
	URL         string `json:"url"`
}
