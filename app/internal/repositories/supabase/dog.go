package supabase

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"life-base-api/app/internal/databases"
	"life-base-api/app/internal/models"
)

// ── Dogs ─────────────────────────────────────────────────────────────────────

// DogRepository persists dogs.
type DogRepository struct {
	client *databases.SupabaseClient
}

// NewDogRepository constructs a DogRepository.
func NewDogRepository(client *databases.SupabaseClient) *DogRepository {
	return &DogRepository{client: client}
}

// List returns every dog.
func (r *DogRepository) List(ctx context.Context) ([]*models.Dog, error) {
	return databases.Get[[]*models.Dog](ctx, r.client, "/rest/v1/dogs", url.Values{"order": []string{"name.asc"}})
}

// FindByID returns a dog by id, or nil if not found.
func (r *DogRepository) FindByID(ctx context.Context, id string) (*models.Dog, error) {
	rows, err := databases.Get[[]*models.Dog](ctx, r.client, "/rest/v1/dogs",
		url.Values{"id": []string{"eq." + id}, "limit": []string{"1"}})
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

// Create inserts a new dog.
func (r *DogRepository) Create(ctx context.Context, input models.DogInput) (*models.Dog, error) {
	rows, err := databases.Post[[]*models.Dog](ctx, r.client, "/rest/v1/dogs", input, "return=representation")
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

// Update patches a dog's fields.
func (r *DogRepository) Update(ctx context.Context, id string, input models.DogInput) (*models.Dog, error) {
	rows, err := databases.Patch[[]*models.Dog](ctx, r.client, "/rest/v1/dogs", databases.EqID(id), input, "return=representation")
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

// Delete removes a dog.
func (r *DogRepository) Delete(ctx context.Context, id string) error {
	return databases.Delete(ctx, r.client, "/rest/v1/dogs", databases.EqID(id))
}

// ── Recipient types ──────────────────────────────────────────────────────────

// DogRecipientTypeRepository persists recipient (container) types.
type DogRecipientTypeRepository struct {
	client *databases.SupabaseClient
}

// NewDogRecipientTypeRepository constructs a DogRecipientTypeRepository.
func NewDogRecipientTypeRepository(client *databases.SupabaseClient) *DogRecipientTypeRepository {
	return &DogRecipientTypeRepository{client: client}
}

// List returns every recipient type.
func (r *DogRecipientTypeRepository) List(ctx context.Context) ([]*models.DogRecipientType, error) {
	return databases.Get[[]*models.DogRecipientType](ctx, r.client, "/rest/v1/dog_recipient_types",
		url.Values{"order": []string{"name.asc"}})
}

// Create inserts a new recipient type.
func (r *DogRecipientTypeRepository) Create(ctx context.Context, input models.DogRecipientTypeInput) (*models.DogRecipientType, error) {
	rows, err := databases.Post[[]*models.DogRecipientType](ctx, r.client, "/rest/v1/dog_recipient_types", input, "return=representation")
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

// Update patches a recipient type's fields.
func (r *DogRecipientTypeRepository) Update(ctx context.Context, id string, input models.DogRecipientTypeInput) (*models.DogRecipientType, error) {
	rows, err := databases.Patch[[]*models.DogRecipientType](ctx, r.client, "/rest/v1/dog_recipient_types",
		databases.EqID(id), input, "return=representation")
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

// FindByID returns a recipient type by id, or nil if not found.
func (r *DogRecipientTypeRepository) FindByID(ctx context.Context, id string) (*models.DogRecipientType, error) {
	rows, err := databases.Get[[]*models.DogRecipientType](ctx, r.client, "/rest/v1/dog_recipient_types",
		url.Values{"id": []string{"eq." + id}, "limit": []string{"1"}})
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

// Delete removes a recipient type (cascades to its allocations).
func (r *DogRecipientTypeRepository) Delete(ctx context.Context, id string) error {
	return databases.Delete(ctx, r.client, "/rest/v1/dog_recipient_types", databases.EqID(id))
}

// ── Recipient allocations ────────────────────────────────────────────────────

// DogRecipientAllocationRepository persists recipient allocations — batches
// of N filled containers of one type assigned to a dog (or dog pair).
type DogRecipientAllocationRepository struct {
	client *databases.SupabaseClient
}

// NewDogRecipientAllocationRepository constructs a DogRecipientAllocationRepository.
func NewDogRecipientAllocationRepository(client *databases.SupabaseClient) *DogRecipientAllocationRepository {
	return &DogRecipientAllocationRepository{client: client}
}

// List returns every allocation, oldest-portioned first.
func (r *DogRecipientAllocationRepository) List(ctx context.Context) ([]*models.DogRecipientAllocation, error) {
	return databases.Get[[]*models.DogRecipientAllocation](ctx, r.client, "/rest/v1/dog_recipient_allocations",
		url.Values{"order": []string{"portioned_at.asc"}})
}

// Create inserts a new allocation row.
func (r *DogRecipientAllocationRepository) Create(ctx context.Context, input models.DogRecipientAllocation) (*models.DogRecipientAllocation, error) {
	rows, err := databases.Post[[]*models.DogRecipientAllocation](ctx, r.client, "/rest/v1/dog_recipient_allocations", input, "return=representation")
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

// FindOldestActiveForDog returns the oldest (by portioned_at) allocation
// with quantity > 0 involving dogID as either dog_id_1 or dog_id_2 — a
// shared allocation is found and fed the same way from either dog's button.
// Returns nil if none is available.
func (r *DogRecipientAllocationRepository) FindOldestActiveForDog(ctx context.Context, dogID string) (*models.DogRecipientAllocation, error) {
	rows, err := databases.Get[[]*models.DogRecipientAllocation](ctx, r.client, "/rest/v1/dog_recipient_allocations",
		url.Values{
			"or":       []string{fmt.Sprintf("(dog_id_1.eq.%s,dog_id_2.eq.%s)", dogID, dogID)},
			"quantity": []string{"gt.0"},
			"order":    []string{"portioned_at.asc"},
			"limit":    []string{"1"},
		})
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

// SetQuantity updates an allocation's remaining quantity.
func (r *DogRecipientAllocationRepository) SetQuantity(ctx context.Context, id string, quantity int) error {
	_, err := databases.Patch[[]*models.DogRecipientAllocation](ctx, r.client, "/rest/v1/dog_recipient_allocations",
		databases.EqID(id), map[string]any{"quantity": quantity}, "")
	return err
}

// Delete removes an allocation (used once its quantity reaches zero).
func (r *DogRecipientAllocationRepository) Delete(ctx context.Context, id string) error {
	return databases.Delete(ctx, r.client, "/rest/v1/dog_recipient_allocations", databases.EqID(id))
}

// ── Feed log ─────────────────────────────────────────────────────────────────

// DogFeedLogRepository persists individual feed events.
type DogFeedLogRepository struct {
	client *databases.SupabaseClient
}

// NewDogFeedLogRepository constructs a DogFeedLogRepository.
func NewDogFeedLogRepository(client *databases.SupabaseClient) *DogFeedLogRepository {
	return &DogFeedLogRepository{client: client}
}

// Create inserts a new feed log entry.
func (r *DogFeedLogRepository) Create(ctx context.Context, entry models.DogFeedLogEntry) (*models.DogFeedLogEntry, error) {
	rows, err := databases.Post[[]*models.DogFeedLogEntry](ctx, r.client, "/rest/v1/dog_feed_log", entry, "return=representation")
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

// CountSince returns how many feed-log rows exist for dogID at or after since.
func (r *DogFeedLogRepository) CountSince(ctx context.Context, dogID string, since time.Time) (int, error) {
	rows, err := databases.Get[[]*models.DogFeedLogEntry](ctx, r.client, "/rest/v1/dog_feed_log",
		url.Values{
			"dog_id": []string{"eq." + dogID},
			"fed_at": []string{"gte." + since.Format(time.RFC3339)},
		})
	if err != nil {
		return 0, err
	}
	return len(rows), nil
}

// ── Bulk bags ────────────────────────────────────────────────────────────────

// DogBulkBagRepository persists bulk (large) raw-food bags.
type DogBulkBagRepository struct {
	client *databases.SupabaseClient
}

// NewDogBulkBagRepository constructs a DogBulkBagRepository.
func NewDogBulkBagRepository(client *databases.SupabaseClient) *DogBulkBagRepository {
	return &DogBulkBagRepository{client: client}
}

// List returns every bulk bag, oldest-purchased first.
func (r *DogBulkBagRepository) List(ctx context.Context) ([]*models.DogBulkBag, error) {
	return databases.Get[[]*models.DogBulkBag](ctx, r.client, "/rest/v1/dog_bulk_bags",
		url.Values{"order": []string{"purchase_date.asc"}})
}

// ListAvailable returns bulk bags with remaining stock > 0, oldest-purchased
// first — the draw order for PortionBatch.
func (r *DogBulkBagRepository) ListAvailable(ctx context.Context) ([]*models.DogBulkBag, error) {
	return databases.Get[[]*models.DogBulkBag](ctx, r.client, "/rest/v1/dog_bulk_bags",
		url.Values{
			"remaining_weight_grams": []string{"gt.0"},
			"order":                  []string{"purchase_date.asc"},
		})
}

// Create inserts a new bulk bag — remaining_weight_grams starts equal to
// total_weight_grams.
func (r *DogBulkBagRepository) Create(ctx context.Context, input models.DogBulkBagInput) (*models.DogBulkBag, error) {
	body := map[string]any{
		"user_id":                input.UserID,
		"label":                  input.Label,
		"total_weight_grams":     input.TotalWeightGrams,
		"remaining_weight_grams": input.TotalWeightGrams,
		"purchase_date":          input.PurchaseDate,
		"price":                  input.Price,
	}
	rows, err := databases.Post[[]*models.DogBulkBag](ctx, r.client, "/rest/v1/dog_bulk_bags", body, "return=representation")
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

// SetRemaining updates a bulk bag's remaining stock.
func (r *DogBulkBagRepository) SetRemaining(ctx context.Context, id string, remainingGrams int) error {
	_, err := databases.Patch[[]*models.DogBulkBag](ctx, r.client, "/rest/v1/dog_bulk_bags",
		databases.EqID(id), map[string]any{"remaining_weight_grams": remainingGrams}, "")
	return err
}

// Delete removes a bulk bag.
func (r *DogBulkBagRepository) Delete(ctx context.Context, id string) error {
	return databases.Delete(ctx, r.client, "/rest/v1/dog_bulk_bags", databases.EqID(id))
}

// ── Settings ─────────────────────────────────────────────────────────────────

// DogSettingsRepository persists the per-user low-stock threshold.
type DogSettingsRepository struct {
	client *databases.SupabaseClient
}

// NewDogSettingsRepository constructs a DogSettingsRepository.
func NewDogSettingsRepository(client *databases.SupabaseClient) *DogSettingsRepository {
	return &DogSettingsRepository{client: client}
}

// FindByUserID returns the user's settings row, or nil if unset.
func (r *DogSettingsRepository) FindByUserID(ctx context.Context, userID string) (*models.DogSettings, error) {
	rows, err := databases.Get[[]*models.DogSettings](ctx, r.client, "/rest/v1/dog_settings",
		url.Values{"user_id": []string{"eq." + userID}, "limit": []string{"1"}})
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

// Upsert creates or replaces the user's settings row.
func (r *DogSettingsRepository) Upsert(ctx context.Context, s *models.DogSettings) (*models.DogSettings, error) {
	rows, err := databases.Post[[]*models.DogSettings](ctx, r.client,
		"/rest/v1/dog_settings?on_conflict=user_id", s, "resolution=merge-duplicates,return=representation")
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}
