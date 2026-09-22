package models

import "time"

// Gear is the stored shape from bikes.gear.
type Gear struct {
	ID               string    `json:"id"`
	UserID           string    `json:"user_id,omitempty"`
	BikeID           *string   `json:"bike_id,omitempty"`
	Name             string    `json:"name"`
	Category         string    `json:"category"`
	Brand            *string   `json:"brand,omitempty"`
	Model            *string   `json:"model,omitempty"`
	PurchaseDate     *string   `json:"purchase_date,omitempty"`
	PurchaseLocation *string   `json:"purchase_location,omitempty"`
	Cost             *float64  `json:"cost,omitempty"`
	DistanceKm       float64   `json:"distance_km"`
	MaxDistanceKm    *float64  `json:"max_distance_km,omitempty"`
	IsActive         bool      `json:"is_active"`
	Notes            *string   `json:"notes,omitempty"`
	CreatedAt        time.Time `json:"created_at,omitempty"`
	UpdatedAt        time.Time `json:"updated_at,omitempty"`
}

// GearInput is the write shape for create/update.
type GearInput struct {
	BikeID           *string  `json:"bike_id,omitempty"`
	Name             string   `json:"name,omitempty"`
	Category         string   `json:"category,omitempty"`
	Brand            *string  `json:"brand,omitempty"`
	Model            *string  `json:"model,omitempty"`
	PurchaseDate     *string  `json:"purchase_date,omitempty"`
	PurchaseLocation *string  `json:"purchase_location,omitempty"`
	Cost             *float64 `json:"cost,omitempty"`
	MaxDistanceKm    *float64 `json:"max_distance_km,omitempty"`
	IsActive         *bool    `json:"is_active,omitempty"`
	Notes            *string  `json:"notes,omitempty"`
}

// Bottle is the stored shape from bikes.bottles.
type Bottle struct {
	ID                   string    `json:"id"`
	UserID               string    `json:"user_id,omitempty"`
	Name                 string    `json:"name"`
	LastCleanedDate      string    `json:"last_cleaned_date"`
	CleaningIntervalDays int       `json:"cleaning_interval_days"`
	CreatedAt            time.Time `json:"created_at,omitempty"`
	UpdatedAt            time.Time `json:"updated_at,omitempty"`
}

// BottleInput is the write shape for create/update.
type BottleInput struct {
	Name                 string `json:"name,omitempty"`
	LastCleanedDate      string `json:"last_cleaned_date,omitempty"`
	CleaningIntervalDays *int   `json:"cleaning_interval_days,omitempty"`
}

// BottleWithStatus enriches Bottle with a derived due-for-cleaning flag.
type BottleWithStatus struct {
	Bottle
	IsDue bool `json:"is_due"`
}

// Supply is the stored shape from bikes.supplies.
type Supply struct {
	ID                     string    `json:"id"`
	UserID                 string    `json:"user_id,omitempty"`
	Name                   string    `json:"name"`
	Category               string    `json:"category"`
	PurchaseDate           string    `json:"purchase_date"`
	PurchaseLocation       string    `json:"purchase_location"`
	PurchaseURL            *string   `json:"purchase_url,omitempty"`
	Cost                   float64   `json:"cost"`
	EstimatedLifespanDays  *int      `json:"estimated_lifespan_days,omitempty"`
	CurrentQuantityPercent int       `json:"current_quantity_percent"`
	Notes                  *string   `json:"notes,omitempty"`
	CreatedAt              time.Time `json:"created_at,omitempty"`
	UpdatedAt              time.Time `json:"updated_at,omitempty"`
}

// SupplyInput is the write shape for create/update.
type SupplyInput struct {
	Name                   string   `json:"name,omitempty"`
	Category               string   `json:"category,omitempty"`
	PurchaseDate           string   `json:"purchase_date,omitempty"`
	PurchaseLocation       string   `json:"purchase_location,omitempty"`
	PurchaseURL            *string  `json:"purchase_url,omitempty"`
	Cost                   *float64 `json:"cost,omitempty"`
	EstimatedLifespanDays  *int     `json:"estimated_lifespan_days,omitempty"`
	CurrentQuantityPercent *int     `json:"current_quantity_percent,omitempty"`
	Notes                  *string  `json:"notes,omitempty"`
}

// SupplyHistory is the stored shape from bikes.supply_history — created when
// a supply is marked depleted (the supplies row is then typically deleted).
type SupplyHistory struct {
	ID               string    `json:"id"`
	UserID           string    `json:"user_id,omitempty"`
	SupplyName       string    `json:"supply_name"`
	Category         string    `json:"category"`
	PurchaseDate     string    `json:"purchase_date"`
	DepletedDate     string    `json:"depleted_date"`
	PurchaseLocation string    `json:"purchase_location"`
	PurchaseURL      *string   `json:"purchase_url,omitempty"`
	Cost             float64   `json:"cost"`
	Notes            *string   `json:"notes,omitempty"`
	CreatedAt        time.Time `json:"created_at,omitempty"`
}

// SupplyHistoryInput is the write shape for create.
type SupplyHistoryInput struct {
	SupplyName       string   `json:"supply_name,omitempty"`
	Category         string   `json:"category,omitempty"`
	PurchaseDate     string   `json:"purchase_date,omitempty"`
	DepletedDate     string   `json:"depleted_date,omitempty"`
	PurchaseLocation string   `json:"purchase_location,omitempty"`
	PurchaseURL      *string  `json:"purchase_url,omitempty"`
	Cost             *float64 `json:"cost,omitempty"`
	Notes            *string  `json:"notes,omitempty"`
}
