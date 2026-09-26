package services

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	supabaserepo "life-base-api/app/internal/repositories/supabase"

	"life-base-api/app/internal/auth"
	"life-base-api/app/internal/models"
)

const scheduledImagesBucket = "social-images"

// ScheduledPostService manages scheduled social posts: uploading and
// storing an image now, then actually posting it at (or after) its
// scheduled time via ProcessDue.
type ScheduledPostService struct {
	repo     *supabaserepo.ScheduledPostRepository
	storage  *supabaserepo.StorageRepository
	social   *SocialService
	captions *CaptionService
	http     *http.Client
}

// NewScheduledPostService constructs a ScheduledPostService.
func NewScheduledPostService(repo *supabaserepo.ScheduledPostRepository, storage *supabaserepo.StorageRepository, social *SocialService, captions *CaptionService) *ScheduledPostService {
	return &ScheduledPostService{repo: repo, storage: storage, social: social, captions: captions, http: &http.Client{Timeout: 30 * time.Second}}
}

// Create uploads the image to storage and inserts a pending scheduled post.
func (s *ScheduledPostService) Create(
	ctx context.Context,
	file io.Reader,
	filename string,
	fbCaption string,
	igCaption *string,
	toFacebook, toInstagram bool,
	categoryIDs []string,
	scheduledAt time.Time,
) (*models.ScheduledPost, error) {
	userID := auth.UserIDFromContext(ctx)
	if userID == "" {
		return nil, fmt.Errorf("no authenticated user")
	}
	if !toFacebook && !toInstagram {
		return nil, fmt.Errorf("select at least one network to post to")
	}

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("read uploaded image: %w", err)
	}

	path := fmt.Sprintf("%s/%d-%s", userID, time.Now().UnixNano(), filename)
	publicURL, err := s.storage.UploadObject(ctx, scheduledImagesBucket, path, data, contentTypeForFilename(filename))
	if err != nil {
		return nil, fmt.Errorf("upload image: %w", err)
	}

	if categoryIDs == nil {
		categoryIDs = []string{}
	}

	return s.repo.Create(ctx, models.ScheduledPostInput{
		UserID:           userID,
		StoragePath:      publicURL,
		Filename:         filename,
		CaptionFacebook:  fbCaption,
		CaptionInstagram: igCaption,
		PostFacebook:     toFacebook,
		PostInstagram:    toInstagram,
		CategoryIDs:      categoryIDs,
		ScheduledAt:      scheduledAt,
	})
}

// List returns the caller's scheduled posts.
func (s *ScheduledPostService) List(ctx context.Context) ([]*models.ScheduledPost, error) {
	return s.repo.List(ctx)
}

// Cancel marks a still-pending scheduled post as canceled.
func (s *ScheduledPostService) Cancel(ctx context.Context, id string) error {
	post, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("find scheduled post: %w", err)
	}
	if post == nil {
		return fmt.Errorf("scheduled post not found")
	}
	if post.Status != models.ScheduledPostPending {
		return fmt.Errorf("only pending scheduled posts can be canceled")
	}
	return s.repo.Update(ctx, id, map[string]any{"status": string(models.ScheduledPostCanceled)})
}

// ProcessDue sends every due, still-pending scheduled post. userID scopes
// to one user (the manual "check now" button); empty means every user
// (the cron sender, running with the service-role key). Best-effort per
// post — one failure doesn't stop the batch.
func (s *ScheduledPostService) ProcessDue(ctx context.Context, userID string) (sent int, failed int, err error) {
	due, err := s.repo.ListDue(ctx, userID)
	if err != nil {
		return 0, 0, fmt.Errorf("list due scheduled posts: %w", err)
	}

	for _, p := range due {
		if procErr := s.processOne(ctx, p); procErr != nil {
			_ = s.repo.Update(ctx, p.ID, map[string]any{
				"status": string(models.ScheduledPostFailed),
				"error":  procErr.Error(),
			})
			failed++
			continue
		}
		sent++
	}
	return sent, failed, nil
}

func (s *ScheduledPostService) processOne(ctx context.Context, p *models.ScheduledPost) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.StoragePath, nil)
	if err != nil {
		return fmt.Errorf("build image fetch request: %w", err)
	}
	resp, err := s.http.Do(req)
	if err != nil {
		return fmt.Errorf("fetch stored image: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("fetch stored image: http %d", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read stored image: %w", err)
	}

	post, err := s.social.postImageForUser(ctx, p.UserID, bytes.NewReader(data), p.Filename,
		&p.CaptionFacebook, p.CaptionInstagram, true, p.PostFacebook, p.PostInstagram)
	if err != nil {
		return fmt.Errorf("post image: %w", err)
	}

	if len(p.CategoryIDs) > 0 {
		// Best-effort: tagging failure shouldn't undo a successful post.
		_ = s.captions.SetPostCategories(ctx, post.ID, p.CategoryIDs)
	}

	return s.repo.Update(ctx, p.ID, map[string]any{
		"status":                string(models.ScheduledPostPosted),
		"result_social_post_id": post.ID,
	})
}

func contentTypeForFilename(filename string) string {
	if len(filename) > 4 && (filename[len(filename)-4:] == ".png" || filename[len(filename)-4:] == ".PNG") {
		return "image/png"
	}
	return "image/jpeg"
}
