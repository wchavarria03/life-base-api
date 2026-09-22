package supabase

import (
	"context"
	"net/url"

	"life-base-api/app/internal/databases"
	"life-base-api/app/internal/models"
)

const notesSchema = "notes"

type NoteRepository struct {
	client *databases.SupabaseClient
}

func NewNoteRepository(client *databases.SupabaseClient) *NoteRepository {
	return &NoteRepository{client: client}
}

func (r *NoteRepository) List(ctx context.Context) ([]*models.Note, error) {
	return databases.Get[[]*models.Note](ctx, r.client, "/rest/v1/notes", url.Values{
		"order": []string{"updated_at.desc"},
	}, notesSchema)
}

func (r *NoteRepository) FindByID(ctx context.Context, id string) (*models.Note, error) {
	rows, err := databases.Get[[]*models.Note](ctx, r.client, "/rest/v1/notes", url.Values{
		"id":    []string{"eq." + id},
		"limit": []string{"1"},
	}, notesSchema)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

func (r *NoteRepository) Create(ctx context.Context, input models.NoteInput) (*models.Note, error) {
	rows, err := databases.Post[[]*models.Note](ctx, r.client, "/rest/v1/notes", input, "return=representation", notesSchema)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

func (r *NoteRepository) Update(ctx context.Context, id string, fields map[string]any) (*models.Note, error) {
	rows, err := databases.Patch[[]*models.Note](ctx, r.client, "/rest/v1/notes?id=eq."+id, fields, "return=representation", notesSchema)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

func (r *NoteRepository) Delete(ctx context.Context, id string) error {
	return databases.Delete(ctx, r.client, "/rest/v1/notes?id=eq."+id, notesSchema)
}
