package supabase

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"life-base-api/app/internal/databases"
	"life-base-api/app/internal/models"
)

type SocialPostRepository struct {
	client *databases.SupabaseClient
}

func NewSocialPostRepository(client *databases.SupabaseClient) *SocialPostRepository {
	return &SocialPostRepository{client: client}
}

// List returns posts newest-first. limit <= 0 and offset <= 0 are treated as
// "no limit" / "no offset" respectively; status, if non-nil, matches a post
// where either network has that status.
func (r *SocialPostRepository) List(ctx context.Context, limit, offset int, status *models.SocialPostStatus) ([]*models.SocialPost, error) {
	params := url.Values{"order": []string{"created_at.desc"}}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	if offset > 0 {
		params.Set("offset", strconv.Itoa(offset))
	}
	if status != nil {
		params.Set("or", fmt.Sprintf("(facebook_status.eq.%s,instagram_status.eq.%s)", *status, *status))
	}
	return databases.Get[[]*models.SocialPost](ctx, r.client, "/rest/v1/social_posts", params)
}

func (r *SocialPostRepository) FindByID(ctx context.Context, id string) (*models.SocialPost, error) {
	rows, err := databases.Get[[]*models.SocialPost](ctx, r.client, "/rest/v1/social_posts", url.Values{
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

// FindRecentByFilename returns posts with this exact filename created at or
// after since, newest first — used to warn about re-posting the same image.
func (r *SocialPostRepository) FindRecentByFilename(ctx context.Context, filename string, since time.Time) ([]*models.SocialPost, error) {
	return databases.Get[[]*models.SocialPost](ctx, r.client, "/rest/v1/social_posts", url.Values{
		"filename":   []string{"eq." + filename},
		"created_at": []string{"gte." + since.UTC().Format(time.RFC3339)},
		"order":      []string{"created_at.desc"},
	})
}

func (r *SocialPostRepository) Create(ctx context.Context, input models.SocialPostInput) (*models.SocialPost, error) {
	rows, err := databases.Post[[]*models.SocialPost](ctx, r.client,
		"/rest/v1/social_posts",
		input,
		"return=representation")
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

func (r *SocialPostRepository) Update(ctx context.Context, id string, fields map[string]any) (*models.SocialPost, error) {
	rows, err := databases.Patch[[]*models.SocialPost](ctx, r.client,
		"/rest/v1/social_posts?id=eq."+id,
		fields,
		"return=representation")
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

func (r *SocialPostRepository) Delete(ctx context.Context, id string) error {
	return databases.Delete(ctx, r.client, "/rest/v1/social_posts?id=eq."+id)
}
