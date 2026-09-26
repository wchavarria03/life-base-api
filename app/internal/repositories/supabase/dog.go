package supabase

import (
	"context"
	"net/url"

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

// Delete removes a recipient type (cascades to its recipients).
func (r *DogRecipientTypeRepository) Delete(ctx context.Context, id string) error {
	return databases.Delete(ctx, r.client, "/rest/v1/dog_recipient_types", databases.EqID(id))
}

// ── Recipients ───────────────────────────────────────────────────────────────

// DogRecipientRepository persists individual recipient containers.
type DogRecipientRepository struct {
	client *databases.SupabaseClient
}

// NewDogRecipientRepository constructs a DogRecipientRepository.
func NewDogRecipientRepository(client *databases.SupabaseClient) *DogRecipientRepository {
	return &DogRecipientRepository{client: client}
}

// List returns every recipient, optionally filtered by status.
func (r *DogRecipientRepository) List(ctx context.Context, status string) ([]*models.DogRecipient, error) {
	params := url.Values{"order": []string{"created_at.asc"}}
	if status != "" {
		params.Set("status", "eq."+status)
	}
	return databases.Get[[]*models.DogRecipient](ctx, r.client, "/rest/v1/dog_recipients", params)
}

// FindByID returns a recipient by id, or nil if not found.
func (r *DogRecipientRepository) FindByID(ctx context.Context, id string) (*models.DogRecipient, error) {
	rows, err := databases.Get[[]*models.DogRecipient](ctx, r.client, "/rest/v1/dog_recipients",
		url.Values{"id": []string{"eq." + id}, "limit": []string{"1"}})
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

// CreateEmpty inserts n empty recipients of recipientTypeID — used to
// provision containers when a recipient type is created or its quantity
// increased.
func (r *DogRecipientRepository) CreateEmpty(ctx context.Context, userID, recipientTypeID string, n int) error {
	if n <= 0 {
		return nil
	}
	rows := make([]map[string]string, n)
	for i := range rows {
		rows[i] = map[string]string{"user_id": userID, "recipient_type_id": recipientTypeID, "status": string(models.DogRecipientEmpty)}
	}
	_, err := databases.Post[[]*models.DogRecipient](ctx, r.client, "/rest/v1/dog_recipients", rows, "")
	return err
}

// MarkPortioned fills a container: status -> portioned, dog assignment set.
func (r *DogRecipientRepository) MarkPortioned(ctx context.Context, id, dogID1 string, dogID2 *string, portionedAt string) error {
	fields := map[string]any{
		"status":       string(models.DogRecipientPortioned),
		"dog_id_1":     dogID1,
		"dog_id_2":     dogID2,
		"portioned_at": portionedAt,
	}
	_, err := databases.Patch[[]*models.DogRecipient](ctx, r.client, "/rest/v1/dog_recipients", databases.EqID(id), fields, "")
	return err
}

// MarkFed recycles a container back to empty.
func (r *DogRecipientRepository) MarkFed(ctx context.Context, id string) error {
	fields := map[string]any{
		"status":       string(models.DogRecipientEmpty),
		"dog_id_1":     nil,
		"dog_id_2":     nil,
		"portioned_at": nil,
	}
	_, err := databases.Patch[[]*models.DogRecipient](ctx, r.client, "/rest/v1/dog_recipients", databases.EqID(id), fields, "")
	return err
}

// ── Bulk bags ────────────────────────────────────────────────────────────────

// DogBulkBagRepository persists bulk (large) frozen food bags.
type DogBulkBagRepository struct {
	client *databases.SupabaseClient
}

// NewDogBulkBagRepository constructs a DogBulkBagRepository.
func NewDogBulkBagRepository(client *databases.SupabaseClient) *DogBulkBagRepository {
	return &DogBulkBagRepository{client: client}
}

// List returns every bulk bag with remaining stock, oldest-frozen first.
func (r *DogBulkBagRepository) List(ctx context.Context) ([]*models.DogBulkBag, error) {
	return databases.Get[[]*models.DogBulkBag](ctx, r.client, "/rest/v1/dog_bulk_bags",
		url.Values{"order": []string{"frozen_date.asc"}})
}

// ListAvailable returns bulk bags with remaining stock > 0, oldest first —
// the draw order for PortionBatch.
func (r *DogBulkBagRepository) ListAvailable(ctx context.Context) ([]*models.DogBulkBag, error) {
	return databases.Get[[]*models.DogBulkBag](ctx, r.client, "/rest/v1/dog_bulk_bags",
		url.Values{
			"remaining_weight_grams": []string{"gt.0"},
			"order":                  []string{"frozen_date.asc"},
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
		"frozen_date":            input.FrozenDate,
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
