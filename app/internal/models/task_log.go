package models

import "time"

// TaskLog is one row of tasks.task_logs: an immutable record that some piece
// of work was performed. Written automatically when a task is completed
// (TaskID set, title/category/area snapshotted) or manually for work that
// was never a task (TaskID nil).
type TaskLog struct {
	ID          string       `json:"id"`
	UserID      string       `json:"user_id,omitempty"`
	TaskID      *string      `json:"task_id,omitempty"`
	Category    TaskCategory `json:"category"`
	Title       string       `json:"title"`
	Area        *string      `json:"area,omitempty"`
	PerformedAt string       `json:"performed_at"`
	Note        *string      `json:"note,omitempty"`
	Cost        *float64     `json:"cost,omitempty"`
	Currency    *string      `json:"currency,omitempty"`
	CreatedAt   time.Time    `json:"created_at,omitempty"`
}

// TaskLogInput is the write shape for tasks.task_logs. PerformedAt is
// omitted when empty so the column default (current_date) applies.
type TaskLogInput struct {
	UserID      string       `json:"user_id,omitempty"`
	TaskID      *string      `json:"task_id,omitempty"`
	Category    TaskCategory `json:"category,omitempty"`
	Title       string       `json:"title,omitempty"`
	Area        *string      `json:"area,omitempty"`
	PerformedAt string       `json:"performed_at,omitempty"`
	Note        *string      `json:"note,omitempty"`
	Cost        *float64     `json:"cost,omitempty"`
	Currency    *string      `json:"currency,omitempty"`
}

// TaskCompletion is the optional body of POST /v1/tasks/:id/complete. Every
// field is optional; an empty body behaves exactly like before.
type TaskCompletion struct {
	PerformedAt string   `json:"performed_at,omitempty"`
	Note        *string  `json:"note,omitempty"`
	Cost        *float64 `json:"cost,omitempty"`
	Currency    *string  `json:"currency,omitempty"`
}

// TaskLogFilter narrows GET /v1/tasks/logs. Empty fields are not applied.
type TaskLogFilter struct {
	From     string
	To       string
	Area     string
	Category string
}

// TaskLogMonthCount is the number of logs performed in one month (1-12).
type TaskLogMonthCount struct {
	Month int `json:"month"`
	Count int `json:"count"`
}

// TaskLogAreaSummary is the log count and per-currency cost for one area.
// Area is "" for logs with no area.
type TaskLogAreaSummary struct {
	Area  string             `json:"area"`
	Count int                `json:"count"`
	Costs map[string]float64 `json:"costs"`
}

// TaskLogSummary is the yearly aggregate behind GET /v1/tasks/logs/summary.
type TaskLogSummary struct {
	Year           int                  `json:"year"`
	Total          int                  `json:"total"`
	ByMonth        []TaskLogMonthCount  `json:"by_month"`
	ByArea         []TaskLogAreaSummary `json:"by_area"`
	CostByCurrency map[string]float64   `json:"cost_by_currency"`
}
