package services

import (
	"context"
	"fmt"
	"time"

	supabaserepo "life-base-api/app/internal/repositories/supabase"

	"life-base-api/app/internal/auth"
	"life-base-api/app/internal/models"
)

// DogFeedReviewLookbackDays is the fixed lookback window (in days) the
// "did you forget to feed?" auto-review compares expected vs actual feeds
// over — matches the 2-day window requested for the feature.
const DogFeedReviewLookbackDays = 2

// DogService manages dogs, recipient types, portioning allocations, bulk
// food bags, and the feed log that ties them together.
type DogService struct {
	dogs           *supabaserepo.DogRepository
	recipientTypes *supabaserepo.DogRecipientTypeRepository
	allocations    *supabaserepo.DogRecipientAllocationRepository
	feedLog        *supabaserepo.DogFeedLogRepository
	bulkBags       *supabaserepo.DogBulkBagRepository
	settings       *supabaserepo.DogSettingsRepository
}

// NewDogService constructs a DogService.
func NewDogService(
	dogs *supabaserepo.DogRepository,
	recipientTypes *supabaserepo.DogRecipientTypeRepository,
	allocations *supabaserepo.DogRecipientAllocationRepository,
	feedLog *supabaserepo.DogFeedLogRepository,
	bulkBags *supabaserepo.DogBulkBagRepository,
	settings *supabaserepo.DogSettingsRepository,
) *DogService {
	return &DogService{
		dogs: dogs, recipientTypes: recipientTypes, allocations: allocations,
		feedLog: feedLog, bulkBags: bulkBags, settings: settings,
	}
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

// CreateRecipientType creates a recipient type. quantity_total is just the
// physical count of containers owned of this size — no per-container rows
// are provisioned.
func (s *DogService) CreateRecipientType(ctx context.Context, input models.DogRecipientTypeInput) (*models.DogRecipientType, error) {
	if input.Name == "" || input.SizeGrams == nil || *input.SizeGrams <= 0 {
		return nil, fmt.Errorf("name and a positive size_grams are required")
	}
	if input.QuantityTotal == nil {
		zero := 0
		input.QuantityTotal = &zero
	}
	input.UserID = auth.UserIDFromContext(ctx)
	return s.recipientTypes.Create(ctx, input)
}

// UpdateRecipientType patches a recipient type. quantity_total may not be
// set below the number of containers currently allocated (portioned and not
// yet fully fed), since that would make the empty count negative.
func (s *DogService) UpdateRecipientType(ctx context.Context, id string, input models.DogRecipientTypeInput) (*models.DogRecipientType, error) {
	if input.QuantityTotal != nil {
		allocated, err := s.allocatedQuantity(ctx, id)
		if err != nil {
			return nil, err
		}
		if *input.QuantityTotal < allocated {
			return nil, fmt.Errorf("quantity_total cannot be less than %d currently-allocated containers", allocated)
		}
	}
	return s.recipientTypes.Update(ctx, id, input)
}

// DeleteRecipientType removes a recipient type and its allocations.
func (s *DogService) DeleteRecipientType(ctx context.Context, id string) error {
	return s.recipientTypes.Delete(ctx, id)
}

// ── Recipient allocations ────────────────────────────────────────────────────

// ListAllocations returns every recipient allocation.
func (s *DogService) ListAllocations(ctx context.Context) ([]*models.DogRecipientAllocation, error) {
	return s.allocations.List(ctx)
}

// allocatedQuantity sums the quantity of every active allocation for a
// recipient type.
func (s *DogService) allocatedQuantity(ctx context.Context, recipientTypeID string) (int, error) {
	allocations, err := s.allocations.List(ctx)
	if err != nil {
		return 0, fmt.Errorf("list allocations: %w", err)
	}
	total := 0
	for _, a := range allocations {
		if a.RecipientTypeID == recipientTypeID {
			total += a.Quantity
		}
	}
	return total, nil
}

// PortionBatch fills one or more type+quantity+dog assignments, validating
// each against the currently-empty container count for that type, deducts
// the corresponding weight from bulk bag stock (oldest-purchased first),
// and creates one allocation row per request. Validates everything up front
// so nothing is partially applied on a validation failure (bulk-stock
// deduction and allocation creation themselves are best-effort, matching
// the rest of this service).
func (s *DogService) PortionBatch(ctx context.Context, requests []models.PortionRequest) error {
	if len(requests) == 0 {
		return fmt.Errorf("no portions selected")
	}

	allocations, err := s.allocations.List(ctx)
	if err != nil {
		return fmt.Errorf("list allocations: %w", err)
	}
	allocatedByType := make(map[string]int)
	for _, a := range allocations {
		allocatedByType[a.RecipientTypeID] += a.Quantity
	}

	typeCache := make(map[string]*models.DogRecipientType)
	requestedByType := make(map[string]int)
	totalGrams := 0
	for i, req := range requests {
		if req.Quantity <= 0 {
			return fmt.Errorf("request %d: quantity must be positive", i)
		}
		if req.DogID1 == "" {
			return fmt.Errorf("request %d: dog_id_1 is required", i)
		}
		rt, ok := typeCache[req.RecipientTypeID]
		if !ok {
			rt, err = s.recipientTypes.FindByID(ctx, req.RecipientTypeID)
			if err != nil {
				return fmt.Errorf("find recipient type: %w", err)
			}
			if rt == nil {
				return fmt.Errorf("recipient type %s not found", req.RecipientTypeID)
			}
			typeCache[req.RecipientTypeID] = rt
		}
		requestedByType[req.RecipientTypeID] += req.Quantity
		totalGrams += rt.SizeGrams * req.Quantity
	}

	for typeID, requested := range requestedByType {
		empty := typeCache[typeID].QuantityTotal - allocatedByType[typeID]
		if requested > empty {
			return fmt.Errorf("not enough empty containers of type %q: requested %d, %d available", typeCache[typeID].Name, requested, empty)
		}
	}

	if err := s.deductBulkStock(ctx, totalGrams); err != nil {
		return err
	}

	now := time.Now().UTC()
	for _, req := range requests {
		alloc := models.DogRecipientAllocation{
			UserID:          auth.UserIDFromContext(ctx),
			RecipientTypeID: req.RecipientTypeID,
			DogID1:          req.DogID1,
			DogID2:          req.DogID2,
			Quantity:        req.Quantity,
			PortionedAt:     now,
		}
		if _, err := s.allocations.Create(ctx, alloc); err != nil {
			return fmt.Errorf("create allocation: %w", err)
		}
	}
	return nil
}

// deductBulkStock subtracts grams from available bulk bags, oldest-purchased
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

// MarkFed feeds one portion to dogID: it finds that dog's oldest active
// allocation (an allocation dogID is dog_id_1 or dog_id_2 of, with
// quantity > 0), decrements it by one (deleting the row once it hits zero),
// and logs the feed. A shared allocation (dog_id_2 set) is decremented the
// same way regardless of which of the two dogs' buttons triggered the feed
// — it represents one feeding event for both dogs at once, not two.
// Returns false (no error) if the dog has no allocated food available.
func (s *DogService) MarkFed(ctx context.Context, dogID string) (bool, error) {
	alloc, err := s.allocations.FindOldestActiveForDog(ctx, dogID)
	if err != nil {
		return false, fmt.Errorf("find allocation: %w", err)
	}
	if alloc == nil {
		return false, nil
	}

	if alloc.Quantity <= 1 {
		if err := s.allocations.Delete(ctx, alloc.ID); err != nil {
			return false, fmt.Errorf("delete exhausted allocation: %w", err)
		}
	} else {
		if err := s.allocations.SetQuantity(ctx, alloc.ID, alloc.Quantity-1); err != nil {
			return false, fmt.Errorf("decrement allocation: %w", err)
		}
	}

	recipientTypeID := alloc.RecipientTypeID
	entry := models.DogFeedLogEntry{
		UserID:          auth.UserIDFromContext(ctx),
		DogID:           dogID,
		RecipientTypeID: &recipientTypeID,
		FedAt:           time.Now().UTC(),
	}
	if _, err := s.feedLog.Create(ctx, entry); err != nil {
		return false, fmt.Errorf("log feed: %w", err)
	}
	return true, nil
}

// CatchUpFeeds logs up to count missed feeds for dogID by calling MarkFed
// repeatedly, stopping early (without error) if allocated stock runs out.
// Returns how many feeds were actually logged.
func (s *DogService) CatchUpFeeds(ctx context.Context, dogID string, count int) (int, error) {
	logged := 0
	for i := 0; i < count; i++ {
		fed, err := s.MarkFed(ctx, dogID)
		if err != nil {
			return logged, err
		}
		if !fed {
			break
		}
		logged++
	}
	return logged, nil
}

// FeedReview computes, for every dog, the expected feeds over the last
// DogFeedReviewLookbackDays days (meals_per_day * lookback) vs the actual
// count of feed-log rows in that window, returning only dogs with a
// shortfall.
func (s *DogService) FeedReview(ctx context.Context) ([]models.DogFeedReviewEntry, error) {
	dogs, err := s.dogs.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list dogs: %w", err)
	}
	since := time.Now().UTC().AddDate(0, 0, -DogFeedReviewLookbackDays)

	entries := make([]models.DogFeedReviewEntry, 0)
	for _, dog := range dogs {
		actual, err := s.feedLog.CountSince(ctx, dog.ID, since)
		if err != nil {
			return nil, fmt.Errorf("count feed log for %s: %w", dog.Name, err)
		}
		expected := dog.MealsPerDay * DogFeedReviewLookbackDays
		if actual < expected {
			entries = append(entries, models.DogFeedReviewEntry{
				DogID: dog.ID, DogName: dog.Name, Expected: expected, Actual: actual,
			})
		}
	}
	return entries, nil
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
	if input.PurchaseDate == "" {
		input.PurchaseDate = time.Now().UTC().Format("2006-01-02")
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
