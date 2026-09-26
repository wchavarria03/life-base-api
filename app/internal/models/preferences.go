package models

// UserPreferences is the stored shape from user_preferences.
type UserPreferences struct {
	UserID             string `json:"user_id,omitempty"`
	PushEnabled        bool   `json:"push_enabled"`
	EmailDigestEnabled bool   `json:"email_digest_enabled"`
}
