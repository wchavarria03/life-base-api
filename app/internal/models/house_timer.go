package models

import "time"

// HouseTimer is the stored shape from house_timers — a one-off countdown
// started from the dashboard (e.g. a wall tablet), announced when it fires.
type HouseTimer struct {
	ID        string    `json:"id,omitempty"`
	UserID    string    `json:"user_id,omitempty"`
	Label     string    `json:"label"`
	Message   string    `json:"message"`
	FireAt    time.Time `json:"fire_at"`
	Announced bool      `json:"announced"`
	Notified  bool      `json:"notified"`
	CreatedAt time.Time `json:"created_at,omitempty"`
}

// HouseTimerInput is the write shape for Create.
type HouseTimerInput struct {
	UserID  string    `json:"user_id,omitempty"`
	Label   string    `json:"label"`
	Message string    `json:"message"`
	FireAt  time.Time `json:"fire_at"`
}
