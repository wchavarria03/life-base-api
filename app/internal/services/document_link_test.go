package services

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"life-base-api/app/internal/models"
)

const testDocID = "0b1c2d3e-1111-4222-8333-444455556666"

func newSigningService(secret string) *DocumentService {
	return &DocumentService{signingSecret: []byte(secret)}
}

func TestValidDocumentEntity(t *testing.T) {
	for _, ok := range []string{"note", "task_log", "bike", "component", "medical_record"} {
		if !validDocumentEntity(ok) {
			t.Errorf("%q should be valid", ok)
		}
	}
	for _, bad := range []string{"", "Note", "document", "note; drop table"} {
		if validDocumentEntity(bad) {
			t.Errorf("%q should be invalid", bad)
		}
	}
}

func TestUUIDPattern(t *testing.T) {
	if !uuidPattern.MatchString(testDocID) {
		t.Error("valid uuid rejected")
	}
	for _, bad := range []string{"", "not-a-uuid", testDocID + ",id.eq.1", "in.(" + testDocID + ")"} {
		if uuidPattern.MatchString(bad) {
			t.Errorf("%q should be rejected", bad)
		}
	}
}

func TestSignedURLTTL(t *testing.T) {
	s := newSigningService("secret")
	now := time.Unix(1_700_000_000, 0)

	plain := s.signDocument(&models.Document{ID: testDocID}, now)
	if got := plain.ExpiresAt.Sub(now); got != signedURLTTL {
		t.Errorf("plain ttl = %v, want %v", got, signedURLTTL)
	}
	sensitive := s.signDocument(&models.Document{ID: testDocID, Sensitive: true}, now)
	if got := sensitive.ExpiresAt.Sub(now); got != sensitiveSignedURLTTL {
		t.Errorf("sensitive ttl = %v, want %v", got, sensitiveSignedURLTTL)
	}
	if sensitiveSignedURLTTL >= signedURLTTL {
		t.Error("sensitive ttl must be shorter")
	}
}

func TestVerifySignature(t *testing.T) {
	s := newSigningService("secret")
	now := time.Unix(1_700_000_000, 0)
	exp := strconv.FormatInt(now.Add(time.Minute).Unix(), 10)
	sig := s.docSignature(testDocID, exp)

	if err := s.verifySignature(testDocID, exp, sig, now); err != nil {
		t.Fatalf("valid signature rejected: %v", err)
	}
	tests := []struct {
		name string
		id   string
		exp  string
		sig  string
		now  time.Time
	}{
		{"expired", testDocID, exp, sig, now.Add(2 * time.Minute)},
		{"wrong id", "other", exp, sig, now},
		{"tampered exp", testDocID, exp + "0", sig, now},
		{"bad sig", testDocID, exp, strings.Repeat("0", len(sig)), now},
		{"non-numeric exp", testDocID, "abc", sig, now},
	}
	for _, tt := range tests {
		if err := s.verifySignature(tt.id, tt.exp, tt.sig, tt.now); !errors.Is(err, ErrInvalidDocSignature) {
			t.Errorf("%s: err = %v, want ErrInvalidDocSignature", tt.name, err)
		}
	}
	if err := newSigningService("").verifySignature(testDocID, exp, sig, now); !errors.Is(err, ErrSigningDisabled) {
		t.Errorf("empty secret: err = %v, want ErrSigningDisabled", err)
	}
	if err := newSigningService("other").verifySignature(testDocID, exp, sig, now); !errors.Is(err, ErrInvalidDocSignature) {
		t.Errorf("different secret must not verify, got %v", err)
	}
}

func TestShareableAttachmentsDisabledWithoutSecret(t *testing.T) {
	if got := newSigningService("").shareableAttachments(context.Background(), testDocID); got != nil {
		t.Errorf("expected nil without signing secret, got %v", got)
	}
}
