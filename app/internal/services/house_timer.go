package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"

	"life-base-api/app/internal/auth"
	"life-base-api/app/internal/models"
	supabaserepo "life-base-api/app/internal/repositories/supabase"
)

// HouseTimerConfig holds the VAPID keypair used to push-notify a timer's
// owner when it fires — same keys as DigestConfig, kept separate so this
// service doesn't carry Digest's unrelated Resend/email fields.
type HouseTimerConfig struct {
	VAPIDPublicKey  string
	VAPIDPrivateKey string
	VAPIDSubject    string
}

// HouseTimerService manages one-off dashboard timers (e.g. "Laundry — 45
// min") and the push notification sent when one fires.
type HouseTimerService struct {
	repo *supabaserepo.HouseTimerRepository
	push *PushService
	cfg  HouseTimerConfig
	http *http.Client
}

// NewHouseTimerService constructs a HouseTimerService.
func NewHouseTimerService(repo *supabaserepo.HouseTimerRepository, push *PushService, cfg HouseTimerConfig) *HouseTimerService {
	return &HouseTimerService{repo: repo, push: push, cfg: cfg, http: &http.Client{Timeout: 15 * time.Second}}
}

// List returns the caller's recent timers.
func (s *HouseTimerService) List(ctx context.Context) ([]*models.HouseTimer, error) {
	return s.repo.List(ctx)
}

// Create starts a new timer, firing in `minutes` from now.
func (s *HouseTimerService) Create(ctx context.Context, label, message string, minutes int) (*models.HouseTimer, error) {
	userID := auth.UserIDFromContext(ctx)
	if userID == "" {
		return nil, fmt.Errorf("no authenticated user")
	}
	if label == "" {
		return nil, fmt.Errorf("label is required")
	}
	if minutes <= 0 {
		return nil, fmt.Errorf("minutes must be positive")
	}
	return s.repo.Create(ctx, models.HouseTimerInput{
		UserID:  userID,
		Label:   label,
		Message: message,
		FireAt:  time.Now().UTC().Add(time.Duration(minutes) * time.Minute),
	})
}

// MarkAnnounced flags a timer as spoken-aloud (called by whichever device —
// the tablet — was open and caught it firing).
func (s *HouseTimerService) MarkAnnounced(ctx context.Context, id string) error {
	return s.repo.Update(ctx, id, map[string]any{"announced": true})
}

// Delete cancels/dismisses a timer.
func (s *HouseTimerService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

// NotifyDue push-notifies the owner of every timer that has fired but
// hasn't been notified yet (run on a short cron — see cmd). Best-effort per
// timer/subscription, same pattern as DigestService.RunPush.
func (s *HouseTimerService) NotifyDue(ctx context.Context) (int, error) {
	if s.cfg.VAPIDPublicKey == "" || s.cfg.VAPIDPrivateKey == "" {
		return 0, fmt.Errorf("VAPID keys not configured")
	}

	due, err := s.repo.ListDueUnnotified(ctx)
	if err != nil {
		return 0, fmt.Errorf("list due timers: %w", err)
	}

	sent := 0
	for _, t := range due {
		subs, err := s.push.ListByUserID(ctx, t.UserID)
		if err != nil {
			log.Printf("house timer: list subscriptions for user=%s: %v", t.UserID, err)
			continue
		}

		payload, err := json.Marshal(map[string]string{"title": t.Label, "body": t.Message})
		if err != nil {
			continue
		}

		for _, sub := range subs {
			resp, err := webpush.SendNotificationWithContext(ctx, payload, &webpush.Subscription{
				Endpoint: sub.Endpoint,
				Keys:     webpush.Keys{P256dh: sub.P256dh, Auth: sub.AuthKey},
			}, &webpush.Options{
				VAPIDPublicKey:  s.cfg.VAPIDPublicKey,
				VAPIDPrivateKey: s.cfg.VAPIDPrivateKey,
				Subscriber:      s.cfg.VAPIDSubject,
				TTL:             3600,
			})
			if err != nil {
				log.Printf("house timer: send push to user=%s: %v", t.UserID, err)
				continue
			}
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone {
				_ = s.push.Delete(ctx, sub.ID)
				continue
			}
			sent++
		}

		if err := s.repo.Update(ctx, t.ID, map[string]any{"notified": true}); err != nil {
			log.Printf("house timer: mark notified id=%s: %v", t.ID, err)
		}
	}
	return sent, nil
}
