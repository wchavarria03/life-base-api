package models

import "time"

// AuditLogEntry is the stored shape from audit_logs — one row per DELETE
// request, written by the AuditLog() middleware.
type AuditLogEntry struct {
	ID        string    `json:"id,omitempty"`
	UserID    string    `json:"user_id,omitempty"`
	Method    string    `json:"method"`
	Path      string    `json:"path"`
	Status    int       `json:"status"`
	CreatedAt time.Time `json:"created_at,omitempty"`
}
