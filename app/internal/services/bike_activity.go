package services

import (
	"context"
	"fmt"

	"life-base-api/app/internal/models"
)

type ActivityService struct {
	activities ActivityRepository
	bikes      BikeRepository
	components ComponentRepository
	gear       GearRepository
}

func NewActivityService(activities ActivityRepository, bikes BikeRepository, components ComponentRepository, gear GearRepository) *ActivityService {
	return &ActivityService{activities: activities, bikes: bikes, components: components, gear: gear}
}

func (s *ActivityService) ListByBikeID(ctx context.Context, bikeID string) ([]*models.Activity, error) {
	return s.activities.ListByBikeID(ctx, bikeID)
}

// Create saves the activity, then accumulates its distance onto the bike's
// mileage and every active component/gear item on that bike — this is the
// actual wear-tracking write path (manual entries and confirmed Strava
// previews both go through here).
func (s *ActivityService) Create(ctx context.Context, input models.ActivityInput) (*models.Activity, error) {
	if input.BikeID == "" || input.Date == "" || input.Name == "" || input.Type == "" {
		return nil, fmt.Errorf("bike_id, date, name, and type are required")
	}

	activity, err := s.activities.Create(ctx, input)
	if err != nil {
		return nil, err
	}
	if activity == nil || activity.DistanceKm <= 0 {
		return activity, nil
	}

	if err := s.bikes.IncrementMileage(ctx, input.BikeID, activity.DistanceKm); err != nil {
		return activity, fmt.Errorf("activity saved, but failed to update bike mileage: %w", err)
	}

	if components, err := s.components.ListActiveByBikeID(ctx, input.BikeID); err == nil {
		for _, c := range components {
			_, _ = s.components.Update(ctx, c.ID, map[string]any{
				"accumulated_km": c.AccumulatedKm + activity.DistanceKm,
			})
		}
	}

	if gearItems, err := s.gear.ListActiveByBikeID(ctx, input.BikeID); err == nil {
		for _, g := range gearItems {
			_, _ = s.gear.Update(ctx, g.ID, map[string]any{
				"distance_km": g.DistanceKm + activity.DistanceKm,
			})
		}
	}

	return activity, nil
}

func (s *ActivityService) Delete(ctx context.Context, id string) error {
	return s.activities.Delete(ctx, id)
}
