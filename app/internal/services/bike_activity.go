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
// previews both go through here). If strava_activity_id is set, refuses to
// create a second activity for the same Strava ride on the same bike.
func (s *ActivityService) Create(ctx context.Context, input models.ActivityInput) (*models.Activity, error) {
	if input.BikeID == "" || input.Date == "" || input.Name == "" || input.Type == "" {
		return nil, fmt.Errorf("bike_id, date, name, and type are required")
	}

	if input.StravaActivityID != nil {
		existing, err := s.activities.FindByStravaActivityID(ctx, input.BikeID, *input.StravaActivityID)
		if err != nil {
			return nil, fmt.Errorf("check for duplicate strava activity: %w", err)
		}
		if existing != nil {
			return nil, fmt.Errorf("strava activity %d was already imported for this bike (activity %s)", *input.StravaActivityID, existing.ID)
		}
	}

	activity, err := s.activities.Create(ctx, input)
	if err != nil {
		return nil, err
	}
	if activity == nil || activity.DistanceKm <= 0 {
		return activity, nil
	}

	s.applyWear(ctx, input.BikeID, activity.DistanceKm)
	return activity, nil
}

// Delete removes the activity and reverses the wear it added — the mirror
// image of Create's accumulation. Reversal is applied against whichever
// components/gear are active *now*, same selection Create used; if a
// component was deactivated or replaced in between, the reversal can't
// perfectly undo what that specific component received.
func (s *ActivityService) Delete(ctx context.Context, id string) error {
	activity, err := s.activities.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("find activity: %w", err)
	}
	if activity == nil {
		return nil
	}

	if err := s.activities.Delete(ctx, id); err != nil {
		return err
	}

	if activity.DistanceKm > 0 {
		s.applyWear(ctx, activity.BikeID, -activity.DistanceKm)
	}
	return nil
}

// applyWear adds deltaKm (negative to reverse) to the bike's mileage and
// every currently-active component's/gear item's accumulated distance.
// Best-effort: a failure on one component/gear item doesn't block the rest.
func (s *ActivityService) applyWear(ctx context.Context, bikeID string, deltaKm float64) {
	if err := s.bikes.IncrementMileage(ctx, bikeID, deltaKm); err != nil {
		return
	}

	if components, err := s.components.ListActiveByBikeID(ctx, bikeID); err == nil {
		for _, c := range components {
			_, _ = s.components.Update(ctx, c.ID, map[string]any{
				"accumulated_km": c.AccumulatedKm + deltaKm,
			})
		}
	}

	if gearItems, err := s.gear.ListActiveByBikeID(ctx, bikeID); err == nil {
		for _, g := range gearItems {
			_, _ = s.gear.Update(ctx, g.ID, map[string]any{
				"distance_km": g.DistanceKm + deltaKm,
			})
		}
	}
}
