package supabase

import (
	"context"
	"net/url"

	"life-base-api/app/internal/databases"
	"life-base-api/app/internal/models"
)

type ActivityRepository struct {
	client *databases.SupabaseClient
}

func NewActivityRepository(client *databases.SupabaseClient) *ActivityRepository {
	return &ActivityRepository{client: client}
}

func (r *ActivityRepository) ListByBikeID(ctx context.Context, bikeID string) ([]*models.Activity, error) {
	return databases.Get[[]*models.Activity](ctx, r.client, "/activities", url.Values{
		"bike_id": []string{"eq." + bikeID},
		"order":   []string{"date.desc"},
	}, bikesSchema)
}

func (r *ActivityRepository) Create(ctx context.Context, input models.ActivityInput) (*models.Activity, error) {
	rows, err := databases.Post[[]*models.Activity](ctx, r.client, "/activities", input, "return=representation", bikesSchema)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

func (r *ActivityRepository) Delete(ctx context.Context, id string) error {
	return databases.Delete(ctx, r.client, "/activities?id=eq."+id, bikesSchema)
}
