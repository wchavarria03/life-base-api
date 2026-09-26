package models

// HouseholdMember is a household_members row, joined with the user's email
// from Supabase auth for display in the admin dashboard.
type HouseholdMember struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	Email  string `json:"email,omitempty"`
}

// PageAccessEntry is one (role, page_key) cell of the access matrix.
type PageAccessEntry struct {
	Role    string `json:"role"`
	PageKey string `json:"page_key"`
	Allowed bool   `json:"allowed"`
}
