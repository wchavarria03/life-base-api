package supabase

import (
	"context"
	"net/url"

	"life-base-api/app/internal/databases"
	"life-base-api/app/internal/models"
)

// CaptionRepository persists caption templates, versions, and categories.
type CaptionRepository struct {
	client *databases.SupabaseClient
}

// NewCaptionRepository constructs a CaptionRepository.
func NewCaptionRepository(client *databases.SupabaseClient) *CaptionRepository {
	return &CaptionRepository{client: client}
}

// ── Categories ───────────────────────────────────────────────────────────────

// ListCategories returns every caption category for the caller.
func (r *CaptionRepository) ListCategories(ctx context.Context) ([]*models.CaptionCategory, error) {
	return databases.Get[[]*models.CaptionCategory](ctx, r.client, "/rest/v1/caption_categories",
		url.Values{"order": []string{"name.asc"}})
}

// CreateCategory inserts a new caption category.
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

// UpdateCategory renames a caption category.
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

// DeleteCategory removes a caption category.
func (r *CaptionRepository) DeleteCategory(ctx context.Context, id string) error {
	return databases.Delete(ctx, r.client, "/rest/v1/caption_categories", databases.EqID(id))
}

// ── Templates ────────────────────────────────────────────────────────────────

// TemplateRow is a caption_templates metadata row, without its body (which
// lives in the current version) or category IDs (a separate junction table).
type TemplateRow struct {
	ID        string `json:"id"`
	UserID    string `json:"user_id,omitempty"`
	Title     string `json:"title"`
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

// ListTemplates returns every caption template's metadata row.
func (r *CaptionRepository) ListTemplates(ctx context.Context) ([]*TemplateRow, error) {
	return databases.Get[[]*TemplateRow](ctx, r.client, "/rest/v1/caption_templates",
		url.Values{"order": []string{"updated_at.desc"}})
}

// CreateTemplate inserts a new template row (without a version yet).
func (r *CaptionRepository) CreateTemplate(ctx context.Context, title string) (*TemplateRow, error) {
	rows, err := databases.Post[[]*TemplateRow](ctx, r.client, "/rest/v1/caption_templates",
		map[string]string{"title": title}, "return=representation")
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

// UpdateTemplateTitle renames a template.
func (r *CaptionRepository) UpdateTemplateTitle(ctx context.Context, id, title string) error {
	_, err := databases.Patch[[]*TemplateRow](ctx, r.client, "/rest/v1/caption_templates",
		databases.EqID(id), map[string]string{"title": title}, "")
	return err
}

// SetCurrentVersion points a template at the given version.
func (r *CaptionRepository) SetCurrentVersion(ctx context.Context, templateID, versionID string) error {
	_, err := databases.Patch[[]*TemplateRow](ctx, r.client, "/rest/v1/caption_templates",
		databases.EqID(templateID), map[string]string{"current_version_id": versionID}, "")
	return err
}

// DeleteTemplate removes a template and its versions/category links.
func (r *CaptionRepository) DeleteTemplate(ctx context.Context, id string) error {
	return databases.Delete(ctx, r.client, "/rest/v1/caption_templates", databases.EqID(id))
}

// ── Versions ─────────────────────────────────────────────────────────────────

// ListVersions returns a template's versions, newest first.
func (r *CaptionRepository) ListVersions(ctx context.Context, templateID string) ([]*models.CaptionTemplateVersion, error) {
	return databases.Get[[]*models.CaptionTemplateVersion](ctx, r.client, "/rest/v1/caption_template_versions",
		url.Values{
			"template_id": []string{"eq." + templateID},
			"order":       []string{"version_number.desc"},
		})
}

// GetVersion returns a single version by id.
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

// ListTemplateCategoryIDs returns every template's assigned category IDs.
func (r *CaptionRepository) ListTemplateCategoryIDs(ctx context.Context) (map[string][]string, error) {
	return listJunctionIDs(ctx, r.client, "/rest/v1/caption_template_categories", "template_id", "category_id")
}

// SetTemplateCategories replaces a template's category assignments.
func (r *CaptionRepository) SetTemplateCategories(ctx context.Context, templateID string, categoryIDs []string) error {
	return replaceJunctionRows(ctx, r.client, "/rest/v1/caption_template_categories", "template_id", templateID, "category_id", categoryIDs)
}

// ── Post <-> category junction ──────────────────────────────────────────────

// ListPostCategoryIDs returns every social post's assigned category IDs.
func (r *CaptionRepository) ListPostCategoryIDs(ctx context.Context) (map[string][]string, error) {
	return listJunctionIDs(ctx, r.client, "/rest/v1/social_post_categories", "social_post_id", "category_id")
}

// SetPostCategories replaces a social post's category assignments.
func (r *CaptionRepository) SetPostCategories(ctx context.Context, postID string, categoryIDs []string) error {
	return replaceJunctionRows(ctx, r.client, "/rest/v1/social_post_categories", "social_post_id", postID, "category_id", categoryIDs)
}

// listJunctionIDs groups a two-column many-to-many table by its left-hand
// column, returning leftID -> []rightID.
func listJunctionIDs(ctx context.Context, client *databases.SupabaseClient, path, leftCol, rightCol string) (map[string][]string, error) {
	rows, err := databases.Get[[]map[string]string](ctx, client, path, nil)
	if err != nil {
		return nil, err
	}
	out := make(map[string][]string)
	for _, row := range rows {
		out[row[leftCol]] = append(out[row[leftCol]], row[rightCol])
	}
	return out, nil
}

// replaceJunctionRows deletes every row for leftID in a two-column
// many-to-many table and reinserts one row per rightID.
func replaceJunctionRows(ctx context.Context, client *databases.SupabaseClient, path, leftCol, leftID, rightCol string, rightIDs []string) error {
	if err := databases.Delete(ctx, client, path, url.Values{leftCol: []string{"eq." + leftID}}); err != nil {
		return err
	}
	if len(rightIDs) == 0 {
		return nil
	}
	rows := make([]map[string]string, len(rightIDs))
	for i, id := range rightIDs {
		rows[i] = map[string]string{leftCol: leftID, rightCol: id}
	}
	_, err := databases.Post[struct{}](ctx, client, path, rows, "")
	return err
}
