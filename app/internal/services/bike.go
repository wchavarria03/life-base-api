package services

import (
	"context"
	"fmt"

	"life-base-api/app/internal/auth"
	"life-base-api/app/internal/models"
)

type BikeService struct {
	bikes       BikeRepository
	components  *ComponentService
	maintenance *MaintenanceTaskService
}

// NewBikeService constructs a BikeService. components/maintenance back
// DashboardSummary, which aggregates each bike's due components/tasks
// server-side instead of the frontend fetching them bike-by-bike.
func NewBikeService(bikes BikeRepository, components *ComponentService, maintenance *MaintenanceTaskService) *BikeService {
	return &BikeService{bikes: bikes, components: components, maintenance: maintenance}
}

func (s *BikeService) List(ctx context.Context) ([]*models.Bike, error) {
	return s.bikes.List(ctx)
}

func (s *BikeService) FindByID(ctx context.Context, id string) (*models.Bike, error) {
	return s.bikes.FindByID(ctx, id)
}

func (s *BikeService) Create(ctx context.Context, input models.BikeInput) (*models.Bike, error) {
	userID := auth.UserIDFromContext(ctx)
	if userID == "" {
		return nil, fmt.Errorf("no authenticated user")
	}
	if input.Name == "" || input.Type == "" || input.Model == "" {
		return nil, fmt.Errorf("name, type, and model are required")
	}
	input.UserID = userID
	return s.bikes.Create(ctx, input)
}

func (s *BikeService) Update(ctx context.Context, id string, fields map[string]any) (*models.Bike, error) {
	if len(fields) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}
	return s.bikes.Update(ctx, id, fields)
}

func (s *BikeService) Delete(ctx context.Context, id string) error {
	return s.bikes.Delete(ctx, id)
}

// DashboardSummary aggregates bike count and total due components/tasks
// across every bike in one call — replaces the frontend's previous
// 1+2N fetch pattern (list bikes, then per-bike components+tasks) with a
// single round trip; the N+1 still happens, just server-side where the
// latency to Supabase is far cheaper than N+1 browser round trips.
func (s *BikeService) DashboardSummary(ctx context.Context) (*models.BikeDashboardSummary, error) {
	bikes, err := s.bikes.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list bikes: %w", err)
	}

	summary := &models.BikeDashboardSummary{BikeCount: len(bikes)}
	for _, b := range bikes {
		components, err := s.components.ListByBikeID(ctx, b.ID)
		if err != nil {
			return nil, fmt.Errorf("list components for bike %s: %w", b.ID, err)
		}
		for _, c := range components {
			if c.IsDue {
				summary.DueCount++
			}
		}

		tasks, err := s.maintenance.ListByBikeID(ctx, b.ID)
		if err != nil {
			return nil, fmt.Errorf("list maintenance tasks for bike %s: %w", b.ID, err)
		}
		for _, t := range tasks {
			if t.IsDue {
				summary.DueCount++
			}
		}
	}
	return summary, nil
}

type BikeFitHistoryService struct {
	history BikeFitHistoryRepository
}

func NewBikeFitHistoryService(history BikeFitHistoryRepository) *BikeFitHistoryService {
	return &BikeFitHistoryService{history: history}
}

func (s *BikeFitHistoryService) ListByBikeID(ctx context.Context, bikeID string) ([]*models.BikeFitHistory, error) {
	return s.history.ListByBikeID(ctx, bikeID)
}

func (s *BikeFitHistoryService) Create(ctx context.Context, input models.BikeFitHistoryInput) (*models.BikeFitHistory, error) {
	if input.BikeID == "" || input.Date == "" || input.Fitter == "" {
		return nil, fmt.Errorf("bike_id, date, and fitter are required")
	}
	return s.history.Create(ctx, input)
}

func (s *BikeFitHistoryService) Delete(ctx context.Context, id string) error {
	return s.history.Delete(ctx, id)
}
