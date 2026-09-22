package services

import (
	"context"
	"fmt"
	"time"

	"life-base-api/app/internal/models"
)

type ServiceLogService struct {
	logs ServiceLogRepository
}

func NewServiceLogService(logs ServiceLogRepository) *ServiceLogService {
	return &ServiceLogService{logs: logs}
}

func (s *ServiceLogService) ListByBikeID(ctx context.Context, bikeID string) ([]*models.ServiceLog, error) {
	return s.logs.ListByBikeID(ctx, bikeID)
}

func (s *ServiceLogService) Create(ctx context.Context, input models.ServiceLogInput) (*models.ServiceLog, error) {
	if input.BikeID == "" || input.Date == "" || input.Description == "" {
		return nil, fmt.Errorf("bike_id, date, and description are required")
	}
	return s.logs.Create(ctx, input)
}

func (s *ServiceLogService) Delete(ctx context.Context, id string) error {
	return s.logs.Delete(ctx, id)
}

type MaintenanceTaskService struct {
	tasks MaintenanceTaskRepository
	bikes BikeRepository
}

func NewMaintenanceTaskService(tasks MaintenanceTaskRepository, bikes BikeRepository) *MaintenanceTaskService {
	return &MaintenanceTaskService{tasks: tasks, bikes: bikes}
}

func (s *MaintenanceTaskService) ListByBikeID(ctx context.Context, bikeID string) ([]models.MaintenanceTaskWithStatus, error) {
	tasks, err := s.tasks.ListByBikeID(ctx, bikeID)
	if err != nil {
		return nil, fmt.Errorf("list maintenance tasks: %w", err)
	}

	var mileage float64
	if bike, err := s.bikes.FindByID(ctx, bikeID); err == nil && bike != nil {
		mileage = bike.Mileage
	}

	result := make([]models.MaintenanceTaskWithStatus, 0, len(tasks))
	for _, t := range tasks {
		result = append(result, withMaintenanceStatus(t, mileage))
	}
	return result, nil
}

func (s *MaintenanceTaskService) Create(ctx context.Context, input models.MaintenanceTaskInput) (*models.MaintenanceTask, error) {
	if input.BikeID == "" || input.Name == "" {
		return nil, fmt.Errorf("bike_id and name are required")
	}
	return s.tasks.Create(ctx, input)
}

func (s *MaintenanceTaskService) Update(ctx context.Context, id string, fields map[string]any) (*models.MaintenanceTask, error) {
	if len(fields) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}
	return s.tasks.Update(ctx, id, fields)
}

func (s *MaintenanceTaskService) Delete(ctx context.Context, id string) error {
	return s.tasks.Delete(ctx, id)
}

// Complete marks a maintenance task done today at the bike's current mileage
// — the reference for the next due-ness computation.
func (s *MaintenanceTaskService) Complete(ctx context.Context, id, bikeID string) (*models.MaintenanceTask, error) {
	fields := map[string]any{
		"last_completed_date": time.Now().Format("2006-01-02"),
	}
	if bike, err := s.bikes.FindByID(ctx, bikeID); err == nil && bike != nil {
		fields["last_completed_km"] = bike.Mileage
	}
	return s.tasks.Update(ctx, id, fields)
}

func withMaintenanceStatus(t *models.MaintenanceTask, currentMileage float64) models.MaintenanceTaskWithStatus {
	status := models.MaintenanceTaskWithStatus{MaintenanceTask: *t}

	if t.LastCompletedDate == nil {
		// Never completed — always due.
		status.IsDue = true
		return status
	}

	if t.IntervalDays != nil {
		if completed, err := time.Parse("2006-01-02", *t.LastCompletedDate); err == nil {
			dueDate := completed.AddDate(0, 0, *t.IntervalDays)
			if !time.Now().Before(dueDate) {
				status.IsDue = true
			}
		}
	}

	if !status.IsDue && t.IntervalKm != nil && t.LastCompletedKm != nil {
		if currentMileage-*t.LastCompletedKm >= float64(*t.IntervalKm) {
			status.IsDue = true
		}
	}

	return status
}
