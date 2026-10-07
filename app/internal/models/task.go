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

// TaskRecurrence is the recurrence cadence for a recurring task — the same
// weekly/biweekly/monthly/yearly enum Reminders uses (models.ReminderRecurrence).
type TaskRecurrence string

// TaskRecurrence values.
const (
	// TaskWeekly repeats a task every 7 days.
	TaskWeekly TaskRecurrence = "weekly"
	// TaskBiweekly repeats a task every 14 days.
	TaskBiweekly TaskRecurrence = "biweekly"
	// TaskMonthly repeats a task every calendar month.
	TaskMonthly TaskRecurrence = "monthly"
	// TaskYearly repeats a task every calendar year.
	TaskYearly TaskRecurrence = "yearly"
)

// TaskPriority is a 3-level urgency scale, color-coded the same way the
// frontend's Badge danger/warning/success variants are elsewhere.
type TaskPriority string

// TaskPriority values.
const (
	// TaskMinor is the default, lowest-urgency priority.
	TaskMinor TaskPriority = "minor"
	// TaskMajor is a mid-urgency priority.
	TaskMajor TaskPriority = "major"
	// TaskCritical is the highest-urgency priority.
	TaskCritical TaskPriority = "critical"
)

// Task is the stored shape from tasks.tasks. Two mutually-exclusive shapes
// on one row, picked by is_recurring:
//   - recurring (household/house chores): interval_days + last_completed_date,
//     or (newer) due_date + recurrence_type — see withTaskStatus/Complete in
//     services/task.go for how the two recurring shapes are reconciled.
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
	RecurrenceType    *string      `json:"recurrence_type,omitempty"`
	NextTaskID        *string      `json:"next_task_id,omitempty"`
	DueDate           *string      `json:"due_date,omitempty"`
	DueTime           *string      `json:"due_time,omitempty"`
	CompletedAt       *time.Time   `json:"completed_at,omitempty"`
	Notes             *string      `json:"notes,omitempty"`
	ListID            *string      `json:"list_id,omitempty"`
	ParentTaskID      *string      `json:"parent_task_id,omitempty"`
	Tags              []string     `json:"tags"`
	AssignedChildID   *string      `json:"assigned_child_id,omitempty"`
	CoinValue         *int         `json:"coin_value,omitempty"`
	PenaltyAppliedAt  *time.Time   `json:"penalty_applied_at,omitempty"`
	CreatedAt         time.Time    `json:"created_at,omitempty"`
	UpdatedAt         time.Time    `json:"updated_at,omitempty"`
}

// TaskInput is the write shape for create/update.
type TaskInput struct {
	UserID          string       `json:"user_id,omitempty"`
	Category        TaskCategory `json:"category,omitempty"`
	Title           string       `json:"title,omitempty"`
	Description     *string      `json:"description,omitempty"`
	Priority        string       `json:"priority,omitempty"`
	IsRecurring     *bool        `json:"is_recurring,omitempty"`
	IntervalDays    *int         `json:"interval_days,omitempty"`
	RecurrenceType  *string      `json:"recurrence_type,omitempty"`
	DueDate         *string      `json:"due_date,omitempty"`
	DueTime         *string      `json:"due_time,omitempty"`
	Notes           *string      `json:"notes,omitempty"`
	ListID          *string      `json:"list_id,omitempty"`
	ParentTaskID    *string      `json:"parent_task_id,omitempty"`
	Tags            []string     `json:"tags,omitempty"`
	AssignedChildID *string      `json:"assigned_child_id,omitempty"`
	CoinValue       *int         `json:"coin_value,omitempty"`
}

// TaskWithStatus enriches Task with a derived status field.
type TaskWithStatus struct {
	Task
	Status TaskStatus `json:"status"`
}
