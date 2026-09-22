package services

import (
	"context"
	"fmt"
	"time"

	"life-base-api/app/internal/auth"
	"life-base-api/app/internal/models"
)

type GearService struct {
	gear GearRepository
}

func NewGearService(gear GearRepository) *GearService {
	return &GearService{gear: gear}
}

func (s *GearService) List(ctx context.Context) ([]*models.Gear, error) {
	return s.gear.List(ctx)
}

func (s *GearService) Create(ctx context.Context, input models.GearInput) (*models.Gear, error) {
	userID := auth.UserIDFromContext(ctx)
	if userID == "" {
		return nil, fmt.Errorf("no authenticated user")
	}
	if input.Name == "" || input.Category == "" {
		return nil, fmt.Errorf("name and category are required")
	}
	return s.gear.Create(ctx, input)
}

func (s *GearService) Update(ctx context.Context, id string, fields map[string]any) (*models.Gear, error) {
	if len(fields) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}
	return s.gear.Update(ctx, id, fields)
}

func (s *GearService) Delete(ctx context.Context, id string) error {
	return s.gear.Delete(ctx, id)
}

type BottleService struct {
	bottles BottleRepository
}

func NewBottleService(bottles BottleRepository) *BottleService {
	return &BottleService{bottles: bottles}
}

func (s *BottleService) List(ctx context.Context) ([]models.BottleWithStatus, error) {
	bottles, err := s.bottles.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list bottles: %w", err)
	}
	result := make([]models.BottleWithStatus, 0, len(bottles))
	for _, b := range bottles {
		status := models.BottleWithStatus{Bottle: *b}
		if cleaned, err := time.Parse("2006-01-02", b.LastCleanedDate); err == nil {
			dueDate := cleaned.AddDate(0, 0, b.CleaningIntervalDays)
			status.IsDue = !time.Now().Before(dueDate)
		}
		result = append(result, status)
	}
	return result, nil
}

func (s *BottleService) Create(ctx context.Context, input models.BottleInput) (*models.Bottle, error) {
	if input.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	return s.bottles.Create(ctx, input)
}

// MarkCleaned resets last_cleaned_date to today.
func (s *BottleService) MarkCleaned(ctx context.Context, id string) (*models.Bottle, error) {
	return s.bottles.Update(ctx, id, map[string]any{
		"last_cleaned_date": time.Now().Format("2006-01-02"),
	})
}

func (s *BottleService) Update(ctx context.Context, id string, fields map[string]any) (*models.Bottle, error) {
	if len(fields) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}
	return s.bottles.Update(ctx, id, fields)
}

func (s *BottleService) Delete(ctx context.Context, id string) error {
	return s.bottles.Delete(ctx, id)
}

type SupplyService struct {
	supplies SupplyRepository
	history  SupplyHistoryRepository
}

func NewSupplyService(supplies SupplyRepository, history SupplyHistoryRepository) *SupplyService {
	return &SupplyService{supplies: supplies, history: history}
}

func (s *SupplyService) List(ctx context.Context) ([]*models.Supply, error) {
	return s.supplies.List(ctx)
}

func (s *SupplyService) Create(ctx context.Context, input models.SupplyInput) (*models.Supply, error) {
	if input.Name == "" || input.Category == "" || input.PurchaseLocation == "" {
		return nil, fmt.Errorf("name, category, and purchase_location are required")
	}
	return s.supplies.Create(ctx, input)
}

func (s *SupplyService) Update(ctx context.Context, id string, fields map[string]any) (*models.Supply, error) {
	if len(fields) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}
	return s.supplies.Update(ctx, id, fields)
}

// Deplete logs the supply to supply_history and removes it from active
// supplies — mirrors the reference app's "depleted" flow.
func (s *SupplyService) Deplete(ctx context.Context, id string) (*models.SupplyHistory, error) {
	supply, err := s.supplies.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("find supply: %w", err)
	}
	if supply == nil {
		return nil, fmt.Errorf("supply not found")
	}

	entry, err := s.history.Create(ctx, models.SupplyHistoryInput{
		SupplyName:       supply.Name,
		Category:         supply.Category,
		PurchaseDate:     supply.PurchaseDate,
		DepletedDate:     time.Now().Format("2006-01-02"),
		PurchaseLocation: supply.PurchaseLocation,
		PurchaseURL:      supply.PurchaseURL,
		Cost:             &supply.Cost,
		Notes:            supply.Notes,
	})
	if err != nil {
		return nil, fmt.Errorf("log depletion: %w", err)
	}

	if err := s.supplies.Delete(ctx, id); err != nil {
		return nil, fmt.Errorf("remove depleted supply: %w", err)
	}
	return entry, nil
}

func (s *SupplyService) Delete(ctx context.Context, id string) error {
	return s.supplies.Delete(ctx, id)
}

func (s *SupplyService) ListHistory(ctx context.Context) ([]*models.SupplyHistory, error) {
	return s.history.List(ctx)
}
