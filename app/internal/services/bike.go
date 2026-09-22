package services

import (
	"context"
	"fmt"

	"life-base-api/app/internal/auth"
	"life-base-api/app/internal/models"
)

type BikeService struct {
	bikes BikeRepository
}

func NewBikeService(bikes BikeRepository) *BikeService {
	return &BikeService{bikes: bikes}
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
