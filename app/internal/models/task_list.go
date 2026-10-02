package models

import "time"

// TaskList is a user-named grouping of tasks within one category (e.g. a
// "Bicicleta" list under todo), independent of the overdue/today/upcoming
// status grouping.
type TaskList struct {
	ID        string    `json:"id,omitempty"`
	UserID    string    `json:"user_id,omitempty"`
	Category  string    `json:"category"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at,omitempty"`
}

// TaskListInput is the write shape for create.
type TaskListInput struct {
	Category string `json:"category"`
	Name     string `json:"name"`
}
