package models

// UserPreferences is the stored shape from user_preferences.
type UserPreferences struct {
	UserID             string `json:"user_id,omitempty"`
	PushEnabled        bool   `json:"push_enabled"`
	EmailDigestEnabled bool   `json:"email_digest_enabled"`
	DefaultPage        string `json:"default_page"`
	// Language is the UI language ('en' or 'es'), synced across devices —
	// localStorage still drives the signed-out/first-load default, but once
	// a user has an account this value wins so switching devices keeps
	// their choice.
	Language string `json:"language"`
}
