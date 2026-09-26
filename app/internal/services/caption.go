package services

import (
	"context"
	"fmt"

	supabaserepo "life-base-api/app/internal/repositories/supabase"

	"life-base-api/app/internal/auth"
	"life-base-api/app/internal/models"
)

// CaptionService manages the caption template library.
type CaptionService struct {
	repo *supabaserepo.CaptionRepository
}

// NewCaptionService constructs a CaptionService.
func NewCaptionService(repo *supabaserepo.CaptionRepository) *CaptionService {
	return &CaptionService{repo: repo}
}

// ── Categories ───────────────────────────────────────────────────────────────

// ListCategories returns every caption category.
func (s *CaptionService) ListCategories(ctx context.Context) ([]*models.CaptionCategory, error) {
	return s.repo.ListCategories(ctx)
}

// CreateCategory creates a new caption category.
func (s *CaptionService) CreateCategory(ctx context.Context, name string) (*models.CaptionCategory, error) {
	return s.repo.CreateCategory(ctx, &models.CaptionCategory{UserID: auth.UserIDFromContext(ctx), Name: name})
}

// UpdateCategory renames a caption category.
func (s *CaptionService) UpdateCategory(ctx context.Context, id, name string) (*models.CaptionCategory, error) {
	return s.repo.UpdateCategory(ctx, id, name)
}

// DeleteCategory removes a caption category.
func (s *CaptionService) DeleteCategory(ctx context.Context, id string) error {
	return s.repo.DeleteCategory(ctx, id)
}

// ── Templates ────────────────────────────────────────────────────────────────

// ListTemplates returns every template with its current body and category
// IDs merged in — three PostgREST calls combined in Go, same approach as
// AdminService.ListMembers merging in auth emails.
func (s *CaptionService) ListTemplates(ctx context.Context) ([]*models.CaptionTemplate, error) {
	rows, err := s.repo.ListTemplates(ctx)
	if err != nil {
		return nil, err
	}

	catsByTemplate, err := s.repo.ListTemplateCategoryIDs(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]*models.CaptionTemplate, 0, len(rows))
	for _, row := range rows {
		t := &models.CaptionTemplate{
			ID:          row.ID,
			Title:       row.Title,
			CategoryIDs: catsByTemplate[row.ID],
		}
		if t.CategoryIDs == nil {
			t.CategoryIDs = []string{}
		}

		versions, err := s.repo.ListVersions(ctx, row.ID)
		if err != nil {
			return nil, err
		}
		if len(versions) > 0 {
			t.CurrentBody = versions[0].Body
		}
		out = append(out, t)
	}
	return out, nil
}

// CreateTemplate creates a template, its first version, and its category
// links in one call.
func (s *CaptionService) CreateTemplate(ctx context.Context, title, body string, categoryIDs []string) (*models.CaptionTemplate, error) {
	row, err := s.repo.CreateTemplate(ctx, auth.UserIDFromContext(ctx), title)
	if err != nil {
		return nil, err
	}

	version, err := s.repo.AddVersion(ctx, row.ID, body, 1)
	if err != nil {
		return nil, fmt.Errorf("add initial version: %w", err)
	}
	if err := s.repo.SetCurrentVersion(ctx, row.ID, version.ID); err != nil {
		return nil, fmt.Errorf("set current version: %w", err)
	}
	if err := s.repo.SetTemplateCategories(ctx, row.ID, categoryIDs); err != nil {
		return nil, fmt.Errorf("set categories: %w", err)
	}

	return &models.CaptionTemplate{ID: row.ID, Title: row.Title, CurrentBody: body, CategoryIDs: categoryIDs}, nil
}

// UpdateTemplateMeta updates title and/or category assignments only — no
// new version.
func (s *CaptionService) UpdateTemplateMeta(ctx context.Context, id string, title *string, categoryIDs *[]string) error {
	if title != nil {
		if err := s.repo.UpdateTemplateTitle(ctx, id, *title); err != nil {
			return err
		}
	}
	if categoryIDs != nil {
		if err := s.repo.SetTemplateCategories(ctx, id, *categoryIDs); err != nil {
			return err
		}
	}
	return nil
}

// DeleteTemplate removes a caption template.
func (s *CaptionService) DeleteTemplate(ctx context.Context, id string) error {
	return s.repo.DeleteTemplate(ctx, id)
}

// ── Versions ─────────────────────────────────────────────────────────────────

// ListVersions returns a template's version history.
func (s *CaptionService) ListVersions(ctx context.Context, templateID string) ([]*models.CaptionTemplateVersion, error) {
	return s.repo.ListVersions(ctx, templateID)
}

// AddVersion is both "customize" and "create new version" — every save
// inserts a new version and makes it current; nothing is overwritten.
func (s *CaptionService) AddVersion(ctx context.Context, templateID, body string) (*models.CaptionTemplateVersion, error) {
	existing, err := s.repo.ListVersions(ctx, templateID)
	if err != nil {
		return nil, err
	}
	next := 1
	if len(existing) > 0 {
		next = existing[0].VersionNumber + 1
	}

	version, err := s.repo.AddVersion(ctx, templateID, body, next)
	if err != nil {
		return nil, err
	}
	if err := s.repo.SetCurrentVersion(ctx, templateID, version.ID); err != nil {
		return nil, err
	}
	return version, nil
}

// RevertToVersion makes an older version current again.
func (s *CaptionService) RevertToVersion(ctx context.Context, templateID, versionID string) error {
	version, err := s.repo.GetVersion(ctx, versionID)
	if err != nil {
		return err
	}
	if version == nil || version.TemplateID != templateID {
		return fmt.Errorf("version not found for template")
	}
	return s.repo.SetCurrentVersion(ctx, templateID, versionID)
}

// ── Post categories ──────────────────────────────────────────────────────────

// SetPostCategories tags a social post with the given categories.
func (s *CaptionService) SetPostCategories(ctx context.Context, postID string, categoryIDs []string) error {
	return s.repo.SetPostCategories(ctx, postID, categoryIDs)
}

// ListPostCategoryIDs returns every social post's assigned category IDs.
func (s *CaptionService) ListPostCategoryIDs(ctx context.Context) (map[string][]string, error) {
	return s.repo.ListPostCategoryIDs(ctx)
}
