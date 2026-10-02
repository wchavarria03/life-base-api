package supabase

import (
	"context"
	"net/url"

	"life-base-api/app/internal/databases"
	"life-base-api/app/internal/models"
)

// TaskListRepository persists user-named task list groupings.
type TaskListRepository struct {
	client *databases.SupabaseClient
}

// NewTaskListRepository constructs a TaskListRepository.
func NewTaskListRepository(client *databases.SupabaseClient) *TaskListRepository {
	return &TaskListRepository{client: client}
}

// List returns the caller's lists, optionally filtered by category.
func (r *TaskListRepository) List(ctx context.Context, category string) ([]*models.TaskList, error) {
	params := url.Values{"order": []string{"name.asc"}}
	if category != "" {
		params.Set("category", "eq."+category)
	}
	return databases.Get[[]*models.TaskList](ctx, r.client, "/rest/v1/task_lists", params)
}

// Create inserts a new task list.
func (r *TaskListRepository) Create(ctx context.Context, l *models.TaskList) (*models.TaskList, error) {
	rows, err := databases.Post[[]*models.TaskList](ctx, r.client, "/rest/v1/task_lists", l, "return=representation")
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

// Delete removes a task list.
func (r *TaskListRepository) Delete(ctx context.Context, id string) error {
	return databases.Delete(ctx, r.client, "/rest/v1/task_lists", databases.EqID(id))
}
