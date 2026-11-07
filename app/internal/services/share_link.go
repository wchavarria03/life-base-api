package services

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	"life-base-api/app/internal/auth"
	"life-base-api/app/internal/models"
	supabaserepo "life-base-api/app/internal/repositories/supabase"
	"life-base-api/app/internal/vault"
)

// sensitiveShareMaxTTL caps how long a link to a sensitive-flagged note may live.
const sensitiveShareMaxTTL = 7 * 24 * time.Hour

// ErrSensitiveConfirmRequired is returned when a link to a sensitive note is
// requested without confirm_sensitive: true.
var ErrSensitiveConfirmRequired = errors.New("this note is flagged sensitive: confirm_sensitive must be true")

// shareTokenAlphabet avoids visually-ambiguous characters (0/O, 1/I/l) so a
// short link reads back cleanly if someone has to type it.
const shareTokenAlphabet = "23456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnpqrstuvwxyz"
const shareTokenLength = 10

// ErrShareLinkNotFound covers an unknown, revoked, or expired token —
// deliberately not distinguished to the public caller (nothing to learn by
// telling them which).
var ErrShareLinkNotFound = errors.New("share link not found")

// ShareLinkService creates and resolves public, read-only share links for
// notes and bikes.
type ShareLinkService struct {
	repo            *supabaserepo.ShareLinkRepository
	notes           *NoteService
	bikes           *BikeService
	components      *ComponentService
	medicalProfiles *MedicalProfileService
	medicalRecords  *MedicalRecordService
	audit           *AuditLogService
}

// NewShareLinkService constructs a ShareLinkService.
func NewShareLinkService(repo *supabaserepo.ShareLinkRepository, notes *NoteService, bikes *BikeService, components *ComponentService, medicalProfiles *MedicalProfileService, medicalRecords *MedicalRecordService, audit *AuditLogService) *ShareLinkService {
	return &ShareLinkService{repo: repo, notes: notes, bikes: bikes, components: components, medicalProfiles: medicalProfiles, medicalRecords: medicalRecords, audit: audit}
}

// List returns the caller's own share links, each enriched with the
// resource's current title/name (best-effort — a deleted resource just
// shows a blank title rather than failing the whole list).
func (s *ShareLinkService) List(ctx context.Context) ([]*models.ShareLink, error) {
	links, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	for _, link := range links {
		link.ResourceTitle = s.resourceTitle(ctx, link.ResourceType, link.ResourceID)
	}
	return links, nil
}

func (s *ShareLinkService) resourceTitle(ctx context.Context, resourceType models.ShareResourceType, resourceID string) string {
	switch resourceType {
	case models.ShareResourceNote:
		if note, err := s.notes.FindByID(ctx, resourceID); err == nil && note != nil {
			return note.Title
		}
	case models.ShareResourceBike:
		if bike, err := s.bikes.FindByID(ctx, resourceID); err == nil && bike != nil {
			return bike.Name
		}
	}
	return ""
}

// Create verifies the caller owns resourceID (via the normal RLS-scoped
// service, under the caller's own JWT) before minting a link — the public
// resolver later trusts resource_id with no further ownership check, so
// this is the one place that matters.
//
// Sensitive-flagged notes need confirmSensitive == true and an expiry of at
// most 7 days ("never" is rejected); creation is audit-logged. Revocation is
// covered by the AuditLog() middleware (DELETE /v1/share-links/:id).
func (s *ShareLinkService) Create(ctx context.Context, resourceType models.ShareResourceType, resourceID string, expiresAt *time.Time, confirmSensitive bool) (*models.ShareLink, error) {
	userID := auth.UserIDFromContext(ctx)
	if userID == "" {
		return nil, fmt.Errorf("no authenticated user")
	}
	if expiresAt != nil && expiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("expires_at must be in the future")
	}
	noteSensitive := false

	switch resourceType {
	case models.ShareResourceNote:
		note, err := s.notes.FindByID(ctx, resourceID)
		if err != nil {
			return nil, fmt.Errorf("find note: %w", err)
		}
		if note == nil || note.DeletedAt != nil {
			return nil, fmt.Errorf("note not found")
		}
		if note.Sensitive {
			noteSensitive = true
			if !confirmSensitive {
				return nil, ErrSensitiveConfirmRequired
			}
			if expiresAt == nil {
				return nil, fmt.Errorf("links to sensitive notes must expire (max 7 days)")
			}
			if expiresAt.After(time.Now().Add(sensitiveShareMaxTTL + time.Minute)) {
				return nil, fmt.Errorf("links to sensitive notes can last at most 7 days")
			}
		}
	case models.ShareResourceBike:
		bike, err := s.bikes.FindByID(ctx, resourceID)
		if err != nil {
			return nil, fmt.Errorf("find bike: %w", err)
		}
		if bike == nil {
			return nil, fmt.Errorf("bike not found")
		}
	case models.ShareResourceMedicalProfile:
		profile, err := s.medicalProfiles.FindByID(ctx, resourceID)
		if err != nil {
			return nil, fmt.Errorf("find medical profile: %w", err)
		}
		if profile == nil {
			return nil, fmt.Errorf("medical profile not found")
		}
	case models.ShareResourceMedicalRecord:
		record, err := s.medicalRecords.FindByID(ctx, resourceID)
		if err != nil {
			return nil, fmt.Errorf("find medical record: %w", err)
		}
		if record == nil {
			return nil, fmt.Errorf("medical record not found")
		}
	default:
		return nil, fmt.Errorf("unsupported resource_type %q", resourceType)
	}

	token, err := generateShortShareToken()
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	link, err := s.repo.Create(ctx, models.ShareLinkInput{
		UserID:       userID,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Token:        token,
		ExpiresAt:    expiresAt,
	})
	if err != nil {
		return nil, err
	}
	if resourceType == models.ShareResourceNote {
		exp := "never"
		if expiresAt != nil {
			exp = expiresAt.UTC().Format(time.RFC3339)
		}
		auditErr := s.audit.Create(ctx, "SHARE_CREATE",
			fmt.Sprintf("/share-links note=%s sensitive=%t expires=%s", resourceID, noteSensitive, exp), 201)
		if auditErr != nil && noteSensitive {
			// Fail closed: a sensitive link must never exist without an audit entry.
			if link != nil {
				_ = s.repo.Update(ctx, link.ID, map[string]any{"revoked": true})
			}
			return nil, fmt.Errorf("audit log: %w", auditErr)
		}
	}
	return link, nil
}

// Revoke soft-deletes a share link (owner-scoped by RLS).
func (s *ShareLinkService) Revoke(ctx context.Context, id string) error {
	return s.repo.Update(ctx, id, map[string]any{"revoked": true})
}

// Resolve looks up a token for the public viewer, rejecting a revoked or
// expired link identically to an unknown one, and best-effort records the
// view.
func (s *ShareLinkService) Resolve(ctx context.Context, token string) (*models.SharedResource, error) {
	link, err := s.repo.FindByToken(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("find share link: %w", err)
	}
	if link == nil || link.Revoked || (link.ExpiresAt != nil && link.ExpiresAt.Before(time.Now())) {
		return nil, ErrShareLinkNotFound
	}

	now := time.Now().UTC()
	_ = s.repo.Update(ctx, link.ID, map[string]any{
		"view_count":     link.ViewCount + 1,
		"last_viewed_at": now.Format(time.RFC3339),
	})

	switch link.ResourceType {
	case models.ShareResourceNote:
		note, err := s.notes.FindByID(ctx, link.ResourceID)
		if err != nil || note == nil {
			return nil, ErrShareLinkNotFound
		}
		if note.DeletedAt != nil {
			return nil, ErrShareLinkNotFound
		}
		view := &models.SharedNoteView{Title: note.Title, ContentFormat: note.ContentFormat}
		if note.ContentFormat == "org" {
			// Never expose raw org source: render sanitized HTML in share mode
			// (no note-to-note links, no image embeds).
			if note.Content != nil {
				view.RenderedHTML = vault.RenderOrg(*note.Content, vault.RenderOptions{Share: true})
			}
		} else {
			view.Content = note.Content
		}
		return &models.SharedResource{ResourceType: models.ShareResourceNote, Note: view}, nil

	case models.ShareResourceBike:
		bike, err := s.bikes.FindByID(ctx, link.ResourceID)
		if err != nil || bike == nil {
			return nil, ErrShareLinkNotFound
		}
		comps, err := s.components.ListByBikeID(ctx, link.ResourceID)
		if err != nil {
			comps = nil // best-effort — a bike with no visible components is still a valid share
		}
		views := make([]models.SharedComponentView, 0, len(comps))
		for _, c := range comps {
			views = append(views, models.SharedComponentView{
				Name: c.Name, Brand: c.Brand, Model: c.Model, IsActive: c.IsActive,
			})
		}
		return &models.SharedResource{
			ResourceType: models.ShareResourceBike,
			Bike: &models.SharedBikeView{
				Name: bike.Name, Type: bike.Type, Model: bike.Model, Mileage: bike.Mileage,
				Components: views,
			},
		}, nil

	case models.ShareResourceMedicalProfile:
		profile, err := s.medicalProfiles.FindByID(ctx, link.ResourceID)
		if err != nil || profile == nil {
			return nil, ErrShareLinkNotFound
		}
		records, err := s.medicalRecords.ListByProfile(ctx, link.ResourceID)
		if err != nil {
			records = nil // best-effort — a profile with no visible records is still a valid share
		}
		summaries := make([]models.SharedMedicalRecordSummary, 0, len(records))
		for _, r := range records {
			summaries = append(summaries, models.SharedMedicalRecordSummary{
				Title: r.Title, RecordType: r.RecordType, RecordDate: r.RecordDate,
			})
		}
		return &models.SharedResource{
			ResourceType:   models.ShareResourceMedicalProfile,
			MedicalProfile: &models.SharedMedicalProfileView{Name: profile.Name, Records: summaries},
		}, nil

	case models.ShareResourceMedicalRecord:
		record, err := s.medicalRecords.FindByID(ctx, link.ResourceID)
		if err != nil || record == nil {
			return nil, ErrShareLinkNotFound
		}
		return &models.SharedResource{
			ResourceType: models.ShareResourceMedicalRecord,
			MedicalRecord: &models.SharedMedicalRecordView{
				Title: record.Title, RecordType: record.RecordType, RecordDate: record.RecordDate,
				Attributes: record.Attributes, Notes: record.Notes,
				DoctorName: record.DoctorName, Recommendations: record.Recommendations,
			},
		}, nil

	default:
		return nil, ErrShareLinkNotFound
	}
}

func generateShortShareToken() (string, error) {
	buf := make([]byte, shareTokenLength)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	out := make([]byte, shareTokenLength)
	for i, b := range buf {
		out[i] = shareTokenAlphabet[int(b)%len(shareTokenAlphabet)]
	}
	return string(out), nil
}
