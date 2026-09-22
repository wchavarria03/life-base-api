package models

import "time"

// Bike is the stored shape from bikes.bikes.
type Bike struct {
	ID               string    `json:"id"`
	UserID           string    `json:"user_id,omitempty"`
	Name             string    `json:"name"`
	Type             string    `json:"type"`
	Model            string    `json:"model"`
	Mileage          float64   `json:"mileage"`
	PurchaseDate     *string   `json:"purchase_date,omitempty"`
	PurchaseLocation *string   `json:"purchase_location,omitempty"`
	CreatedAt        time.Time `json:"created_at,omitempty"`
	UpdatedAt        time.Time `json:"updated_at,omitempty"`
}

// BikeInput is the write shape for create/update.
type BikeInput struct {
	UserID           string  `json:"user_id,omitempty"`
	Name             string  `json:"name,omitempty"`
	Type             string  `json:"type,omitempty"`
	Model            string  `json:"model,omitempty"`
	PurchaseDate     *string `json:"purchase_date,omitempty"`
	PurchaseLocation *string `json:"purchase_location,omitempty"`
}

// BikeFitHistory is the stored shape from bikes.bike_fit_history — an
// append-only log of fit sessions for a bike.
type BikeFitHistory struct {
	ID                     string    `json:"id"`
	BikeID                 string    `json:"bike_id"`
	Date                   string    `json:"date"`
	Fitter                 string    `json:"fitter"`
	Location               *string   `json:"location,omitempty"`
	SaddleHeight           *float64  `json:"saddle_height,omitempty"`
	SaddleHeightOverBars   *float64  `json:"saddle_height_over_bars,omitempty"`
	SaddleToHandlebarReach *float64  `json:"saddle_to_handlebar_reach,omitempty"`
	SaddleAngle            *float64  `json:"saddle_angle,omitempty"`
	SaddleForeAft          *float64  `json:"saddle_fore_aft,omitempty"`
	SaddleBrandModel       *string   `json:"saddle_brand_model,omitempty"`
	StemLength             *float64  `json:"stem_length,omitempty"`
	StemAngle              *float64  `json:"stem_angle,omitempty"`
	HandlebarBrandModel    *string   `json:"handlebar_brand_model,omitempty"`
	HandlebarWidth         *float64  `json:"handlebar_width,omitempty"`
	HandlebarAngle         *float64  `json:"handlebar_angle,omitempty"`
	HandlebarExtension     *float64  `json:"handlebar_extension,omitempty"`
	BrakeLeverPosition     *string   `json:"brake_lever_position,omitempty"`
	CrankLength            *float64  `json:"crank_length,omitempty"`
	Chainrings             *string   `json:"chainrings,omitempty"`
	PedalMake              *string   `json:"pedal_make,omitempty"`
	PedalModel             *string   `json:"pedal_model,omitempty"`
	ShoeSize               *string   `json:"shoe_size,omitempty"`
	ShoeMakeModel          *string   `json:"shoe_make_model,omitempty"`
	CleatPosition          *string   `json:"cleat_position,omitempty"`
	Notes                  *string   `json:"notes,omitempty"`
	CreatedAt              time.Time `json:"created_at,omitempty"`
}

// BikeFitHistoryInput is the write shape for create. Fit sessions aren't edited
// after the fact, only appended.
type BikeFitHistoryInput struct {
	BikeID                 string   `json:"bike_id,omitempty"`
	Date                   string   `json:"date,omitempty"`
	Fitter                 string   `json:"fitter,omitempty"`
	Location               *string  `json:"location,omitempty"`
	SaddleHeight           *float64 `json:"saddle_height,omitempty"`
	SaddleHeightOverBars   *float64 `json:"saddle_height_over_bars,omitempty"`
	SaddleToHandlebarReach *float64 `json:"saddle_to_handlebar_reach,omitempty"`
	SaddleAngle            *float64 `json:"saddle_angle,omitempty"`
	SaddleForeAft          *float64 `json:"saddle_fore_aft,omitempty"`
	SaddleBrandModel       *string  `json:"saddle_brand_model,omitempty"`
	StemLength             *float64 `json:"stem_length,omitempty"`
	StemAngle              *float64 `json:"stem_angle,omitempty"`
	HandlebarBrandModel    *string  `json:"handlebar_brand_model,omitempty"`
	HandlebarWidth         *float64 `json:"handlebar_width,omitempty"`
	HandlebarAngle         *float64 `json:"handlebar_angle,omitempty"`
	HandlebarExtension     *float64 `json:"handlebar_extension,omitempty"`
	BrakeLeverPosition     *string  `json:"brake_lever_position,omitempty"`
	CrankLength            *float64 `json:"crank_length,omitempty"`
	Chainrings             *string  `json:"chainrings,omitempty"`
	PedalMake              *string  `json:"pedal_make,omitempty"`
	PedalModel             *string  `json:"pedal_model,omitempty"`
	ShoeSize               *string  `json:"shoe_size,omitempty"`
	ShoeMakeModel          *string  `json:"shoe_make_model,omitempty"`
	CleatPosition          *string  `json:"cleat_position,omitempty"`
	Notes                  *string  `json:"notes,omitempty"`
}
