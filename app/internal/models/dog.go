package models

import "time"

// Dog is the stored shape from dogs.
type Dog struct {
	ID                     string    `json:"id"`
	UserID                 string    `json:"user_id,omitempty"`
	Name                   string    `json:"name"`
	MealsPerDay            int       `json:"meals_per_day"`
	DefaultRecipientTypeID *string   `json:"default_recipient_type_id,omitempty"`
	Notes                  *string   `json:"notes,omitempty"`
	CreatedAt              time.Time `json:"created_at,omitempty"`
}

// DogInput is the write shape for Dog create/update.
type DogInput struct {
	UserID                 string  `json:"user_id,omitempty"`
	Name                   string  `json:"name,omitempty"`
	MealsPerDay            *int    `json:"meals_per_day,omitempty"`
	DefaultRecipientTypeID *string `json:"default_recipient_type_id,omitempty"`
	Notes                  *string `json:"notes,omitempty"`
}

// DogRecipientType is a reusable container size (e.g. "Large cup", 250g),
// with the total physical count of that container owned.
type DogRecipientType struct {
	ID            string    `json:"id"`
	UserID        string    `json:"user_id,omitempty"`
	Name          string    `json:"name"`
	SizeGrams     int       `json:"size_grams"`
	QuantityTotal int       `json:"quantity_total"`
	CreatedAt     time.Time `json:"created_at,omitempty"`
}

// DogRecipientTypeInput is the write shape for create/update.
type DogRecipientTypeInput struct {
	UserID        string `json:"user_id,omitempty"`
	Name          string `json:"name,omitempty"`
	SizeGrams     *int   `json:"size_grams,omitempty"`
	QuantityTotal *int   `json:"quantity_total,omitempty"`
}

// DogRecipientStatus is the lifecycle state of one physical container.
type DogRecipientStatus string

const (
	// DogRecipientEmpty means the container is washed and ready to be filled.
	DogRecipientEmpty DogRecipientStatus = "empty"
	// DogRecipientPortioned means it's filled, frozen, and assigned to a dog.
	DogRecipientPortioned DogRecipientStatus = "portioned"
)

// DogRecipient is one physical container, cycling between empty and
// portioned. DogID2 is set only when the container is shared between two
// dogs for one feeding.
type DogRecipient struct {
	ID              string             `json:"id"`
	UserID          string             `json:"user_id,omitempty"`
	RecipientTypeID string             `json:"recipient_type_id"`
	Status          DogRecipientStatus `json:"status"`
	DogID1          *string            `json:"dog_id_1,omitempty"`
	DogID2          *string            `json:"dog_id_2,omitempty"`
	PortionedAt     *time.Time         `json:"portioned_at,omitempty"`
	CreatedAt       time.Time          `json:"created_at,omitempty"`
}

// DogBulkBag is a large frozen raw-food bag — the bulk stock that gets
// restocked and drawn down as batches are portioned.
type DogBulkBag struct {
	ID                   string    `json:"id"`
	UserID               string    `json:"user_id,omitempty"`
	Label                string    `json:"label"`
	TotalWeightGrams     int       `json:"total_weight_grams"`
	RemainingWeightGrams int       `json:"remaining_weight_grams"`
	FrozenDate           string    `json:"frozen_date"` // YYYY-MM-DD
	CreatedAt            time.Time `json:"created_at,omitempty"`
}

// DogBulkBagInput is the write shape for create.
type DogBulkBagInput struct {
	UserID           string `json:"user_id,omitempty"`
	Label            string `json:"label,omitempty"`
	TotalWeightGrams int    `json:"total_weight_grams,omitempty"`
	FrozenDate       string `json:"frozen_date,omitempty"`
}

// DogSettings is the per-user singleton row for the low-stock threshold.
type DogSettings struct {
	UserID                 string `json:"user_id,omitempty"`
	LowStockThresholdGrams int    `json:"low_stock_threshold_grams"`
}

// PortionRequest is one container to fill in a PortionBatch call.
type PortionRequest struct {
	RecipientID string  `json:"recipient_id"`
	DogID1      string  `json:"dog_id_1"`
	DogID2      *string `json:"dog_id_2,omitempty"`
}
