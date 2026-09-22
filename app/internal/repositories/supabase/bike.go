package supabase

import (
	"context"
	"net/url"

	"life-base-api/app/internal/databases"
	"life-base-api/app/internal/models"
)

// bikesSchema is the Postgres schema the whole Bikes domain lives in —
// shared by every repository file in this domain.
const bikesSchema = "bikes"

type BikeRepository struct {
	client *databases.SupabaseClient
}

func NewBikeRepository(client *databases.SupabaseClient) *BikeRepository {
	return &BikeRepository{client: client}
}

func (r *BikeRepository) List(ctx context.Context) ([]*models.Bike, error) {
	return databases.Get[[]*models.Bike](ctx, r.client, "/rest/v1/bikes", url.Values{
		"order": []string{"created_at.asc"},
	}, bikesSchema)
}

func (r *BikeRepository) FindByID(ctx context.Context, id string) (*models.Bike, error) {
	rows, err := databases.Get[[]*models.Bike](ctx, r.client, "/rest/v1/bikes", url.Values{
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

func (r *BikeRepository) Create(ctx context.Context, input models.BikeInput) (*models.Bike, error) {
	rows, err := databases.Post[[]*models.Bike](ctx, r.client, "/rest/v1/bikes", input, "return=representation", bikesSchema)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

func (r *BikeRepository) Update(ctx context.Context, id string, fields map[string]any) (*models.Bike, error) {
	rows, err := databases.Patch[[]*models.Bike](ctx, r.client, "/rest/v1/bikes?id=eq."+id, fields, "return=representation", bikesSchema)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

func (r *BikeRepository) Delete(ctx context.Context, id string) error {
	return databases.Delete(ctx, r.client, "/rest/v1/bikes?id=eq."+id, bikesSchema)
}

// IncrementMileage adds distanceKm to the bike's mileage (called when an
// activity is logged against it).
func (r *BikeRepository) IncrementMileage(ctx context.Context, id string, distanceKm float64) error {
	bike, err := r.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if bike == nil {
		return nil
	}
	_, err = databases.Patch[[]*models.Bike](ctx, r.client, "/rest/v1/bikes?id=eq."+id,
		map[string]any{"mileage": bike.Mileage + distanceKm}, "return=minimal", bikesSchema)
	return err
}

type BikeFitHistoryRepository struct {
	client *databases.SupabaseClient
}

func NewBikeFitHistoryRepository(client *databases.SupabaseClient) *BikeFitHistoryRepository {
	return &BikeFitHistoryRepository{client: client}
}

func (r *BikeFitHistoryRepository) ListByBikeID(ctx context.Context, bikeID string) ([]*models.BikeFitHistory, error) {
	return databases.Get[[]*models.BikeFitHistory](ctx, r.client, "/rest/v1/bike_fit_history", url.Values{
		"bike_id": []string{"eq." + bikeID},
		"order":   []string{"date.desc"},
	}, bikesSchema)
}

func (r *BikeFitHistoryRepository) Create(ctx context.Context, input models.BikeFitHistoryInput) (*models.BikeFitHistory, error) {
	rows, err := databases.Post[[]*models.BikeFitHistory](ctx, r.client, "/rest/v1/bike_fit_history", input, "return=representation", bikesSchema)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

func (r *BikeFitHistoryRepository) Delete(ctx context.Context, id string) error {
	return databases.Delete(ctx, r.client, "/rest/v1/bike_fit_history?id=eq."+id, bikesSchema)
}
