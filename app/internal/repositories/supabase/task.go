package supabase

import (
	"context"
	"net/url"

	"life-base-api/app/internal/databases"
	"life-base-api/app/internal/models"
)

const tasksSchema = "tasks"

type TaskRepository struct {
	client *databases.SupabaseClient
}

func NewTaskRepository(client *databases.SupabaseClient) *TaskRepository {
	return &TaskRepository{client: client}
}

func (r *TaskRepository) List(ctx context.Context, category *models.TaskCategory) ([]*models.Task, error) {
	params := url.Values{"order": []string{"created_at.asc"}}
	if category != nil {
		params.Set("category", "eq."+string(*category))
	}
	return databases.Get[[]*models.Task](ctx, r.client, "/rest/v1/tasks", params, tasksSchema)
}

func (r *TaskRepository) FindByID(ctx context.Context, id string) (*models.Task, error) {
	rows, err := databases.Get[[]*models.Task](ctx, r.client, "/rest/v1/tasks", url.Values{
		"id":    []string{"eq." + id},
		"limit": []string{"1"},
	}, tasksSchema)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

func (r *TaskRepository) Create(ctx context.Context, input models.TaskInput) (*models.Task, error) {
	rows, err := databases.Post[[]*models.Task](ctx, r.client, "/rest/v1/tasks", input, "return=representation", tasksSchema)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

func (r *TaskRepository) Update(ctx context.Context, id string, fields map[string]any) (*models.Task, error) {
	rows, err := databases.Patch[[]*models.Task](ctx, r.client, "/rest/v1/tasks?id=eq."+id, fields, "return=representation", tasksSchema)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

func (r *TaskRepository) Delete(ctx context.Context, id string) error {
	return databases.Delete(ctx, r.client, "/rest/v1/tasks?id=eq."+id, tasksSchema)
}
