package models

// UserMenuPreference is the stored shape from user_menu_preferences: whether
// a given page_key is hidden from this user's own nav.
type UserMenuPreference struct {
	UserID  string `json:"user_id,omitempty"`
	PageKey string `json:"page_key"`
	Hidden  bool   `json:"hidden"`
}
