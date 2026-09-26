package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"

	supabaserepo "life-base-api/app/internal/repositories/supabase"
)

// resendAPIURL is Resend's transactional email endpoint — a plain REST call
// (Authorization: Bearer <key>, JSON body), not worth a full SDK dependency
// for one call.
const resendAPIURL = "https://api.resend.com/emails"

// DigestConfig holds the VAPID keypair used to sign web push requests, and
// the Resend API key/from-address used for the email digest. Generate the
// VAPID keypair once with webpush.GenerateVAPIDKeys(); the public key is
// also baked into the frontend for PushManager.subscribe.
type DigestConfig struct {
	VAPIDPublicKey  string
	VAPIDPrivateKey string
	VAPIDSubject    string // "mailto:you@example.com" — required by the push spec

	ResendAPIKey string
	ResendFrom   string // e.g. "Life-Base <notifications@yourdomain.com>"
}

// DigestService computes and sends the push/email notification digest.
type DigestService struct {
	reminders ReminderRepository
	prefs     *PreferencesService
	push      *PushService
	admin     *supabaserepo.AdminRepository
	cfg       DigestConfig
	http      *http.Client
}

// NewDigestService constructs a DigestService.
func NewDigestService(reminders ReminderRepository, prefs *PreferencesService, push *PushService, admin *supabaserepo.AdminRepository, cfg DigestConfig) *DigestService {
	return &DigestService{reminders: reminders, prefs: prefs, push: push, admin: admin, cfg: cfg, http: &http.Client{Timeout: 15 * time.Second}}
}

// alertSummary is what changes from "nothing to report" to a sendable
// message — kept small so it can back both a push payload and (in a later
// commit) an email digest body.
type alertSummary struct {
	OverdueCount int
}

func (a alertSummary) empty() bool { return a.OverdueCount == 0 }

func (a alertSummary) message() (title, body string) {
	title = "Life-Base"
	if a.OverdueCount == 1 {
		body = "You have 1 overdue reminder."
	} else {
		body = fmt.Sprintf("You have %d overdue reminders.", a.OverdueCount)
	}
	return title, body
}

func (s *DigestService) computeAlerts(ctx context.Context, userID string) (alertSummary, error) {
	reminders, err := s.reminders.ListByUserID(ctx, userID)
	if err != nil {
		return alertSummary{}, fmt.Errorf("list reminders: %w", err)
	}
	today := time.Now().UTC().Format("2006-01-02")
	var overdue int
	for _, r := range reminders {
		if r.DueDate < today {
			overdue++
		}
	}
	return alertSummary{OverdueCount: overdue}, nil
}

// RunPush sends a push notification to every device of every user with
// push_enabled who currently has something to report. Best-effort per
// user/subscription — one failure doesn't stop the run. Returns the number
// of notifications actually sent.
func (s *DigestService) RunPush(ctx context.Context) (int, error) {
	if s.cfg.VAPIDPublicKey == "" || s.cfg.VAPIDPrivateKey == "" {
		return 0, fmt.Errorf("VAPID keys not configured")
	}

	enabled, err := s.prefs.ListEnabledForPush(ctx)
	if err != nil {
		return 0, fmt.Errorf("list push-enabled users: %w", err)
	}

	sent := 0
	for _, p := range enabled {
		summary, err := s.computeAlerts(ctx, p.UserID)
		if err != nil {
			log.Printf("digest: compute alerts for user=%s: %v", p.UserID, err)
			continue
		}
		if summary.empty() {
			continue
		}

		subs, err := s.push.ListByUserID(ctx, p.UserID)
		if err != nil {
			log.Printf("digest: list subscriptions for user=%s: %v", p.UserID, err)
			continue
		}

		title, body := summary.message()
		payload, err := json.Marshal(map[string]string{"title": title, "body": body})
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
				log.Printf("digest: send push to user=%s: %v", p.UserID, err)
				continue
			}
			_ = resp.Body.Close()
			if resp.StatusCode == 404 || resp.StatusCode == 410 {
				// Subscription is gone (browser data cleared, uninstalled, etc.).
				_ = s.push.Delete(ctx, sub.ID)
				continue
			}
			sent++
		}
	}
	return sent, nil
}

// RunEmail sends a digest email to every user with email_digest_enabled who
// currently has something to report. Best-effort per user.
func (s *DigestService) RunEmail(ctx context.Context) (int, error) {
	if s.cfg.ResendAPIKey == "" || s.cfg.ResendFrom == "" {
		return 0, fmt.Errorf("resend not configured")
	}

	enabled, err := s.prefs.ListEnabledForEmailDigest(ctx)
	if err != nil {
		return 0, fmt.Errorf("list email-digest-enabled users: %w", err)
	}
	if len(enabled) == 0 {
		return 0, nil
	}

	emails, err := s.admin.ListAuthEmails(ctx)
	if err != nil {
		return 0, fmt.Errorf("list auth emails: %w", err)
	}

	sent := 0
	for _, p := range enabled {
		summary, err := s.computeAlerts(ctx, p.UserID)
		if err != nil {
			log.Printf("digest: compute alerts for user=%s: %v", p.UserID, err)
			continue
		}
		if summary.empty() {
			continue
		}

		to := emails[p.UserID]
		if to == "" {
			continue
		}

		_, body := summary.message()
		if err := s.sendEmail(ctx, to, body); err != nil {
			log.Printf("digest: send email to user=%s: %v", p.UserID, err)
			continue
		}
		sent++
	}
	return sent, nil
}

func (s *DigestService) sendEmail(ctx context.Context, to, body string) error {
	payload, err := json.Marshal(map[string]any{
		"from":    s.cfg.ResendFrom,
		"to":      []string{to},
		"subject": "Life-Base: needs your attention",
		"text":    body,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, resendAPIURL, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+s.cfg.ResendAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.http.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("resend: http %d", resp.StatusCode)
	}
	return nil
}
