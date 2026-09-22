package models

import "time"

// ServiceLog is the stored shape from bikes.service_logs.
type ServiceLog struct {
	ID          string    `json:"id"`
	BikeID      string    `json:"bike_id"`
	ComponentID *string   `json:"component_id,omitempty"`
	Date        string    `json:"date"`
	Description string    `json:"description"`
	Cost        *float64  `json:"cost,omitempty"`
	Notes       *string   `json:"notes,omitempty"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
}

// ServiceLogInput is the write shape for create.
type ServiceLogInput struct {
	BikeID      string   `json:"bike_id,omitempty"`
	ComponentID *string  `json:"component_id,omitempty"`
	Date        string   `json:"date,omitempty"`
	Description string   `json:"description,omitempty"`
	Cost        *float64 `json:"cost,omitempty"`
	Notes       *string  `json:"notes,omitempty"`
}

// MaintenanceTask is the stored shape from bikes.maintenance_tasks.
type MaintenanceTask struct {
	ID                string    `json:"id"`
	BikeID            string    `json:"bike_id"`
	Name              string    `json:"name"`
	Description       *string   `json:"description,omitempty"`
	IntervalDays      *int      `json:"interval_days,omitempty"`
	IntervalKm        *int      `json:"interval_km,omitempty"`
	LastCompletedDate *string   `json:"last_completed_date,omitempty"`
	LastCompletedKm   *float64  `json:"last_completed_km,omitempty"`
	IsRecurring       bool      `json:"is_recurring"`
	Priority          string    `json:"priority"`
	Notes             *string   `json:"notes,omitempty"`
	CreatedAt         time.Time `json:"created_at,omitempty"`
	UpdatedAt         time.Time `json:"updated_at,omitempty"`
}

// MaintenanceTaskInput is the write shape for create/update.
type MaintenanceTaskInput struct {
	BikeID            string   `json:"bike_id,omitempty"`
	Name              string   `json:"name,omitempty"`
	Description       *string  `json:"description,omitempty"`
	IntervalDays      *int     `json:"interval_days,omitempty"`
	IntervalKm        *int     `json:"interval_km,omitempty"`
	LastCompletedDate *string  `json:"last_completed_date,omitempty"`
	LastCompletedKm   *float64 `json:"last_completed_km,omitempty"`
	IsRecurring       *bool    `json:"is_recurring,omitempty"`
	Priority          string   `json:"priority,omitempty"`
	Notes             *string  `json:"notes,omitempty"`
}

// MaintenanceTaskWithStatus enriches MaintenanceTask with a derived due flag,
// computed from interval_days/interval_km against last_completed_date and the
// bike's current mileage (passed in by the service, not stored).
type MaintenanceTaskWithStatus struct {
	MaintenanceTask
	IsDue bool `json:"is_due"`
}
