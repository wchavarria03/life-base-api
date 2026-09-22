package services

import (
	"context"
	"fmt"
	"time"

	"life-base-api/app/internal/models"
)

type ComponentService struct {
	components ComponentRepository
	history    ComponentHistoryRepository
}

func NewComponentService(components ComponentRepository, history ComponentHistoryRepository) *ComponentService {
	return &ComponentService{components: components, history: history}
}

func (s *ComponentService) ListByBikeID(ctx context.Context, bikeID string) ([]models.ComponentWithStatus, error) {
	components, err := s.components.ListByBikeID(ctx, bikeID)
	if err != nil {
		return nil, fmt.Errorf("list components: %w", err)
	}
	result := make([]models.ComponentWithStatus, 0, len(components))
	for _, c := range components {
		result = append(result, withComponentStatus(c))
	}
	return result, nil
}

func (s *ComponentService) Create(ctx context.Context, input models.ComponentInput) (*models.Component, error) {
	if input.BikeID == "" || input.Name == "" || input.LastReplacedDate == "" {
		return nil, fmt.Errorf("bike_id, name, and last_replaced_date are required")
	}
	return s.components.Create(ctx, input)
}

func (s *ComponentService) Update(ctx context.Context, id string, fields map[string]any) (*models.Component, error) {
	if len(fields) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}
	return s.components.Update(ctx, id, fields)
}

func (s *ComponentService) Delete(ctx context.Context, id string) error {
	return s.components.Delete(ctx, id)
}

// Replace logs a replacement in component_history and resets the component's
// last_replaced_date/accumulated_km so wear tracking starts over.
func (s *ComponentService) Replace(ctx context.Context, componentID, replacedDate string, mileageAtReplacement *float64, notes *string) (*models.Component, error) {
	if replacedDate == "" {
		return nil, fmt.Errorf("replaced_date is required")
	}
	if _, err := s.history.Create(ctx, models.ComponentHistoryInput{
		ComponentID:          componentID,
		ReplacedDate:         replacedDate,
		MileageAtReplacement: mileageAtReplacement,
		Notes:                notes,
	}); err != nil {
		return nil, fmt.Errorf("log replacement: %w", err)
	}
	return s.components.Update(ctx, componentID, map[string]any{
		"last_replaced_date": replacedDate,
		"accumulated_km":     0,
	})
}

func (s *ComponentService) ListHistory(ctx context.Context, componentID string) ([]*models.ComponentHistory, error) {
	return s.history.ListByComponentID(ctx, componentID)
}

// withComponentStatus derives wear_percentage and is_due — never stored,
// computed fresh from accumulated_km/replacement_interval_km(/days) each read.
func withComponentStatus(c *models.Component) models.ComponentWithStatus {
	status := models.ComponentWithStatus{Component: *c}

	if c.ReplacementIntervalKm != nil && *c.ReplacementIntervalKm > 0 {
		pct := c.AccumulatedKm / float64(*c.ReplacementIntervalKm) * 100
		if pct > 100 {
			pct = 100
		}
		status.WearPercentage = &pct
		if pct >= 100 {
			status.IsDue = true
		}
	}

	if !status.IsDue && c.ReplacementIntervalDays != nil {
		if replaced, err := time.Parse("2006-01-02", c.LastReplacedDate); err == nil {
			dueDate := replaced.AddDate(0, 0, *c.ReplacementIntervalDays)
			if !time.Now().Before(dueDate) {
				status.IsDue = true
			}
		}
	}

	return status
}
