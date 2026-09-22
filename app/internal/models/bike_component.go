package models

import "time"

// Component is the stored shape from bikes.components. Wear percentage is
// derived at read time (see ComponentWithStatus), not stored — accumulated_km
// is the raw input, updated as activities are logged against the bike.
type Component struct {
	ID                      string    `json:"id"`
	BikeID                  string    `json:"bike_id"`
	Name                    string    `json:"name"`
	Brand                   *string   `json:"brand,omitempty"`
	Model                   *string   `json:"model,omitempty"`
	IsActive                bool      `json:"is_active"`
	LastReplacedDate        string    `json:"last_replaced_date"`
	ReplacementIntervalKm   *int      `json:"replacement_interval_km,omitempty"`
	ReplacementIntervalDays *int      `json:"replacement_interval_days,omitempty"`
	AccumulatedKm           float64   `json:"accumulated_km"`
	Notes                   *string   `json:"notes,omitempty"`
	CreatedAt               time.Time `json:"created_at,omitempty"`
	UpdatedAt               time.Time `json:"updated_at,omitempty"`
}

// ComponentInput is the write shape for create/update.
type ComponentInput struct {
	BikeID                  string  `json:"bike_id,omitempty"`
	Name                    string  `json:"name,omitempty"`
	Brand                   *string `json:"brand,omitempty"`
	Model                   *string `json:"model,omitempty"`
	IsActive                *bool   `json:"is_active,omitempty"`
	LastReplacedDate        string  `json:"last_replaced_date,omitempty"`
	ReplacementIntervalKm   *int    `json:"replacement_interval_km,omitempty"`
	ReplacementIntervalDays *int    `json:"replacement_interval_days,omitempty"`
	Notes                   *string `json:"notes,omitempty"`
}

// ComponentWithStatus enriches Component with a derived wear percentage
// (accumulated_km / replacement_interval_km, capped at 100) and a due flag —
// same "derive, don't store" pattern as ReminderWithStatus.
type ComponentWithStatus struct {
	Component
	WearPercentage *float64 `json:"wear_percentage,omitempty"`
	IsDue          bool     `json:"is_due"`
}

// ComponentHistory is the stored shape from bikes.component_history — a
// replacement log entry, created when a component is replaced.
type ComponentHistory struct {
	ID                   string    `json:"id"`
	ComponentID          string    `json:"component_id"`
	ReplacedDate         string    `json:"replaced_date"`
	MileageAtReplacement *float64  `json:"mileage_at_replacement,omitempty"`
	Notes                *string   `json:"notes,omitempty"`
	CreatedAt            time.Time `json:"created_at,omitempty"`
}

// ComponentHistoryInput is the write shape for create.
type ComponentHistoryInput struct {
	ComponentID          string   `json:"component_id,omitempty"`
	ReplacedDate         string   `json:"replaced_date,omitempty"`
	MileageAtReplacement *float64 `json:"mileage_at_replacement,omitempty"`
	Notes                *string  `json:"notes,omitempty"`
}
