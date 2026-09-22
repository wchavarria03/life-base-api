package services

import (
	"context"
	"fmt"

	"life-base-api/app/internal/auth"
	"life-base-api/app/internal/models"
)

type NoteRepository interface {
	List(ctx context.Context) ([]*models.Note, error)
	FindByID(ctx context.Context, id string) (*models.Note, error)
	Create(ctx context.Context, input models.NoteInput) (*models.Note, error)
	Update(ctx context.Context, id string, fields map[string]any) (*models.Note, error)
	Delete(ctx context.Context, id string) error
}

type NoteService struct {
	notes NoteRepository
}

func NewNoteService(notes NoteRepository) *NoteService {
	return &NoteService{notes: notes}
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

func (s *NoteService) Update(ctx context.Context, id string, fields map[string]any) (*models.Note, error) {
	if len(fields) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}
	return s.notes.Update(ctx, id, fields)
}

func (s *NoteService) Delete(ctx context.Context, id string) error {
	return s.notes.Delete(ctx, id)
}
