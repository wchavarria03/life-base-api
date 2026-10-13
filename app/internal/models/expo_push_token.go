package models

// ExpoPushToken is the stored shape from expo_push_tokens — one row per
// mobile device (ledger-mobile) a user has granted notification permission
// on.
type ExpoPushToken struct {
	ID       string `json:"id,omitempty"`
	UserID   string `json:"user_id,omitempty"`
	Token    string `json:"token"`
	Platform string `json:"platform"`
}
