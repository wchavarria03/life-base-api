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
)

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
	repo       *supabaserepo.ShareLinkRepository
	notes      *NoteService
	bikes      *BikeService
	components *ComponentService
}

// NewShareLinkService constructs a ShareLinkService.
func NewShareLinkService(repo *supabaserepo.ShareLinkRepository, notes *NoteService, bikes *BikeService, components *ComponentService) *ShareLinkService {
	return &ShareLinkService{repo: repo, notes: notes, bikes: bikes, components: components}
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
func (s *ShareLinkService) Create(ctx context.Context, resourceType models.ShareResourceType, resourceID string, expiresAt *time.Time) (*models.ShareLink, error) {
	userID := auth.UserIDFromContext(ctx)
	if userID == "" {
		return nil, fmt.Errorf("no authenticated user")
	}

	switch resourceType {
	case models.ShareResourceNote:
		note, err := s.notes.FindByID(ctx, resourceID)
		if err != nil {
			return nil, fmt.Errorf("find note: %w", err)
		}
		if note == nil {
			return nil, fmt.Errorf("note not found")
		}
	case models.ShareResourceBike:
		bike, err := s.bikes.FindByID(ctx, resourceID)
		if err != nil {
			return nil, fmt.Errorf("find bike: %w", err)
		}
		if bike == nil {
			return nil, fmt.Errorf("bike not found")
		}
	default:
		return nil, fmt.Errorf("unsupported resource_type %q", resourceType)
	}

	token, err := generateShortShareToken()
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	return s.repo.Create(ctx, models.ShareLinkInput{
		UserID:       userID,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Token:        token,
		ExpiresAt:    expiresAt,
	})
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
		return &models.SharedResource{
			ResourceType: models.ShareResourceNote,
			Note:         &models.SharedNoteView{Title: note.Title, Content: note.Content},
		}, nil

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
