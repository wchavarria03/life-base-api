package services

import (
	"context"
	"fmt"
	"time"

	"life-base-api/app/internal/auth"
	"life-base-api/app/internal/models"
)

type TaskRepository interface {
	List(ctx context.Context, category *models.TaskCategory) ([]*models.Task, error)
	ListByUserAndCategory(ctx context.Context, userID string, category models.TaskCategory) ([]*models.Task, error)
	FindByID(ctx context.Context, id string) (*models.Task, error)
	Create(ctx context.Context, input models.TaskInput) (*models.Task, error)
	Update(ctx context.Context, id string, fields map[string]any) (*models.Task, error)
	Delete(ctx context.Context, id string) error
}

type TaskService struct {
	tasks TaskRepository
}

func NewTaskService(tasks TaskRepository) *TaskService {
	return &TaskService{tasks: tasks}
}

func (s *TaskService) List(ctx context.Context, category *models.TaskCategory) ([]models.TaskWithStatus, error) {
	tasks, err := s.tasks.List(ctx, category)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	result := make([]models.TaskWithStatus, 0, len(tasks))
	for _, t := range tasks {
		result = append(result, withTaskStatus(t))
	}
	return result, nil
}

// ListByUserAndCategory returns userID's tasks in category with derived
// status, bypassing the caller's own JWT/RLS context — used by
// SharedTaskListService to serve a public share link, where the "viewer" is
// resolved from an opaque token rather than a logged-in user.
func (s *TaskService) ListByUserAndCategory(ctx context.Context, userID string, category models.TaskCategory) ([]models.TaskWithStatus, error) {
	tasks, err := s.tasks.ListByUserAndCategory(ctx, userID, category)
	if err != nil {
		return nil, fmt.Errorf("list tasks by user and category: %w", err)
	}
	result := make([]models.TaskWithStatus, 0, len(tasks))
	for _, t := range tasks {
		result = append(result, withTaskStatus(t))
	}
	return result, nil
}

// FindByID looks up a single task, for callers (like SharedTaskListService)
// that need to validate ownership/category themselves before acting.
func (s *TaskService) FindByID(ctx context.Context, id string) (*models.Task, error) {
	return s.tasks.FindByID(ctx, id)
}

func (s *TaskService) Create(ctx context.Context, input models.TaskInput) (*models.Task, error) {
	userID := auth.UserIDFromContext(ctx)
	if userID == "" {
		return nil, fmt.Errorf("no authenticated user")
	}
	if input.Category == "" || input.Title == "" {
		return nil, fmt.Errorf("category and title are required")
	}
	switch input.Category {
	case models.TaskHousehold, models.TaskHouse, models.TaskTodo:
	default:
		return nil, fmt.Errorf("invalid category: %s", input.Category)
	}
	if input.Priority == "" {
		input.Priority = string(models.TaskMinor)
	}
	if err := validateTaskPriority(input.Priority); err != nil {
		return nil, err
	}
	if input.RecurrenceType != nil && *input.RecurrenceType != "" {
		if err := validateTaskRecurrence(*input.RecurrenceType); err != nil {
			return nil, err
		}
		if input.DueDate == nil || *input.DueDate == "" {
			return nil, fmt.Errorf("due_date is required when recurrence_type is set")
		}
	}
	// Bug pattern warned about repo-wide: an insert struct that never sets
	// user_id violates the NOT NULL constraint under RLS. See
	// services/preferences.go, services/push.go, services/dog.go for the
	// same fix.
	input.UserID = userID
	return s.tasks.Create(ctx, input)
}

func (s *TaskService) Update(ctx context.Context, id string, fields map[string]any) (*models.Task, error) {
	if len(fields) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}
	if p, ok := fields["priority"].(string); ok {
		if err := validateTaskPriority(p); err != nil {
			return nil, err
		}
	}
	if rt, ok := fields["recurrence_type"].(string); ok && rt != "" {
		if err := validateTaskRecurrence(rt); err != nil {
			return nil, err
		}
	}
	return s.tasks.Update(ctx, id, fields)
}

func (s *TaskService) Delete(ctx context.Context, id string) error {
	return s.tasks.Delete(ctx, id)
}

// Complete marks a task done. Three shapes, mirroring
// ReminderService.Complete's handling of recurrence:
//   - recurrence_type set (new-style recurring, with a due_date anchor):
//     stamp this row completed_at, then auto-create the next occurrence with
//     the advanced due date (advanceTaskDate) and link it back via
//     next_task_id — exact same shape as advanceReminderDate/next_reminder_id
//     in services/reminder.go.
//   - legacy is_recurring + interval_days (household/house chores predating
//     recurrence_type): today's date as last_completed_date; due-ness
//     recomputes from there, no new row.
//   - one-off: stamp completed_at, no next row.
func (s *TaskService) Complete(ctx context.Context, id string) (*models.Task, error) {
	task, err := s.tasks.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("find task: %w", err)
	}
	if task == nil {
		return nil, fmt.Errorf("task not found")
	}

	if task.RecurrenceType != nil && *task.RecurrenceType != "" && task.DueDate != nil {
		if _, err := s.tasks.Update(ctx, id, map[string]any{
			"completed_at": time.Now().UTC().Format(time.RFC3339),
		}); err != nil {
			return nil, fmt.Errorf("mark completed: %w", err)
		}

		nextDue := advanceTaskDate(*task.DueDate, models.TaskRecurrence(*task.RecurrenceType))
		next, err := s.tasks.Create(ctx, models.TaskInput{
			UserID:         task.UserID,
			Category:       task.Category,
			Title:          task.Title,
			Description:    task.Description,
			Priority:       task.Priority,
			DueDate:        &nextDue,
			RecurrenceType: task.RecurrenceType,
			Notes:          task.Notes,
		})
		if err != nil {
			return nil, fmt.Errorf("create next occurrence: %w", err)
		}
		if next != nil {
			if _, err := s.tasks.Update(ctx, id, map[string]any{"next_task_id": next.ID}); err != nil {
				return nil, fmt.Errorf("link next occurrence: %w", err)
			}
		}
		return s.tasks.FindByID(ctx, id)
	}

	if task.IsRecurring {
		return s.tasks.Update(ctx, id, map[string]any{
			"last_completed_date": time.Now().Format("2006-01-02"),
		})
	}
	return s.tasks.Update(ctx, id, map[string]any{
		"completed_at": time.Now().UTC().Format(time.RFC3339),
	})
}

func withTaskStatus(t *models.Task) models.TaskWithStatus {
	// New-style recurring: due-ness derived from due_date, same as a one-off,
	// except a completed row stays "completed" (the next occurrence, not this
	// row, is what's due going forward).
	if t.RecurrenceType != nil && *t.RecurrenceType != "" {
		if t.CompletedAt != nil {
			return models.TaskWithStatus{Task: *t, Status: models.TaskCompleted}
		}
		return models.TaskWithStatus{Task: *t, Status: deriveDueStatus(t.DueDate)}
	}

	// Legacy recurring: due-ness derived from interval_days elapsed since
	// last_completed_date.
	if t.IsRecurring {
		due := t.LastCompletedDate == nil
		if !due && t.IntervalDays != nil {
			if completed, err := time.Parse("2006-01-02", *t.LastCompletedDate); err == nil {
				due = !time.Now().Before(completed.AddDate(0, 0, *t.IntervalDays))
			}
		}
		status := models.TaskUpcoming
		if due {
			status = models.TaskOverdue
		}
		return models.TaskWithStatus{Task: *t, Status: status}
	}

	if t.CompletedAt != nil {
		return models.TaskWithStatus{Task: *t, Status: models.TaskCompleted}
	}
	if t.DueDate == nil {
		return models.TaskWithStatus{Task: *t, Status: models.TaskNoDueDate}
	}
	return models.TaskWithStatus{Task: *t, Status: deriveDueStatus(t.DueDate)}
}

// deriveDueStatus compares a due_date to today, same rule
// deriveReminderStatus in services/reminder.go uses.
func deriveDueStatus(dueDate *string) models.TaskStatus {
	if dueDate == nil {
		return models.TaskNoDueDate
	}
	today := time.Now().UTC().Format("2006-01-02")
	switch {
	case *dueDate < today:
		return models.TaskOverdue
	case *dueDate == today:
		return models.TaskDueToday
	default:
		return models.TaskUpcoming
	}
}

// advanceTaskDate is deliberately the same shape as advanceReminderDate in
// reminder.go — copying that exact recurrence pattern per this feature's
// design (see docs/HUB_PLAN.md), not accidental duplication.
//
//nolint:dupl // intentional mirror of advanceReminderDate's recurrence logic
func advanceTaskDate(date string, recurrence models.TaskRecurrence) string {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return date
	}
	switch recurrence {
	case models.TaskWeekly:
		return t.AddDate(0, 0, 7).Format("2006-01-02")
	case models.TaskBiweekly:
		return t.AddDate(0, 0, 14).Format("2006-01-02")
	case models.TaskMonthly:
		return t.AddDate(0, 1, 0).Format("2006-01-02")
	case models.TaskYearly:
		return t.AddDate(1, 0, 0).Format("2006-01-02")
	}
	return date
}

func validateTaskRecurrence(rt string) error {
	switch models.TaskRecurrence(rt) {
	case models.TaskWeekly, models.TaskBiweekly, models.TaskMonthly, models.TaskYearly:
		return nil
	}
	return fmt.Errorf("recurrence_type must be one of: weekly, biweekly, monthly, yearly")
}

func validateTaskPriority(p string) error {
	switch models.TaskPriority(p) {
	case models.TaskMinor, models.TaskMajor, models.TaskCritical:
		return nil
	}
	return fmt.Errorf("priority must be one of: minor, major, critical")
}
