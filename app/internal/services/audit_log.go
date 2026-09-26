package services

import (
	"context"

	supabaserepo "life-base-api/app/internal/repositories/supabase"

	"life-base-api/app/internal/auth"
	"life-base-api/app/internal/models"
)

// AuditLogService records and lists audit log entries.
type AuditLogService struct {
	repo *supabaserepo.AuditLogRepository
}

// NewAuditLogService constructs an AuditLogService.
func NewAuditLogService(repo *supabaserepo.AuditLogRepository) *AuditLogService {
	return &AuditLogService{repo: repo}
}

// Create records one audit entry for the request's authenticated user.
// Called from the AuditLog() middleware.
func (s *AuditLogService) Create(ctx context.Context, method, path string, status int) error {
	userID := auth.UserIDFromContext(ctx)
	if userID == "" {
		return nil
	}
	return s.repo.Create(ctx, &models.AuditLogEntry{
		UserID: userID,
		Method: method,
		Path:   path,
		Status: status,
	})
}

// List returns audit log entries, newest first.
func (s *AuditLogService) List(ctx context.Context, limit, offset int) ([]*models.AuditLogEntry, error) {
	return s.repo.List(ctx, limit, offset)
}
