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
		input.Priority = "medium"
	}
	return s.tasks.Create(ctx, input)
}

func (s *TaskService) Update(ctx context.Context, id string, fields map[string]any) (*models.Task, error) {
	if len(fields) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}
	return s.tasks.Update(ctx, id, fields)
}

func (s *TaskService) Delete(ctx context.Context, id string) error {
	return s.tasks.Delete(ctx, id)
}

// Complete marks a task done — for a recurring task that's today's date as
// last_completed_date (due-ness recomputes from there); for a one-off task
// it's completed_at.
func (s *TaskService) Complete(ctx context.Context, id string) (*models.Task, error) {
	task, err := s.tasks.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("find task: %w", err)
	}
	if task == nil {
		return nil, fmt.Errorf("task not found")
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

	today := time.Now().UTC().Format("2006-01-02")
	status := models.TaskUpcoming
	switch {
	case *t.DueDate < today:
		status = models.TaskOverdue
	case *t.DueDate == today:
		status = models.TaskDueToday
	}
	return models.TaskWithStatus{Task: *t, Status: status}
}
