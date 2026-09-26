package services

import (
	"context"
	"fmt"
	"time"

	supabaserepo "life-base-api/app/internal/repositories/supabase"

	"life-base-api/app/internal/auth"
	"life-base-api/app/internal/models"
)

// DogService manages dogs, recipient types/containers, bulk food bags, and
// the portioning workflow that ties them together.
type DogService struct {
	dogs           *supabaserepo.DogRepository
	recipientTypes *supabaserepo.DogRecipientTypeRepository
	recipients     *supabaserepo.DogRecipientRepository
	bulkBags       *supabaserepo.DogBulkBagRepository
	settings       *supabaserepo.DogSettingsRepository
}

// NewDogService constructs a DogService.
func NewDogService(
	dogs *supabaserepo.DogRepository,
	recipientTypes *supabaserepo.DogRecipientTypeRepository,
	recipients *supabaserepo.DogRecipientRepository,
	bulkBags *supabaserepo.DogBulkBagRepository,
	settings *supabaserepo.DogSettingsRepository,
) *DogService {
	return &DogService{dogs: dogs, recipientTypes: recipientTypes, recipients: recipients, bulkBags: bulkBags, settings: settings}
}

// ── Dogs ─────────────────────────────────────────────────────────────────────

// ListDogs returns every dog.
func (s *DogService) ListDogs(ctx context.Context) ([]*models.Dog, error) {
	return s.dogs.List(ctx)
}

// CreateDog creates a new dog.
func (s *DogService) CreateDog(ctx context.Context, input models.DogInput) (*models.Dog, error) {
	if input.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if input.MealsPerDay == nil || *input.MealsPerDay <= 0 {
		two := 2
		input.MealsPerDay = &two
	}
	input.UserID = auth.UserIDFromContext(ctx)
	return s.dogs.Create(ctx, input)
}

// UpdateDog patches a dog's fields.
func (s *DogService) UpdateDog(ctx context.Context, id string, input models.DogInput) (*models.Dog, error) {
	return s.dogs.Update(ctx, id, input)
}

// DeleteDog removes a dog.
func (s *DogService) DeleteDog(ctx context.Context, id string) error {
	return s.dogs.Delete(ctx, id)
}

// ── Recipient types ──────────────────────────────────────────────────────────

// ListRecipientTypes returns every recipient type.
func (s *DogService) ListRecipientTypes(ctx context.Context) ([]*models.DogRecipientType, error) {
	return s.recipientTypes.List(ctx)
}

// CreateRecipientType creates a recipient type and provisions its initial
// empty containers.
func (s *DogService) CreateRecipientType(ctx context.Context, input models.DogRecipientTypeInput) (*models.DogRecipientType, error) {
	if input.Name == "" || input.SizeGrams == nil || *input.SizeGrams <= 0 {
		return nil, fmt.Errorf("name and a positive size_grams are required")
	}
	if input.QuantityTotal == nil {
		zero := 0
		input.QuantityTotal = &zero
	}
	userID := auth.UserIDFromContext(ctx)
	input.UserID = userID
	rt, err := s.recipientTypes.Create(ctx, input)
	if err != nil {
		return nil, err
	}
	if err := s.recipients.CreateEmpty(ctx, userID, rt.ID, *input.QuantityTotal); err != nil {
		return nil, fmt.Errorf("provision containers: %w", err)
	}
	return rt, nil
}

// UpdateRecipientType patches a recipient type. Increasing quantity_total
// provisions the delta as new empty containers; decreasing it is not
// supported (existing containers are never removed).
func (s *DogService) UpdateRecipientType(ctx context.Context, id string, input models.DogRecipientTypeInput) (*models.DogRecipientType, error) {
	if input.QuantityTotal != nil {
		existing, err := s.recipientTypes.FindByID(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("find recipient type: %w", err)
		}
		if existing == nil {
			return nil, fmt.Errorf("recipient type not found")
		}
		delta := *input.QuantityTotal - existing.QuantityTotal
		if delta < 0 {
			return nil, fmt.Errorf("quantity_total cannot be decreased (existing containers are never removed)")
		}
		if delta > 0 {
			if err := s.recipients.CreateEmpty(ctx, auth.UserIDFromContext(ctx), id, delta); err != nil {
				return nil, fmt.Errorf("provision additional containers: %w", err)
			}
		}
	}
	return s.recipientTypes.Update(ctx, id, input)
}

// DeleteRecipientType removes a recipient type and its containers.
func (s *DogService) DeleteRecipientType(ctx context.Context, id string) error {
	return s.recipientTypes.Delete(ctx, id)
}

// ── Recipients ───────────────────────────────────────────────────────────────

// ListRecipients returns every recipient, optionally filtered by status
// ("empty" or "portioned").
func (s *DogService) ListRecipients(ctx context.Context, status string) ([]*models.DogRecipient, error) {
	return s.recipients.List(ctx, status)
}

// PortionBatch fills a batch of currently-empty containers, assigning each
// to one dog (or two, if shared), and deducts the corresponding weight from
// bulk bag stock (oldest-frozen first). Validates total bulk stock covers
// the batch before making any change — nothing is partially applied.
func (s *DogService) PortionBatch(ctx context.Context, requests []models.PortionRequest) error {
	if len(requests) == 0 {
		return fmt.Errorf("no containers selected")
	}

	typeCache := make(map[string]*models.DogRecipientType)
	totalGrams := 0
	for _, req := range requests {
		recipient, err := s.recipients.FindByID(ctx, req.RecipientID)
		if err != nil {
			return fmt.Errorf("find recipient %s: %w", req.RecipientID, err)
		}
		if recipient == nil {
			return fmt.Errorf("recipient %s not found", req.RecipientID)
		}
		if recipient.Status != models.DogRecipientEmpty {
			return fmt.Errorf("recipient %s is not empty", req.RecipientID)
		}
		if req.DogID1 == "" {
			return fmt.Errorf("recipient %s needs at least one dog assigned", req.RecipientID)
		}

		rt, ok := typeCache[recipient.RecipientTypeID]
		if !ok {
			rt, err = s.recipientTypes.FindByID(ctx, recipient.RecipientTypeID)
			if err != nil {
				return fmt.Errorf("find recipient type: %w", err)
			}
			if rt == nil {
				return fmt.Errorf("recipient type not found for recipient %s", req.RecipientID)
			}
			typeCache[recipient.RecipientTypeID] = rt
		}
		totalGrams += rt.SizeGrams
	}

	if err := s.deductBulkStock(ctx, totalGrams); err != nil {
		return err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	for _, req := range requests {
		if err := s.recipients.MarkPortioned(ctx, req.RecipientID, req.DogID1, req.DogID2, now); err != nil {
			return fmt.Errorf("mark recipient %s portioned: %w", req.RecipientID, err)
		}
	}
	return nil
}

// deductBulkStock subtracts grams from available bulk bags, oldest-frozen
// first, spilling into the next bag when one is exhausted. Errors (without
// writing anything) if total available stock can't cover it.
func (s *DogService) deductBulkStock(ctx context.Context, grams int) error {
	bags, err := s.bulkBags.ListAvailable(ctx)
	if err != nil {
		return fmt.Errorf("list bulk bags: %w", err)
	}

	remaining := grams
	plan := make(map[string]int, len(bags))
	for _, bag := range bags {
		if remaining <= 0 {
			break
		}
		take := bag.RemainingWeightGrams
		if take > remaining {
			take = remaining
		}
		plan[bag.ID] = bag.RemainingWeightGrams - take
		remaining -= take
	}
	if remaining > 0 {
		return fmt.Errorf("not enough bulk stock: short by %dg", remaining)
	}

	for id, newRemaining := range plan {
		if err := s.bulkBags.SetRemaining(ctx, id, newRemaining); err != nil {
			return fmt.Errorf("update bulk bag %s: %w", id, err)
		}
	}
	return nil
}

// MarkFed recycles a portioned container back to empty.
func (s *DogService) MarkFed(ctx context.Context, id string) error {
	recipient, err := s.recipients.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("find recipient: %w", err)
	}
	if recipient == nil {
		return fmt.Errorf("recipient not found")
	}
	if recipient.Status != models.DogRecipientPortioned {
		return fmt.Errorf("recipient is not portioned")
	}
	return s.recipients.MarkFed(ctx, id)
}

// ── Bulk bags ────────────────────────────────────────────────────────────────

// ListBulkBags returns every bulk bag.
func (s *DogService) ListBulkBags(ctx context.Context) ([]*models.DogBulkBag, error) {
	return s.bulkBags.List(ctx)
}

// CreateBulkBag adds a new bulk bag to stock.
func (s *DogService) CreateBulkBag(ctx context.Context, input models.DogBulkBagInput) (*models.DogBulkBag, error) {
	if input.Label == "" || input.TotalWeightGrams <= 0 {
		return nil, fmt.Errorf("label and a positive total_weight_grams are required")
	}
	if input.FrozenDate == "" {
		input.FrozenDate = time.Now().UTC().Format("2006-01-02")
	}
	input.UserID = auth.UserIDFromContext(ctx)
	return s.bulkBags.Create(ctx, input)
}

// DeleteBulkBag removes a bulk bag.
func (s *DogService) DeleteBulkBag(ctx context.Context, id string) error {
	return s.bulkBags.Delete(ctx, id)
}

// ── Settings ─────────────────────────────────────────────────────────────────

// GetSettings returns the caller's dog settings, defaulting the threshold
// to 0 when unset.
func (s *DogService) GetSettings(ctx context.Context) (*models.DogSettings, error) {
	userID := auth.UserIDFromContext(ctx)
	settings, err := s.settings.FindByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get settings: %w", err)
	}
	if settings == nil {
		return &models.DogSettings{UserID: userID}, nil
	}
	return settings, nil
}

// SetSettings updates the caller's low-stock threshold.
func (s *DogService) SetSettings(ctx context.Context, thresholdGrams int) (*models.DogSettings, error) {
	userID := auth.UserIDFromContext(ctx)
	return s.settings.Upsert(ctx, &models.DogSettings{UserID: userID, LowStockThresholdGrams: thresholdGrams})
}
