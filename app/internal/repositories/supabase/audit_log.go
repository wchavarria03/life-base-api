package supabase

import (
	"context"
	"net/url"
	"strconv"

	"life-base-api/app/internal/databases"
	"life-base-api/app/internal/models"
)

type AuditLogRepository struct {
	client *databases.SupabaseClient
}

func NewAuditLogRepository(client *databases.SupabaseClient) *AuditLogRepository {
	return &AuditLogRepository{client: client}
}

func (r *AuditLogRepository) Create(ctx context.Context, entry *models.AuditLogEntry) error {
	_, err := databases.Post[[]*models.AuditLogEntry](ctx, r.client, "/rest/v1/audit_logs", entry, "")
	return err
}

func (r *AuditLogRepository) List(ctx context.Context, limit, offset int) ([]*models.AuditLogEntry, error) {
	return databases.Get[[]*models.AuditLogEntry](ctx, r.client, "/rest/v1/audit_logs", url.Values{
		"order":  []string{"created_at.desc"},
		"limit":  []string{strconv.Itoa(limit)},
		"offset": []string{strconv.Itoa(offset)},
	})
}
