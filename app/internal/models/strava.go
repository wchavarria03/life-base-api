package models

import "time"

// StravaConnection is the stored shape from bikes.strava_connections. Tokens
// are never serialized to JSON — callers get connection status (athlete
// name, expiry), never the encrypted token bytes.
type StravaConnection struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id,omitempty"`
	AthleteID   int64     `json:"athlete_id"`
	AthleteName *string   `json:"athlete_name,omitempty"`
	ExpiresAt   time.Time `json:"expires_at"`
	Scope       *string   `json:"scope,omitempty"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
	UpdatedAt   time.Time `json:"updated_at,omitempty"`

	// AccessTokenEncrypted/RefreshTokenEncrypted round-trip through
	// PostgREST as opaque bytea-hex strings — never decoded, only passed
	// back to the decrypt_token_pgp RPC. Excluded from JSON responses.
	AccessTokenEncrypted  string `json:"-"`
	RefreshTokenEncrypted string `json:"-"`
}

// StravaConnectionInput is the write shape for create/update.
type StravaConnectionInput struct {
	UserID                string  `json:"user_id,omitempty"`
	AthleteID             int64   `json:"athlete_id,omitempty"`
	AthleteName           *string `json:"athlete_name,omitempty"`
	AccessTokenEncrypted  string  `json:"access_token_encrypted,omitempty"`
	RefreshTokenEncrypted string  `json:"refresh_token_encrypted,omitempty"`
	ExpiresAt             string  `json:"expires_at,omitempty"`
	Scope                 *string `json:"scope,omitempty"`
}

// OAuthState is the stored shape from bikes.oauth_states — a short-lived
// CSRF token for the OAuth authorize/callback round trip.
type OAuthState struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	Provider    string    `json:"provider"`
	State       string    `json:"state"`
	RedirectURI string    `json:"redirect_uri"`
	ExpiresAt   time.Time `json:"expires_at"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
}

// OAuthStateInput is the write shape for create.
type OAuthStateInput struct {
	UserID      string `json:"user_id,omitempty"`
	Provider    string `json:"provider,omitempty"`
	State       string `json:"state,omitempty"`
	RedirectURI string `json:"redirect_uri,omitempty"`
	ExpiresAt   string `json:"expires_at,omitempty"`
}

// StravaActivityPreview is a Strava activity mapped into our shape, returned
// by FetchActivities for the user to review before confirming — it is not
// persisted until the caller POSTs it as an ActivityInput.
type StravaActivityPreview struct {
	StravaID        int64   `json:"strava_id"`
	Name            string  `json:"name"`
	Type            string  `json:"type"`
	Date            string  `json:"date"`
	DistanceKm      float64 `json:"distance_km"`
	DurationMinutes int     `json:"duration_minutes"`
	ElevationGain   float64 `json:"elevation_gain"`
	Source          string  `json:"source"`
}
