package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	supabaserepo "life-base-api/app/internal/repositories/supabase"

	"life-base-api/app/internal/models"
)

// expoPushAPIURL is Expo's push notification endpoint — a plain REST call
// (JSON array of messages, no auth token required for the free tier), not
// worth a full SDK dependency for one call. See
// https://docs.expo.dev/push-notifications/sending-notifications/.
const expoPushAPIURL = "https://exp.host/--/api/v2/push/send"

type expoPushMessage struct {
	To    string `json:"to"`
	Title string `json:"title"`
	Body  string `json:"body"`
}

type expoPushTicket struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Details struct {
		Error string `json:"error"`
	} `json:"details"`
}

type expoPushResponse struct {
	Data []expoPushTicket `json:"data"`
}

// ExpoPushService sends push notifications to ledger-mobile devices via
// Expo's push service — the mobile counterpart to PushService (which
// handles web push/VAPID). Kept separate since the two use unrelated
// delivery mechanisms and credentials.
type ExpoPushService struct {
	repo *supabaserepo.ExpoPushTokenRepository
	http *http.Client
}

// NewExpoPushService constructs an ExpoPushService.
func NewExpoPushService(repo *supabaserepo.ExpoPushTokenRepository) *ExpoPushService {
	return &ExpoPushService{repo: repo, http: &http.Client{Timeout: 15 * time.Second}}
}

// Register upserts a device's push token for a user.
func (s *ExpoPushService) Register(ctx context.Context, userID, token, platform string) error {
	_, err := s.repo.Upsert(ctx, &models.ExpoPushToken{UserID: userID, Token: token, Platform: platform})
	return err
}

// Unregister removes a device's push token.
func (s *ExpoPushService) Unregister(ctx context.Context, token string) error {
	return s.repo.DeleteByToken(ctx, token)
}

// Send pushes a notification to every device a user has registered,
// dropping any token Expo reports as no longer registered. Best-effort:
// errors for individual devices are logged, not returned.
func (s *ExpoPushService) Send(ctx context.Context, userID, title, body string) (int, error) {
	tokens, err := s.repo.ListByUserID(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("list expo push tokens for user=%s: %w", userID, err)
	}
	if len(tokens) == 0 {
		return 0, nil
	}

	messages := make([]expoPushMessage, len(tokens))
	for i, t := range tokens {
		messages[i] = expoPushMessage{To: t.Token, Title: title, Body: body}
	}

	payload, err := json.Marshal(messages)
	if err != nil {
		return 0, fmt.Errorf("marshal expo push messages: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, expoPushAPIURL, bytes.NewReader(payload))
	if err != nil {
		return 0, fmt.Errorf("build expo push request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := s.http.Do(req)
	if err != nil {
		return 0, fmt.Errorf("send expo push: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var parsed expoPushResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return 0, fmt.Errorf("decode expo push response: %w", err)
	}

	sent := 0
	for i, ticket := range parsed.Data {
		if i >= len(tokens) {
			break
		}
		if ticket.Status == "ok" {
			sent++
			continue
		}
		log.Printf("expo push: send to user=%s token=%s: %s", userID, tokens[i].Token, ticket.Message)
		if ticket.Details.Error == "DeviceNotRegistered" {
			_ = s.repo.Delete(ctx, tokens[i].ID)
		}
	}
	return sent, nil
}
