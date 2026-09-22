package models

import "time"

// Activity is the stored shape from bikes.activities — a ride, manually
// entered or confirmed from a Strava import preview.
type Activity struct {
	ID               string    `json:"id"`
	BikeID           string    `json:"bike_id"`
	Date             string    `json:"date"`
	Name             string    `json:"name"`
	Type             string    `json:"type"`
	DistanceKm       float64   `json:"distance_km"`
	DurationMinutes  *int      `json:"duration_minutes,omitempty"`
	ElevationGain    *float64  `json:"elevation_gain,omitempty"`
	Source           *string   `json:"source,omitempty"`
	StravaActivityID *int64    `json:"strava_activity_id,omitempty"`
	GearIDs          []string  `json:"gear_ids,omitempty"`
	Notes            *string   `json:"notes,omitempty"`
	CreatedAt        time.Time `json:"created_at,omitempty"`
}

// ActivityInput is the write shape for create. Activities aren't edited after
// the fact — correct by deleting and re-creating.
type ActivityInput struct {
	BikeID           string   `json:"bike_id,omitempty"`
	Date             string   `json:"date,omitempty"`
	Name             string   `json:"name,omitempty"`
	Type             string   `json:"type,omitempty"`
	DistanceKm       float64  `json:"distance_km,omitempty"`
	DurationMinutes  *int     `json:"duration_minutes,omitempty"`
	ElevationGain    *float64 `json:"elevation_gain,omitempty"`
	Source           *string  `json:"source,omitempty"`
	StravaActivityID *int64   `json:"strava_activity_id,omitempty"`
	GearIDs          []string `json:"gear_ids,omitempty"`
	Notes            *string  `json:"notes,omitempty"`
}
