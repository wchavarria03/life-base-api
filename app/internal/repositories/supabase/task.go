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

// ListByUserAndCategory returns userID's tasks in category, explicitly
// filtered (rather than relying on RLS) because this backs the public
// /public/shared/:token route: that path has no user JWT, so requests go
// out with the service-role key, which bypasses RLS entirely — see
// databases.resolveKeys and services/shared_task_list.go.
func (r *TaskRepository) ListByUserAndCategory(ctx context.Context, userID string, category models.TaskCategory) ([]*models.Task, error) {
	params := url.Values{
		"user_id":  []string{"eq." + userID},
		"category": []string{"eq." + string(category)},
		"order":    []string{"created_at.asc"},
	}
	return databases.Get[[]*models.Task](ctx, r.client, "/rest/v1/tasks", params, tasksSchema)
}

func (r *TaskRepository) FindByID(ctx context.Context, id string) (*models.Task, error) {
	return databases.First(databases.Get[[]*models.Task](ctx, r.client, "/rest/v1/tasks", url.Values{
		"id":    []string{"eq." + id},
		"limit": []string{"1"},
	}, tasksSchema))
}

func (r *TaskRepository) Create(ctx context.Context, input models.TaskInput) (*models.Task, error) {
	return databases.First(databases.Post[[]*models.Task](ctx, r.client, "/rest/v1/tasks", input, "return=representation", tasksSchema))
}

func (r *TaskRepository) Update(ctx context.Context, id string, fields map[string]any) (*models.Task, error) {
	return databases.First(databases.Patch[[]*models.Task](ctx, r.client, "/rest/v1/tasks", databases.EqID(id), fields, "return=representation", tasksSchema))
}

func (r *TaskRepository) Delete(ctx context.Context, id string) error {
	return databases.Delete(ctx, r.client, "/rest/v1/tasks", databases.EqID(id), tasksSchema)
}
