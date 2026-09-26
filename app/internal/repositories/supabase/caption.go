package supabase

import (
	"context"
	"net/url"

	"life-base-api/app/internal/databases"
	"life-base-api/app/internal/models"
)

type CaptionRepository struct {
	client *databases.SupabaseClient
}

func NewCaptionRepository(client *databases.SupabaseClient) *CaptionRepository {
	return &CaptionRepository{client: client}
}

// ── Categories ───────────────────────────────────────────────────────────────

func (r *CaptionRepository) ListCategories(ctx context.Context) ([]*models.CaptionCategory, error) {
	return databases.Get[[]*models.CaptionCategory](ctx, r.client, "/rest/v1/caption_categories",
		url.Values{"order": []string{"name.asc"}})
}

func (r *CaptionRepository) CreateCategory(ctx context.Context, c *models.CaptionCategory) (*models.CaptionCategory, error) {
	rows, err := databases.Post[[]*models.CaptionCategory](ctx, r.client, "/rest/v1/caption_categories", c, "return=representation")
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

func (r *CaptionRepository) UpdateCategory(ctx context.Context, id, name string) (*models.CaptionCategory, error) {
	rows, err := databases.Patch[[]*models.CaptionCategory](ctx, r.client, "/rest/v1/caption_categories",
		databases.EqID(id), map[string]string{"name": name}, "return=representation")
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

func (r *CaptionRepository) DeleteCategory(ctx context.Context, id string) error {
	return databases.Delete(ctx, r.client, "/rest/v1/caption_categories", databases.EqID(id))
}

// ── Templates ────────────────────────────────────────────────────────────────

type templateRow struct {
	ID        string `json:"id"`
	UserID    string `json:"user_id,omitempty"`
	Title     string `json:"title"`
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

func (r *CaptionRepository) ListTemplates(ctx context.Context) ([]*templateRow, error) {
	return databases.Get[[]*templateRow](ctx, r.client, "/rest/v1/caption_templates",
		url.Values{"order": []string{"updated_at.desc"}})
}

func (r *CaptionRepository) CreateTemplate(ctx context.Context, title string) (*templateRow, error) {
	rows, err := databases.Post[[]*templateRow](ctx, r.client, "/rest/v1/caption_templates",
		map[string]string{"title": title}, "return=representation")
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

func (r *CaptionRepository) UpdateTemplateTitle(ctx context.Context, id, title string) error {
	_, err := databases.Patch[[]*templateRow](ctx, r.client, "/rest/v1/caption_templates",
		databases.EqID(id), map[string]string{"title": title}, "")
	return err
}

func (r *CaptionRepository) SetCurrentVersion(ctx context.Context, templateID, versionID string) error {
	_, err := databases.Patch[[]*templateRow](ctx, r.client, "/rest/v1/caption_templates",
		databases.EqID(templateID), map[string]string{"current_version_id": versionID}, "")
	return err
}

func (r *CaptionRepository) DeleteTemplate(ctx context.Context, id string) error {
	return databases.Delete(ctx, r.client, "/rest/v1/caption_templates", databases.EqID(id))
}

// ── Versions ─────────────────────────────────────────────────────────────────

func (r *CaptionRepository) ListVersions(ctx context.Context, templateID string) ([]*models.CaptionTemplateVersion, error) {
	return databases.Get[[]*models.CaptionTemplateVersion](ctx, r.client, "/rest/v1/caption_template_versions",
		url.Values{
			"template_id": []string{"eq." + templateID},
			"order":       []string{"version_number.desc"},
		})
}

func (r *CaptionRepository) GetVersion(ctx context.Context, id string) (*models.CaptionTemplateVersion, error) {
	rows, err := databases.Get[[]*models.CaptionTemplateVersion](ctx, r.client, "/rest/v1/caption_template_versions",
		url.Values{"id": []string{"eq." + id}, "limit": []string{"1"}})
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

// AddVersion inserts a new version at nextVersionNumber and returns it.
func (r *CaptionRepository) AddVersion(ctx context.Context, templateID, body string, nextVersionNumber int) (*models.CaptionTemplateVersion, error) {
	rows, err := databases.Post[[]*models.CaptionTemplateVersion](ctx, r.client, "/rest/v1/caption_template_versions",
		map[string]any{"template_id": templateID, "body": body, "version_number": nextVersionNumber},
		"return=representation")
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

// ── Template <-> category junction ──────────────────────────────────────────

func (r *CaptionRepository) ListTemplateCategoryIDs(ctx context.Context) (map[string][]string, error) {
	type row struct {
		TemplateID string `json:"template_id"`
		CategoryID string `json:"category_id"`
	}
	rows, err := databases.Get[[]row](ctx, r.client, "/rest/v1/caption_template_categories", nil)
	if err != nil {
		return nil, err
	}
	out := make(map[string][]string)
	for _, row := range rows {
		out[row.TemplateID] = append(out[row.TemplateID], row.CategoryID)
	}
	return out, nil
}

func (r *CaptionRepository) SetTemplateCategories(ctx context.Context, templateID string, categoryIDs []string) error {
	if err := databases.Delete(ctx, r.client, "/rest/v1/caption_template_categories",
		url.Values{"template_id": []string{"eq." + templateID}}); err != nil {
		return err
	}
	if len(categoryIDs) == 0 {
		return nil
	}
	type row struct {
		TemplateID string `json:"template_id"`
		CategoryID string `json:"category_id"`
	}
	rows := make([]row, len(categoryIDs))
	for i, id := range categoryIDs {
		rows[i] = row{TemplateID: templateID, CategoryID: id}
	}
	_, err := databases.Post[struct{}](ctx, r.client, "/rest/v1/caption_template_categories", rows, "")
	return err
}

// ── Post <-> category junction ──────────────────────────────────────────────

func (r *CaptionRepository) ListPostCategoryIDs(ctx context.Context) (map[string][]string, error) {
	type row struct {
		SocialPostID string `json:"social_post_id"`
		CategoryID   string `json:"category_id"`
	}
	rows, err := databases.Get[[]row](ctx, r.client, "/rest/v1/social_post_categories", nil)
	if err != nil {
		return nil, err
	}
	out := make(map[string][]string)
	for _, row := range rows {
		out[row.SocialPostID] = append(out[row.SocialPostID], row.CategoryID)
	}
	return out, nil
}

func (r *CaptionRepository) SetPostCategories(ctx context.Context, postID string, categoryIDs []string) error {
	if err := databases.Delete(ctx, r.client, "/rest/v1/social_post_categories",
		url.Values{"social_post_id": []string{"eq." + postID}}); err != nil {
		return err
	}
	if len(categoryIDs) == 0 {
		return nil
	}
	type row struct {
		SocialPostID string `json:"social_post_id"`
		CategoryID   string `json:"category_id"`
	}
	rows := make([]row, len(categoryIDs))
	for i, id := range categoryIDs {
		rows[i] = row{SocialPostID: postID, CategoryID: id}
	}
	_, err := databases.Post[struct{}](ctx, r.client, "/rest/v1/social_post_categories", rows, "")
	return err
}
