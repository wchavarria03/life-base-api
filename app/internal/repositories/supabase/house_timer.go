package supabase

import (
	"context"
	"net/url"
	"time"

	"life-base-api/app/internal/databases"
	"life-base-api/app/internal/models"
)

// HouseTimerRepository persists house_timers rows.
type HouseTimerRepository struct {
	client *databases.SupabaseClient
}

// NewHouseTimerRepository constructs a HouseTimerRepository.
func NewHouseTimerRepository(client *databases.SupabaseClient) *HouseTimerRepository {
	return &HouseTimerRepository{client: client}
}

// List returns the caller's timers from the last 24h (covers "still
// counting down" and "recently fired, still shown until dismissed"),
// newest fire_at first.
func (r *HouseTimerRepository) List(ctx context.Context) ([]*models.HouseTimer, error) {
	since := time.Now().Add(-24 * time.Hour).UTC().Format(time.RFC3339)
	return databases.Get[[]*models.HouseTimer](ctx, r.client, "/rest/v1/house_timers", url.Values{
		"created_at": []string{"gte." + since},
		"order":      []string{"fire_at.desc"},
	})
}

// Create inserts a new timer.
func (r *HouseTimerRepository) Create(ctx context.Context, input models.HouseTimerInput) (*models.HouseTimer, error) {
	return databases.First(databases.Post[[]*models.HouseTimer](ctx, r.client, "/rest/v1/house_timers", input, "return=representation"))
}

// Update patches arbitrary fields (owner-scoped by RLS when called with a
// user JWT; unscoped via service-role from the notify cron).
func (r *HouseTimerRepository) Update(ctx context.Context, id string, fields map[string]any) error {
	_, err := databases.Patch[[]*models.HouseTimer](ctx, r.client, "/rest/v1/house_timers",
		databases.EqID(id), fields, "return=representation")
	return err
}

// Delete removes a timer (RLS-scoped).
func (r *HouseTimerRepository) Delete(ctx context.Context, id string) error {
	return databases.Delete(ctx, r.client, "/rest/v1/house_timers", databases.EqID(id))
}

// ListDueUnnotified returns every user's timers that have fired but not yet
// been push-notified — called from the notify cron with no user scoping
// (service-role key).
func (r *HouseTimerRepository) ListDueUnnotified(ctx context.Context) ([]*models.HouseTimer, error) {
	return databases.Get[[]*models.HouseTimer](ctx, r.client, "/rest/v1/house_timers", url.Values{
		"fire_at":  []string{"lte." + time.Now().UTC().Format(time.RFC3339)},
		"notified": []string{"eq.false"},
	})
}
