package services

import (
	"context"
	"fmt"

	"life-base-api/app/internal/auth"
	"life-base-api/app/internal/models"
	supabaserepo "life-base-api/app/internal/repositories/supabase"
)

// TaskListService manages user-named task list groupings.
type TaskListService struct {
	repo *supabaserepo.TaskListRepository
}

// NewTaskListService constructs a TaskListService.
func NewTaskListService(repo *supabaserepo.TaskListRepository) *TaskListService {
	return &TaskListService{repo: repo}
}

// List returns the caller's lists, optionally filtered by category.
func (s *TaskListService) List(ctx context.Context, category string) ([]*models.TaskList, error) {
	lists, err := s.repo.List(ctx, category)
	if err != nil {
		return nil, fmt.Errorf("list task lists: %w", err)
	}
	return lists, nil
}

// Create adds a new task list for the caller.
func (s *TaskListService) Create(ctx context.Context, input models.TaskListInput) (*models.TaskList, error) {
	userID := auth.UserIDFromContext(ctx)
	if userID == "" {
		return nil, fmt.Errorf("no authenticated user")
	}
	if input.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	switch models.TaskCategory(input.Category) {
	case models.TaskHousehold, models.TaskHouse, models.TaskTodo:
	default:
		return nil, fmt.Errorf("invalid category: %s", input.Category)
	}
	list, err := s.repo.Create(ctx, &models.TaskList{
		UserID:   userID,
		Category: input.Category,
		Name:     input.Name,
	})
	if err != nil {
		return nil, fmt.Errorf("create task list: %w", err)
	}
	return list, nil
}

// Delete removes a task list.
func (s *TaskListService) Delete(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete task list: %w", err)
	}
	return nil
}
