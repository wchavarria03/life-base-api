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
	rows, err := databases.Patch[[]*models.Note](ctx, r.client, "/rest/v1/notes", databases.EqID(id), fields, "return=representation", notesSchema)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

func (r *NoteRepository) Delete(ctx context.Context, id string) error {
	return databases.Delete(ctx, r.client, "/rest/v1/notes", databases.EqID(id), notesSchema)
}

// NoteVersionRepository persists notes.note_versions rows.
type NoteVersionRepository struct {
	client *databases.SupabaseClient
}

// NewNoteVersionRepository constructs a NoteVersionRepository.
func NewNoteVersionRepository(client *databases.SupabaseClient) *NoteVersionRepository {
	return &NoteVersionRepository{client: client}
}

// ListByNote returns a note's archived versions, newest first.
func (r *NoteVersionRepository) ListByNote(ctx context.Context, noteID string) ([]*models.NoteVersion, error) {
	return databases.Get[[]*models.NoteVersion](ctx, r.client, "/rest/v1/note_versions", url.Values{
		"note_id": []string{"eq." + noteID}, "order": []string{"version_number.desc"},
	}, notesSchema)
}

// Create archives a version.
func (r *NoteVersionRepository) Create(ctx context.Context, v *models.NoteVersion) (*models.NoteVersion, error) {
	rows, err := databases.Post[[]*models.NoteVersion](ctx, r.client, "/rest/v1/note_versions", v, "return=representation", notesSchema)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}
