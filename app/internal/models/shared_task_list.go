package models

import "time"

// SharedTaskList is a tokenized, unauthenticated read/complete link into one
// owner's task list for a single category. The token IS the access
// credential — see services/shared_task_list.go for how it's generated and
// resolved. Token is only ever populated in the response right after
// creation; List responses always leave it empty (json omitempty).
type SharedTaskList struct {
	ID        string       `json:"id"`
	UserID    string       `json:"user_id,omitempty"`
	Category  TaskCategory `json:"category"`
	Token     string       `json:"token,omitempty"`
	CreatedAt time.Time    `json:"created_at,omitempty"`
	Revoked   bool         `json:"revoked"`
}

// SharedTaskListInput is the write shape for creating a share row.
type SharedTaskListInput struct {
	UserID   string       `json:"user_id,omitempty"`
	Category TaskCategory `json:"category,omitempty"`
	Token    string       `json:"token,omitempty"`
}
