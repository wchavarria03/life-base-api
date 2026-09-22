package supabase

import (
	"context"
	"net/url"

	"life-base-api/app/internal/databases"
	"life-base-api/app/internal/models"
)

type GearRepository struct {
	client *databases.SupabaseClient
}

func NewGearRepository(client *databases.SupabaseClient) *GearRepository {
	return &GearRepository{client: client}
}

func (r *GearRepository) List(ctx context.Context) ([]*models.Gear, error) {
	return databases.Get[[]*models.Gear](ctx, r.client, "/gear", url.Values{
		"order": []string{"created_at.asc"},
	}, bikesSchema)
}

func (r *GearRepository) FindByID(ctx context.Context, id string) (*models.Gear, error) {
	rows, err := databases.Get[[]*models.Gear](ctx, r.client, "/gear", url.Values{
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

// ListActiveByBikeID returns active gear linked to a bike — used when an
// activity is logged, to accumulate distance onto gear worn for that bike.
func (r *GearRepository) ListActiveByBikeID(ctx context.Context, bikeID string) ([]*models.Gear, error) {
	return databases.Get[[]*models.Gear](ctx, r.client, "/gear", url.Values{
		"bike_id":   []string{"eq." + bikeID},
		"is_active": []string{"eq.true"},
	}, bikesSchema)
}

func (r *GearRepository) Create(ctx context.Context, input models.GearInput) (*models.Gear, error) {
	rows, err := databases.Post[[]*models.Gear](ctx, r.client, "/gear", input, "return=representation", bikesSchema)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

func (r *GearRepository) Update(ctx context.Context, id string, fields map[string]any) (*models.Gear, error) {
	rows, err := databases.Patch[[]*models.Gear](ctx, r.client, "/gear?id=eq."+id, fields, "return=representation", bikesSchema)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

func (r *GearRepository) Delete(ctx context.Context, id string) error {
	return databases.Delete(ctx, r.client, "/gear?id=eq."+id, bikesSchema)
}

type BottleRepository struct {
	client *databases.SupabaseClient
}

func NewBottleRepository(client *databases.SupabaseClient) *BottleRepository {
	return &BottleRepository{client: client}
}

func (r *BottleRepository) List(ctx context.Context) ([]*models.Bottle, error) {
	return databases.Get[[]*models.Bottle](ctx, r.client, "/bottles", url.Values{
		"order": []string{"last_cleaned_date.asc"},
	}, bikesSchema)
}

func (r *BottleRepository) Create(ctx context.Context, input models.BottleInput) (*models.Bottle, error) {
	rows, err := databases.Post[[]*models.Bottle](ctx, r.client, "/bottles", input, "return=representation", bikesSchema)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

func (r *BottleRepository) Update(ctx context.Context, id string, fields map[string]any) (*models.Bottle, error) {
	rows, err := databases.Patch[[]*models.Bottle](ctx, r.client, "/bottles?id=eq."+id, fields, "return=representation", bikesSchema)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

func (r *BottleRepository) Delete(ctx context.Context, id string) error {
	return databases.Delete(ctx, r.client, "/bottles?id=eq."+id, bikesSchema)
}

type SupplyRepository struct {
	client *databases.SupabaseClient
}

func NewSupplyRepository(client *databases.SupabaseClient) *SupplyRepository {
	return &SupplyRepository{client: client}
}

func (r *SupplyRepository) List(ctx context.Context) ([]*models.Supply, error) {
	return databases.Get[[]*models.Supply](ctx, r.client, "/supplies", url.Values{
		"order": []string{"created_at.asc"},
	}, bikesSchema)
}

func (r *SupplyRepository) FindByID(ctx context.Context, id string) (*models.Supply, error) {
	rows, err := databases.Get[[]*models.Supply](ctx, r.client, "/supplies", url.Values{
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

func (r *SupplyRepository) Create(ctx context.Context, input models.SupplyInput) (*models.Supply, error) {
	rows, err := databases.Post[[]*models.Supply](ctx, r.client, "/supplies", input, "return=representation", bikesSchema)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

func (r *SupplyRepository) Update(ctx context.Context, id string, fields map[string]any) (*models.Supply, error) {
	rows, err := databases.Patch[[]*models.Supply](ctx, r.client, "/supplies?id=eq."+id, fields, "return=representation", bikesSchema)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

func (r *SupplyRepository) Delete(ctx context.Context, id string) error {
	return databases.Delete(ctx, r.client, "/supplies?id=eq."+id, bikesSchema)
}

type SupplyHistoryRepository struct {
	client *databases.SupabaseClient
}

func NewSupplyHistoryRepository(client *databases.SupabaseClient) *SupplyHistoryRepository {
	return &SupplyHistoryRepository{client: client}
}

func (r *SupplyHistoryRepository) List(ctx context.Context) ([]*models.SupplyHistory, error) {
	return databases.Get[[]*models.SupplyHistory](ctx, r.client, "/supply_history", url.Values{
		"order": []string{"depleted_date.desc"},
	}, bikesSchema)
}

func (r *SupplyHistoryRepository) Create(ctx context.Context, input models.SupplyHistoryInput) (*models.SupplyHistory, error) {
	rows, err := databases.Post[[]*models.SupplyHistory](ctx, r.client, "/supply_history", input, "return=representation", bikesSchema)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}
