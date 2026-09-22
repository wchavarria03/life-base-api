package supabase

import (
	"context"
	"net/url"

	"life-base-api/app/internal/databases"
	"life-base-api/app/internal/models"
)

type ServiceLogRepository struct {
	client *databases.SupabaseClient
}

func NewServiceLogRepository(client *databases.SupabaseClient) *ServiceLogRepository {
	return &ServiceLogRepository{client: client}
}

func (r *ServiceLogRepository) ListByBikeID(ctx context.Context, bikeID string) ([]*models.ServiceLog, error) {
	return databases.Get[[]*models.ServiceLog](ctx, r.client, "/service_logs", url.Values{
		"bike_id": []string{"eq." + bikeID},
		"order":   []string{"date.desc"},
	}, bikesSchema)
}

func (r *ServiceLogRepository) Create(ctx context.Context, input models.ServiceLogInput) (*models.ServiceLog, error) {
	rows, err := databases.Post[[]*models.ServiceLog](ctx, r.client, "/service_logs", input, "return=representation", bikesSchema)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

func (r *ServiceLogRepository) Delete(ctx context.Context, id string) error {
	return databases.Delete(ctx, r.client, "/service_logs?id=eq."+id, bikesSchema)
}

type MaintenanceTaskRepository struct {
	client *databases.SupabaseClient
}

func NewMaintenanceTaskRepository(client *databases.SupabaseClient) *MaintenanceTaskRepository {
	return &MaintenanceTaskRepository{client: client}
}

func (r *MaintenanceTaskRepository) ListByBikeID(ctx context.Context, bikeID string) ([]*models.MaintenanceTask, error) {
	return databases.Get[[]*models.MaintenanceTask](ctx, r.client, "/maintenance_tasks", url.Values{
		"bike_id": []string{"eq." + bikeID},
		"order":   []string{"created_at.asc"},
	}, bikesSchema)
}

func (r *MaintenanceTaskRepository) Create(ctx context.Context, input models.MaintenanceTaskInput) (*models.MaintenanceTask, error) {
	rows, err := databases.Post[[]*models.MaintenanceTask](ctx, r.client, "/maintenance_tasks", input, "return=representation", bikesSchema)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

func (r *MaintenanceTaskRepository) Update(ctx context.Context, id string, fields map[string]any) (*models.MaintenanceTask, error) {
	rows, err := databases.Patch[[]*models.MaintenanceTask](ctx, r.client, "/maintenance_tasks?id=eq."+id, fields, "return=representation", bikesSchema)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

func (r *MaintenanceTaskRepository) Delete(ctx context.Context, id string) error {
	return databases.Delete(ctx, r.client, "/maintenance_tasks?id=eq."+id, bikesSchema)
}
