package services

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"life-base-api/app/internal/auth"
	"life-base-api/app/internal/models"
)

const stravaTokenURL = "https://www.strava.com/oauth/token"
const stravaAuthorizeURL = "https://www.strava.com/oauth/authorize"
const stravaActivitiesURL = "https://www.strava.com/api/v3/athlete/activities"

// oauthStateTTL is how long a generated CSRF state token is valid for.
const oauthStateTTL = 5 * time.Minute

// tokenRefreshMargin: refresh if the access token expires within this window.
const tokenRefreshMargin = 5 * time.Minute

type StravaConfig struct {
	ClientID      string
	ClientSecret  string
	EncryptionKey string
}

type StravaService struct {
	repo       StravaRepository
	cfg        StravaConfig
	httpClient *http.Client
}

func NewStravaService(repo StravaRepository, cfg StravaConfig) *StravaService {
	return &StravaService{repo: repo, cfg: cfg, httpClient: &http.Client{Timeout: 30 * time.Second}}
}

// Authorize generates a CSRF state token, stores it, and returns the Strava
// authorization URL the frontend should redirect the user to.
func (s *StravaService) Authorize(ctx context.Context, redirectURI string) (authURL string, err error) {
	userID := auth.UserIDFromContext(ctx)
	if userID == "" {
		return "", fmt.Errorf("no authenticated user")
	}
	if s.cfg.ClientID == "" {
		return "", fmt.Errorf("STRAVA_CLIENT_ID is not configured")
	}
	if redirectURI == "" {
		return "", fmt.Errorf("redirect_uri is required")
	}

	state, err := randomState()
	if err != nil {
		return "", fmt.Errorf("generate state: %w", err)
	}

	if _, err := s.repo.CreateOAuthState(ctx, models.OAuthStateInput{
		UserID:      userID,
		Provider:    "strava",
		State:       state,
		RedirectURI: redirectURI,
		ExpiresAt:   time.Now().Add(oauthStateTTL).UTC().Format(time.RFC3339),
	}); err != nil {
		return "", fmt.Errorf("store oauth state: %w", err)
	}

	u, _ := url.Parse(stravaAuthorizeURL)
	q := u.Query()
	q.Set("client_id", s.cfg.ClientID)
	q.Set("response_type", "code")
	q.Set("redirect_uri", redirectURI)
	q.Set("scope", "read,activity:read_all")
	q.Set("state", state)
	u.RawQuery = q.Encode()

	return u.String(), nil
}

// Callback validates the CSRF state, exchanges the authorization code for
// tokens, encrypts them, and stores the connection.
func (s *StravaService) Callback(ctx context.Context, code, state string) (*models.StravaConnection, error) {
	userID := auth.UserIDFromContext(ctx)
	if userID == "" {
		return nil, fmt.Errorf("no authenticated user")
	}
	if code == "" || state == "" {
		return nil, fmt.Errorf("code and state are required")
	}

	stateRow, err := s.repo.FindOAuthState(ctx, state, userID)
	if err != nil {
		return nil, fmt.Errorf("look up oauth state: %w", err)
	}
	if stateRow == nil {
		return nil, fmt.Errorf("invalid or expired oauth state")
	}
	_ = s.repo.DeleteOAuthState(ctx, stateRow.ID) // one-time use, best-effort cleanup
	if time.Now().After(stateRow.ExpiresAt) {
		return nil, fmt.Errorf("oauth state expired — please try again")
	}

	var tokenData struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresAt    int64  `json:"expires_at"`
		Athlete      struct {
			ID        int64  `json:"id"`
			FirstName string `json:"firstname"`
			LastName  string `json:"lastname"`
		} `json:"athlete"`
	}
	if err := s.exchangeToken(ctx, url.Values{
		"client_id":     {s.cfg.ClientID},
		"client_secret": {s.cfg.ClientSecret},
		"code":          {code},
		"grant_type":    {"authorization_code"},
	}, &tokenData); err != nil {
		return nil, fmt.Errorf("token exchange: %w", err)
	}

	encryptedAccess, err := s.repo.EncryptToken(ctx, tokenData.AccessToken, s.cfg.EncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("encrypt access token: %w", err)
	}
	encryptedRefresh, err := s.repo.EncryptToken(ctx, tokenData.RefreshToken, s.cfg.EncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("encrypt refresh token: %w", err)
	}

	athleteName := fmt.Sprintf("%s %s", tokenData.Athlete.FirstName, tokenData.Athlete.LastName)
	return s.repo.UpsertConnection(ctx, models.StravaConnectionInput{
		UserID:                userID,
		AthleteID:             tokenData.Athlete.ID,
		AthleteName:           &athleteName,
		AccessTokenEncrypted:  encryptedAccess,
		RefreshTokenEncrypted: encryptedRefresh,
		ExpiresAt:             time.Unix(tokenData.ExpiresAt, 0).UTC().Format(time.RFC3339),
	})
}

func (s *StravaService) Status(ctx context.Context) (*models.StravaConnection, error) {
	userID := auth.UserIDFromContext(ctx)
	if userID == "" {
		return nil, fmt.Errorf("no authenticated user")
	}
	return s.repo.FindConnectionByUserID(ctx, userID)
}

func (s *StravaService) Disconnect(ctx context.Context) error {
	userID := auth.UserIDFromContext(ctx)
	if userID == "" {
		return fmt.Errorf("no authenticated user")
	}
	return s.repo.DeleteConnection(ctx, userID)
}

// FetchActivities returns the caller's Strava activities mapped to our
// shape, as a preview — nothing is written to the activities table here
// (matches the reference Edge Function exactly; the caller confirms and
// POSTs individual activities separately via ActivityService.Create).
func (s *StravaService) FetchActivities(ctx context.Context, after, before *time.Time, page, perPage int) ([]models.StravaActivityPreview, error) {
	userID := auth.UserIDFromContext(ctx)
	if userID == "" {
		return nil, fmt.Errorf("no authenticated user")
	}

	accessToken, err := s.validAccessToken(ctx, userID)
	if err != nil {
		return nil, err
	}

	u, _ := url.Parse(stravaActivitiesURL)
	q := u.Query()
	if after != nil {
		q.Set("after", strconv.FormatInt(after.Unix(), 10))
	}
	if before != nil {
		q.Set("before", strconv.FormatInt(before.Unix(), 10))
	}
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 30
	}
	q.Set("page", strconv.Itoa(page))
	q.Set("per_page", strconv.Itoa(perPage))
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("build strava request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("strava request failed: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read strava response: %w", err)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("strava api error %d: %s", resp.StatusCode, body)
	}

	var raw []struct {
		ID                 int64   `json:"id"`
		Name               string  `json:"name"`
		Type               string  `json:"type"`
		SportType          string  `json:"sport_type"`
		Distance           float64 `json:"distance"`
		MovingTime         int     `json:"moving_time"`
		TotalElevationGain float64 `json:"total_elevation_gain"`
		StartDateLocal     string  `json:"start_date_local"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("decode strava response: %w", err)
	}

	previews := make([]models.StravaActivityPreview, 0, len(raw))
	for _, a := range raw {
		sportType := a.SportType
		if sportType == "" {
			sportType = a.Type
		}
		previews = append(previews, models.StravaActivityPreview{
			StravaID:        a.ID,
			Name:            a.Name,
			Type:            mapStravaType(sportType),
			Date:            a.StartDateLocal,
			DistanceKm:      float64(int(a.Distance/10)) / 100, // meters -> km, 2 decimals
			DurationMinutes: a.MovingTime / 60,
			ElevationGain:   a.TotalElevationGain,
			Source:          "Strava",
		})
	}
	return previews, nil
}

// validAccessToken returns a usable access token for the user, refreshing
// against Strava (and re-encrypting the result) if it's expired or close to it.
func (s *StravaService) validAccessToken(ctx context.Context, userID string) (string, error) {
	conn, err := s.repo.FindConnectionByUserID(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("look up strava connection: %w", err)
	}
	if conn == nil {
		return "", fmt.Errorf("no strava connection — connect strava first")
	}

	if time.Now().Add(tokenRefreshMargin).Before(conn.ExpiresAt) {
		return s.repo.DecryptToken(ctx, conn.AccessTokenEncrypted, s.cfg.EncryptionKey)
	}

	refreshToken, err := s.repo.DecryptToken(ctx, conn.RefreshTokenEncrypted, s.cfg.EncryptionKey)
	if err != nil {
		return "", fmt.Errorf("decrypt refresh token: %w", err)
	}

	var tokenData struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresAt    int64  `json:"expires_at"`
	}
	if err := s.exchangeToken(ctx, url.Values{
		"client_id":     {s.cfg.ClientID},
		"client_secret": {s.cfg.ClientSecret},
		"refresh_token": {refreshToken},
		"grant_type":    {"refresh_token"},
	}, &tokenData); err != nil {
		return "", fmt.Errorf("refresh token: %w", err)
	}

	encryptedAccess, err := s.repo.EncryptToken(ctx, tokenData.AccessToken, s.cfg.EncryptionKey)
	if err != nil {
		return "", fmt.Errorf("encrypt refreshed access token: %w", err)
	}
	encryptedRefresh, err := s.repo.EncryptToken(ctx, tokenData.RefreshToken, s.cfg.EncryptionKey)
	if err != nil {
		return "", fmt.Errorf("encrypt refreshed refresh token: %w", err)
	}

	if _, err := s.repo.UpdateConnection(ctx, conn.ID, map[string]any{
		"access_token_encrypted":  encryptedAccess,
		"refresh_token_encrypted": encryptedRefresh,
		"expires_at":              time.Unix(tokenData.ExpiresAt, 0).UTC().Format(time.RFC3339),
	}); err != nil {
		return "", fmt.Errorf("save refreshed tokens: %w", err)
	}

	return tokenData.AccessToken, nil
}

func (s *StravaService) exchangeToken(ctx context.Context, form url.Values, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, stravaTokenURL,
		bytes.NewReader([]byte(form.Encode())))
	if err != nil {
		return fmt.Errorf("build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read token response: %w", err)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("strava token error %d: %s", resp.StatusCode, body)
	}
	return json.Unmarshal(body, out)
}

func randomState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func mapStravaType(stravaType string) string {
	switch stravaType {
	case "Ride", "GravelRide", "MountainBikeRide", "EBikeRide", "Velomobile":
		return "Ride"
	case "VirtualRide", "Workout":
		return "Training"
	case "Race":
		return "Race"
	default:
		return "Ride"
	}
}
