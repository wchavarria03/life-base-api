package supabase

import (
	"context"
	"net/url"

	"life-base-api/app/internal/databases"
	"life-base-api/app/internal/models"
)

type ComponentRepository struct {
	client *databases.SupabaseClient
}

func NewComponentRepository(client *databases.SupabaseClient) *ComponentRepository {
	return &ComponentRepository{client: client}
}

func (r *ComponentRepository) ListByBikeID(ctx context.Context, bikeID string) ([]*models.Component, error) {
	return databases.Get[[]*models.Component](ctx, r.client, "/components", url.Values{
		"bike_id": []string{"eq." + bikeID},
		"order":   []string{"created_at.asc"},
	}, bikesSchema)
}

func (r *ComponentRepository) FindByID(ctx context.Context, id string) (*models.Component, error) {
	rows, err := databases.Get[[]*models.Component](ctx, r.client, "/components", url.Values{
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

// ListActiveByBikeID returns active components for a bike — used when an
// activity is logged, to accumulate distance onto every active component.
func (r *ComponentRepository) ListActiveByBikeID(ctx context.Context, bikeID string) ([]*models.Component, error) {
	return databases.Get[[]*models.Component](ctx, r.client, "/components", url.Values{
		"bike_id":   []string{"eq." + bikeID},
		"is_active": []string{"eq.true"},
	}, bikesSchema)
}

func (r *ComponentRepository) Create(ctx context.Context, input models.ComponentInput) (*models.Component, error) {
	rows, err := databases.Post[[]*models.Component](ctx, r.client, "/components", input, "return=representation", bikesSchema)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

func (r *ComponentRepository) Update(ctx context.Context, id string, fields map[string]any) (*models.Component, error) {
	rows, err := databases.Patch[[]*models.Component](ctx, r.client, "/components?id=eq."+id, fields, "return=representation", bikesSchema)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

func (r *ComponentRepository) Delete(ctx context.Context, id string) error {
	return databases.Delete(ctx, r.client, "/components?id=eq."+id, bikesSchema)
}

type ComponentHistoryRepository struct {
	client *databases.SupabaseClient
}

func NewComponentHistoryRepository(client *databases.SupabaseClient) *ComponentHistoryRepository {
	return &ComponentHistoryRepository{client: client}
}

func (r *ComponentHistoryRepository) ListByComponentID(ctx context.Context, componentID string) ([]*models.ComponentHistory, error) {
	return databases.Get[[]*models.ComponentHistory](ctx, r.client, "/component_history", url.Values{
		"component_id": []string{"eq." + componentID},
		"order":        []string{"replaced_date.desc"},
	}, bikesSchema)
}

func (r *ComponentHistoryRepository) Create(ctx context.Context, input models.ComponentHistoryInput) (*models.ComponentHistory, error) {
	rows, err := databases.Post[[]*models.ComponentHistory](ctx, r.client, "/component_history", input, "return=representation", bikesSchema)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}
