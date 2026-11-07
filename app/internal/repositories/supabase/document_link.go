package supabase

import (
	"context"
	"net/url"

	"life-base-api/app/internal/databases"
	"life-base-api/app/internal/models"
)

// DocumentLinkRepository persists document_links rows.
type DocumentLinkRepository struct {
	client *databases.SupabaseClient
}

// NewDocumentLinkRepository constructs a DocumentLinkRepository.
func NewDocumentLinkRepository(client *databases.SupabaseClient) *DocumentLinkRepository {
	return &DocumentLinkRepository{client: client}
}

// ListByDocument returns every link of one document.
func (r *DocumentLinkRepository) ListByDocument(ctx context.Context, documentID string) ([]*models.DocumentLink, error) {
	return databases.Get[[]*models.DocumentLink](ctx, r.client, "/rest/v1/document_links", url.Values{
		"document_id": []string{"eq." + documentID},
		"order":       []string{"created_at.desc"},
	})
}

// ListByEntity returns every link attached to one entity.
func (r *DocumentLinkRepository) ListByEntity(ctx context.Context, entityType, entityID string) ([]*models.DocumentLink, error) {
	return databases.Get[[]*models.DocumentLink](ctx, r.client, "/rest/v1/document_links", url.Values{
		"entity_type": []string{"eq." + entityType},
		"entity_id":   []string{"eq." + entityID},
		"order":       []string{"created_at.desc"},
	})
}

// Create inserts a link; an existing identical link is returned unchanged.
func (r *DocumentLinkRepository) Create(ctx context.Context, input models.DocumentLinkInput) (*models.DocumentLink, error) {
	return databases.First(databases.Post[[]*models.DocumentLink](ctx, r.client,
		"/rest/v1/document_links?on_conflict=document_id,entity_type,entity_id", input,
		"resolution=merge-duplicates,return=representation"))
}

// Delete removes one link of a document (RLS-scoped).
func (r *DocumentLinkRepository) Delete(ctx context.Context, documentID, linkID string) error {
	return databases.Delete(ctx, r.client, "/rest/v1/document_links", url.Values{
		"id":          []string{"eq." + linkID},
		"document_id": []string{"eq." + documentID},
	})
}
