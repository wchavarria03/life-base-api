package supabase

import (
	"context"
	"net/url"

	"life-base-api/app/internal/databases"
	"life-base-api/app/internal/models"
)

// DocumentRepository persists document vault rows (metadata only — the
// file itself lives in the private "documents" storage bucket).
type DocumentRepository struct {
	client *databases.SupabaseClient
}

// NewDocumentRepository constructs a DocumentRepository.
func NewDocumentRepository(client *databases.SupabaseClient) *DocumentRepository {
	return &DocumentRepository{client: client}
}

// List returns every document, newest first.
func (r *DocumentRepository) List(ctx context.Context) ([]*models.Document, error) {
	return databases.Get[[]*models.Document](ctx, r.client, "/rest/v1/documents",
		url.Values{"order": []string{"created_at.desc"}})
}

// FindByID looks up one document (RLS-scoped).
func (r *DocumentRepository) FindByID(ctx context.Context, id string) (*models.Document, error) {
	rows, err := databases.Get[[]*models.Document](ctx, r.client, "/rest/v1/documents", url.Values{
		"id":    []string{"eq." + id},
		"limit": []string{"1"},
	})
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

// Create inserts a new document row.
func (r *DocumentRepository) Create(ctx context.Context, input models.DocumentInput) (*models.Document, error) {
	rows, err := databases.Post[[]*models.Document](ctx, r.client, "/rest/v1/documents", input, "return=representation")
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

// Delete removes a document row (RLS-scoped).
func (r *DocumentRepository) Delete(ctx context.Context, id string) error {
	return databases.Delete(ctx, r.client, "/rest/v1/documents", databases.EqID(id))
}
