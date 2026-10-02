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
}
