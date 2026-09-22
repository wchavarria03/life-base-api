package models

import "time"

// TaskCategory discriminates which module a task belongs to. All share one
// schema/table — see docs/HUB_PLAN.md Phase 3 for why.
type TaskCategory string

const (
	TaskHousehold TaskCategory = "household"
	TaskHouse     TaskCategory = "house"
	TaskTodo      TaskCategory = "todo"
)

// TaskStatus is derived, never stored — same pattern as ReminderStatus.
type TaskStatus string

const (
	TaskOverdue   TaskStatus = "overdue"
	TaskDueToday  TaskStatus = "due_today"
	TaskUpcoming  TaskStatus = "upcoming"
	TaskCompleted TaskStatus = "completed"
	// TaskNoDueDate: a one-off task with no due_date set — not orderable by
	// urgency, always shown but never "overdue".
	TaskNoDueDate TaskStatus = "no_due_date"
)

// Task is the stored shape from tasks.tasks. Two mutually-exclusive shapes
// on one row, picked by is_recurring:
//   - recurring (household/house chores): interval_days + last_completed_date
//   - one-off (most TODOs, a single house repair): due_date + completed_at
type Task struct {
	ID                string       `json:"id"`
	UserID            string       `json:"user_id,omitempty"`
	Category          TaskCategory `json:"category"`
	Title             string       `json:"title"`
	Description       *string      `json:"description,omitempty"`
	Priority          string       `json:"priority"`
	IsRecurring       bool         `json:"is_recurring"`
	IntervalDays      *int         `json:"interval_days,omitempty"`
	LastCompletedDate *string      `json:"last_completed_date,omitempty"`
	DueDate           *string      `json:"due_date,omitempty"`
	CompletedAt       *time.Time   `json:"completed_at,omitempty"`
	Notes             *string      `json:"notes,omitempty"`
	CreatedAt         time.Time    `json:"created_at,omitempty"`
	UpdatedAt         time.Time    `json:"updated_at,omitempty"`
}

// TaskInput is the write shape for create/update.
type TaskInput struct {
	Category     TaskCategory `json:"category,omitempty"`
	Title        string       `json:"title,omitempty"`
	Description  *string      `json:"description,omitempty"`
	Priority     string       `json:"priority,omitempty"`
	IsRecurring  *bool        `json:"is_recurring,omitempty"`
	IntervalDays *int         `json:"interval_days,omitempty"`
	DueDate      *string      `json:"due_date,omitempty"`
	Notes        *string      `json:"notes,omitempty"`
}

// TaskWithStatus enriches Task with a derived status field.
type TaskWithStatus struct {
	Task
	Status TaskStatus `json:"status"`
}
