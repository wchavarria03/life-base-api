package supabase

import (
	"context"
	"net/url"
	"strconv"

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
	return databases.Get[[]*models.Activity](ctx, r.client, "/rest/v1/activities", url.Values{
		"bike_id": []string{"eq." + bikeID},
		"order":   []string{"date.desc"},
	}, bikesSchema)
}

func (r *ActivityRepository) FindByID(ctx context.Context, id string) (*models.Activity, error) {
	rows, err := databases.Get[[]*models.Activity](ctx, r.client, "/rest/v1/activities", url.Values{
		"id":    []string{"eq." + id},
		"limit": []string{"1"},
	}, bikesSchema)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

// FindByStravaActivityID looks for an existing activity on this bike already
// imported from the same Strava activity, to guard against double-import.
func (r *ActivityRepository) FindByStravaActivityID(ctx context.Context, bikeID string, stravaActivityID int64) (*models.Activity, error) {
	rows, err := databases.Get[[]*models.Activity](ctx, r.client, "/rest/v1/activities", url.Values{
		"bike_id":            []string{"eq." + bikeID},
		"strava_activity_id": []string{"eq." + strconv.FormatInt(stravaActivityID, 10)},
		"limit":              []string{"1"},
	}, bikesSchema)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

func (r *ActivityRepository) Create(ctx context.Context, input models.ActivityInput) (*models.Activity, error) {
	rows, err := databases.Post[[]*models.Activity](ctx, r.client, "/rest/v1/activities", input, "return=representation", bikesSchema)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

func (r *ActivityRepository) Delete(ctx context.Context, id string) error {
	return databases.Delete(ctx, r.client, "/rest/v1/activities?id=eq."+id, bikesSchema)
}
