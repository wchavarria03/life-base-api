package supabase

import (
	"context"
	"net/url"

	"life-base-api/app/internal/databases"
	"life-base-api/app/internal/models"
)

// TaskLogRepository reads and writes tasks.task_logs.
type TaskLogRepository struct {
	client *databases.SupabaseClient
}

// NewTaskLogRepository constructs a TaskLogRepository.
func NewTaskLogRepository(client *databases.SupabaseClient) *TaskLogRepository {
	return &TaskLogRepository{client: client}
}

// List returns logs matching filter, newest first. Scoping to the caller
// relies on RLS, like TaskRepository.List.
func (r *TaskLogRepository) List(ctx context.Context, filter models.TaskLogFilter) ([]*models.TaskLog, error) {
	params := url.Values{"order": []string{"performed_at.desc,created_at.desc"}}
	if filter.From != "" {
		params.Add("performed_at", "gte."+filter.From)
	}
	if filter.To != "" {
		params.Add("performed_at", "lte."+filter.To)
	}
	if filter.Area != "" {
		params.Set("area", "eq."+filter.Area)
	}
	if filter.Category != "" {
		params.Set("category", "eq."+filter.Category)
	}
	return databases.Get[[]*models.TaskLog](ctx, r.client, "/rest/v1/task_logs", params, tasksSchema)
}

// Create inserts a log row and returns it.
func (r *TaskLogRepository) Create(ctx context.Context, input models.TaskLogInput) (*models.TaskLog, error) {
	return databases.First(databases.Post[[]*models.TaskLog](ctx, r.client, "/rest/v1/task_logs", input, "return=representation", tasksSchema))
}

// Delete removes a log row by id.
func (r *TaskLogRepository) Delete(ctx context.Context, id string) error {
	return databases.Delete(ctx, r.client, "/rest/v1/task_logs", databases.EqID(id), tasksSchema)
}

// ListOpenByUser returns userID's not-yet-completed house/household tasks,
// explicitly filtered by user (not RLS) because the digest cron has no user
// JWT and runs with the service-role key — same reason as
// ReminderRepository.ListByUserID. Lives here rather than in task.go to keep
// that file's diff small.
func (r *TaskRepository) ListOpenByUser(ctx context.Context, userID string) ([]*models.Task, error) {
	return databases.Get[[]*models.Task](ctx, r.client, "/rest/v1/tasks", url.Values{
		"user_id":      []string{"eq." + userID},
		"category":     []string{"in.(house,household)"},
		"completed_at": []string{"is.null"},
		"order":        []string{"created_at.asc"},
	}, tasksSchema)
}
