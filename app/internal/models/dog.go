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
// with the total physical count of that container owned. This total doesn't
// change as containers are portioned/fed — it's just how many physically
// exist; see DogRecipientAllocation for how many are currently filled.
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

// DogRecipientAllocation is N containers of one type, filled and assigned to
// a dog (or a dog pair, when one container is shared between two dogs for a
// single feeding) from one portioning session. DogID2 is set only for a
// shared allocation. The "currently empty" count for a recipient type is its
// quantity_total minus the sum of Quantity across its active allocations.
type DogRecipientAllocation struct {
	ID              string    `json:"id,omitempty"`
	UserID          string    `json:"user_id,omitempty"`
	RecipientTypeID string    `json:"recipient_type_id"`
	DogID1          string    `json:"dog_id_1"`
	DogID2          *string   `json:"dog_id_2,omitempty"`
	Quantity        int       `json:"quantity"`
	PortionedAt     time.Time `json:"portioned_at,omitempty"`
	CreatedAt       time.Time `json:"created_at,omitempty"`
}

// DogFeedLogEntry records one portion actually fed to a dog — used both as
// a feeding history and to reconcile the "did you forget to feed?" review.
type DogFeedLogEntry struct {
	ID              string    `json:"id,omitempty"`
	UserID          string    `json:"user_id,omitempty"`
	DogID           string    `json:"dog_id"`
	RecipientTypeID *string   `json:"recipient_type_id,omitempty"`
	FedAt           time.Time `json:"fed_at,omitempty"`
	CreatedAt       time.Time `json:"created_at,omitempty"`
}

// DogBulkBag is a large raw-food bag — the bulk stock that gets restocked
// and drawn down as batches are portioned.
type DogBulkBag struct {
	ID                   string    `json:"id"`
	UserID               string    `json:"user_id,omitempty"`
	Label                string    `json:"label"`
	TotalWeightGrams     int       `json:"total_weight_grams"`
	RemainingWeightGrams int       `json:"remaining_weight_grams"`
	PurchaseDate         string    `json:"purchase_date"` // YYYY-MM-DD
	Price                *float64  `json:"price,omitempty"`
	CreatedAt            time.Time `json:"created_at,omitempty"`
}

// DogBulkBagInput is the write shape for create.
type DogBulkBagInput struct {
	UserID           string   `json:"user_id,omitempty"`
	Label            string   `json:"label,omitempty"`
	TotalWeightGrams int      `json:"total_weight_grams,omitempty"`
	PurchaseDate     string   `json:"purchase_date,omitempty"`
	Price            *float64 `json:"price,omitempty"`
}

// DogSettings is the per-user singleton row for the low-stock threshold.
type DogSettings struct {
	UserID                 string `json:"user_id,omitempty"`
	LowStockThresholdGrams int    `json:"low_stock_threshold_grams"`
}

// PortionRequest is one recipient-type+quantity+dog assignment to fill in a
// PortionBatch call.
type PortionRequest struct {
	RecipientTypeID string  `json:"recipient_type_id"`
	Quantity        int     `json:"quantity"`
	DogID1          string  `json:"dog_id_1"`
	DogID2          *string `json:"dog_id_2,omitempty"`
}

// DogFeedReviewEntry reports a per-dog shortfall between expected and
// actually-logged feeds over the review lookback window.
type DogFeedReviewEntry struct {
	DogID    string `json:"dog_id"`
	DogName  string `json:"dog_name"`
	Expected int    `json:"expected"`
	Actual   int    `json:"actual"`
}
