package services

import (
	"context"
	"fmt"

	"life-base-api/app/internal/auth"
	"life-base-api/app/internal/models"
	supabaserepo "life-base-api/app/internal/repositories/supabase"
)

type NoteRepository interface {
	List(ctx context.Context) ([]*models.Note, error)
	FindByID(ctx context.Context, id string) (*models.Note, error)
	Create(ctx context.Context, input models.NoteInput) (*models.Note, error)
	Update(ctx context.Context, id string, fields map[string]any) (*models.Note, error)
	Delete(ctx context.Context, id string) error
}

type NoteService struct {
	notes    NoteRepository
	versions *supabaserepo.NoteVersionRepository
}

// NewNoteService constructs a NoteService.
func NewNoteService(notes NoteRepository, versions *supabaserepo.NoteVersionRepository) *NoteService {
	return &NoteService{notes: notes, versions: versions}
}

func (s *NoteService) List(ctx context.Context) ([]*models.Note, error) {
	return s.notes.List(ctx)
}

func (s *NoteService) FindByID(ctx context.Context, id string) (*models.Note, error) {
	return s.notes.FindByID(ctx, id)
}

func (s *NoteService) Create(ctx context.Context, input models.NoteInput) (*models.Note, error) {
	if auth.UserIDFromContext(ctx) == "" {
		return nil, fmt.Errorf("no authenticated user")
	}
	if input.Title == "" {
		return nil, fmt.Errorf("title is required")
	}
	return s.notes.Create(ctx, input)
}

// Update patches a note, first archiving its current title/content as a
// new version — so every edit (e.g. a nutritionist's updated plan) leaves
// the prior one visible in history.
func (s *NoteService) Update(ctx context.Context, id string, fields map[string]any) (*models.Note, error) {
	if len(fields) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}
	current, err := s.notes.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("find note: %w", err)
	}
	if current != nil {
		existing, err := s.versions.ListByNote(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("list note versions: %w", err)
		}
		next := 1
		if len(existing) > 0 {
			next = existing[0].VersionNumber + 1
		}
		if _, err := s.versions.Create(ctx, &models.NoteVersion{
			NoteID: id, Title: current.Title, Content: current.Content, VersionNumber: next,
		}); err != nil {
			return nil, fmt.Errorf("archive note version: %w", err)
		}
	}
	return s.notes.Update(ctx, id, fields)
}

// ListVersions returns a note's archived prior versions, newest first.
func (s *NoteService) ListVersions(ctx context.Context, noteID string) ([]*models.NoteVersion, error) {
	return s.versions.ListByNote(ctx, noteID)
}

func (s *NoteService) Delete(ctx context.Context, id string) error {
	return s.notes.Delete(ctx, id)
}
