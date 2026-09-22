package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"mime/multipart"
	"net/http"
	"net/url"
	"time"

	"life-base-api/app/internal/auth"
	"life-base-api/app/internal/models"
)

const graphAPIBase = "https://graph.facebook.com/v21.0"

// duplicatePostWindow is how far back PostImage looks for a same-filename
// post before warning the caller (bypassable with force=true).
const duplicatePostWindow = 24 * time.Hour

// ErrDuplicateSocialPost means a post with this filename already exists
// within duplicatePostWindow and force wasn't set.
var ErrDuplicateSocialPost = errors.New("image was already posted recently")

// ErrInstagramNotRetryable means RetryInstagram was called on a post whose
// Instagram leg can't be retried as-is (Facebook didn't succeed, or
// Instagram already succeeded).
var ErrInstagramNotRetryable = errors.New("instagram leg is not retryable for this post")

// weeklyCaptions mirrors the rotating caption pool in
// streaming-hub/posts/scripts/post-weekly-card.py — used when the caller
// doesn't supply a caption.
var weeklyCaptions = []string{
	"New week, new goals \U0001F4AA Here's what we're doing to keep moving forward — " +
		"every session counts, every pedal stroke adds up. Let's get it done! \U0001F6B4",
	"The plan is set, now it's time to execute \U0001F5D3️ " +
		"Consistency is what separates goals from results — one session at a time \U0001F6B4",
	"Another week, another chance to get better \U0001F525 " +
		"Here's what's on the schedule — no excuses, just work \U0001F4AA",
	"Training doesn't stop \U0001F6B4 Here's the weekly plan — " +
		"every ride, every rep, every rest day has a purpose. Let's go! \U0001F4A5",
	"Small steps every day lead to big results \U0001F4C8 " +
		"Here's how we're building this week — stay consistent, stay focused \U0001F3AF",
}

const socialHashtags = "#wallyrides_cr #cycling #zwift #indoorcycling #costarica #trainingplan #weeklyschedule #gaviaacademy"

// SocialConfig holds the Meta credentials needed to post. The page and IG
// user IDs aren't secret (Meta requires them in every request URL) but are
// still config rather than hardcoded, since this is a general backend and
// not a single-purpose script.
type SocialConfig struct {
	AccessToken   string
	FacebookPage  string
	InstagramUser string
}

// SocialService posts images to Facebook and Instagram via the Meta Graph
// API, persisting a history row for every attempt.
type SocialService struct {
	posts      SocialPostRepository
	cfg        SocialConfig
	httpClient *http.Client
}

func NewSocialService(posts SocialPostRepository, cfg SocialConfig) *SocialService {
	return &SocialService{
		posts:      posts,
		cfg:        cfg,
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
}

// PostImage posts an image to Facebook and, if that succeeds, to Instagram
// (Instagram publishes from the Facebook-hosted photo URL, so it depends on
// the Facebook leg). Every outcome — success, failure, or skipped — is
// persisted; PostImage only returns an error if it couldn't even persist
// the attempt.
func (s *SocialService) PostImage(ctx context.Context, file io.Reader, filename string, customCaption *string, force bool) (*models.SocialPost, error) {
	userID := auth.UserIDFromContext(ctx)
	if userID == "" {
		return nil, fmt.Errorf("no authenticated user")
	}
	if s.cfg.AccessToken == "" {
		return nil, fmt.Errorf("META_ACCESS_TOKEN is not configured")
	}

	if !force {
		since := time.Now().Add(-duplicatePostWindow)
		existing, err := s.posts.FindRecentByFilename(ctx, filename, since)
		if err != nil {
			return nil, fmt.Errorf("check for duplicate post: %w", err)
		}
		if len(existing) > 0 {
			return nil, fmt.Errorf("%w: %q was posted at %s — pass force=true to post it again",
				ErrDuplicateSocialPost, filename, existing[0].CreatedAt.Format(time.RFC3339))
		}
	}

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("read uploaded image: %w", err)
	}

	caption := buildCaption(customCaption)

	input := models.SocialPostInput{
		UserID:   userID,
		Filename: filename,
		Caption:  caption,
	}

	postID, photoID, err := s.postFacebook(ctx, data, filename, caption)
	if err != nil {
		input.FacebookStatus = models.SocialPostFailed
		msg := err.Error()
		input.FacebookError = &msg
		input.InstagramStatus = models.SocialPostSkipped
		return s.posts.Create(ctx, input)
	}
	input.FacebookStatus = models.SocialPostSuccess
	input.FacebookPostID = &postID

	photoURL, permalink, err := s.getFacebookPhotoInfo(ctx, photoID)
	if err != nil {
		input.InstagramStatus = models.SocialPostFailed
		msg := err.Error()
		input.InstagramError = &msg
		return s.posts.Create(ctx, input)
	}
	input.FacebookPhotoURL = &photoURL
	if permalink != "" {
		input.FacebookPermalink = &permalink
	}

	mediaID, err := s.postInstagram(ctx, photoURL, caption)
	if err != nil {
		input.InstagramStatus = models.SocialPostFailed
		msg := err.Error()
		input.InstagramError = &msg
		return s.posts.Create(ctx, input)
	}
	input.InstagramStatus = models.SocialPostSuccess
	input.InstagramMediaID = &mediaID
	if igPermalink, err := s.getInstagramPermalink(ctx, mediaID); err == nil && igPermalink != "" {
		input.InstagramPermalink = &igPermalink
	}

	return s.posts.Create(ctx, input)
}

// RetryInstagram re-attempts the Instagram leg of an existing post, reusing
// the Facebook-hosted photo URL captured the first time — no re-upload
// needed. Only valid when Facebook succeeded and Instagram hasn't already.
func (s *SocialService) RetryInstagram(ctx context.Context, id string) (*models.SocialPost, error) {
	userID := auth.UserIDFromContext(ctx)
	if userID == "" {
		return nil, fmt.Errorf("no authenticated user")
	}

	post, err := s.posts.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("find post: %w", err)
	}
	if post == nil {
		return nil, fmt.Errorf("post not found")
	}
	if post.FacebookStatus != models.SocialPostSuccess || post.FacebookPhotoURL == nil {
		return nil, fmt.Errorf("%w: facebook leg did not succeed — resubmit the image instead", ErrInstagramNotRetryable)
	}
	if post.InstagramStatus == models.SocialPostSuccess {
		return nil, fmt.Errorf("%w: instagram already posted", ErrInstagramNotRetryable)
	}

	fields := map[string]any{}
	mediaID, err := s.postInstagram(ctx, *post.FacebookPhotoURL, post.Caption)
	if err != nil {
		fields["instagram_status"] = models.SocialPostFailed
		fields["instagram_error"] = err.Error()
	} else {
		fields["instagram_status"] = models.SocialPostSuccess
		fields["instagram_media_id"] = mediaID
		fields["instagram_error"] = nil
		if permalink, permErr := s.getInstagramPermalink(ctx, mediaID); permErr == nil && permalink != "" {
			fields["instagram_permalink"] = permalink
		}
	}

	return s.posts.Update(ctx, id, fields)
}

func (s *SocialService) Delete(ctx context.Context, id string) error {
	userID := auth.UserIDFromContext(ctx)
	if userID == "" {
		return fmt.Errorf("no authenticated user")
	}
	return s.posts.Delete(ctx, id)
}

func (s *SocialService) List(ctx context.Context, limit, offset int, status *models.SocialPostStatus) ([]*models.SocialPost, error) {
	return s.posts.List(ctx, limit, offset, status)
}

// buildCaption returns the caller-supplied caption, or picks from the
// rotating pool (seeded by today's date, matching the script) if none given.
func buildCaption(custom *string) string {
	if custom != nil && *custom != "" {
		return *custom
	}
	seed := time.Now().UTC().Format("20060102")
	var n int64
	for _, c := range seed {
		n = n*10 + int64(c-'0')
	}
	body := weeklyCaptions[rand.New(rand.NewSource(n)).Intn(len(weeklyCaptions))]
	return fmt.Sprintf("%s\n\n\U0001F3A5 Live on YouTube: youtube.com/@wallyrides_cr\n\n%s", body, socialHashtags)
}

// graphError mirrors the {"error": {"message": "..."}} shape returned by
// the Meta Graph API on failure.
type graphError struct {
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (s *SocialService) postFacebook(ctx context.Context, imageData []byte, filename, caption string) (postID, photoID string, err error) {
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	_ = w.WriteField("caption", caption)
	_ = w.WriteField("access_token", s.cfg.AccessToken)
	_ = w.WriteField("published", "true")
	part, err := w.CreateFormFile("source", filename)
	if err != nil {
		return "", "", fmt.Errorf("build facebook request: %w", err)
	}
	if _, err := part.Write(imageData); err != nil {
		return "", "", fmt.Errorf("build facebook request: %w", err)
	}
	if err := w.Close(); err != nil {
		return "", "", fmt.Errorf("build facebook request: %w", err)
	}

	url := fmt.Sprintf("%s/%s/photos", graphAPIBase, s.cfg.FacebookPage)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &body)
	if err != nil {
		return "", "", fmt.Errorf("build facebook request: %w", err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	var out struct {
		ID     string `json:"id"`
		PostID string `json:"post_id"`
	}
	if err := s.doGraphRequest(req, &out); err != nil {
		return "", "", fmt.Errorf("facebook: %w", err)
	}
	postID = out.PostID
	if postID == "" {
		postID = out.ID
	}
	return postID, out.ID, nil
}

// getFacebookPhotoInfo fetches the public image URL Instagram needs to
// publish from, plus the Facebook permalink for the post/photo.
func (s *SocialService) getFacebookPhotoInfo(ctx context.Context, photoID string) (photoURL, permalink string, err error) {
	url := fmt.Sprintf("%s/%s?fields=images,link&access_token=%s", graphAPIBase, photoID, s.cfg.AccessToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", "", fmt.Errorf("build facebook photo info request: %w", err)
	}

	var out struct {
		Images []struct {
			Source string `json:"source"`
		} `json:"images"`
		Link string `json:"link"`
	}
	if err := s.doGraphRequest(req, &out); err != nil {
		return "", "", fmt.Errorf("facebook photo info: %w", err)
	}
	if len(out.Images) == 0 {
		return "", "", fmt.Errorf("no images returned for uploaded facebook photo")
	}
	return out.Images[0].Source, out.Link, nil
}

// getInstagramPermalink fetches the public permalink for a published
// Instagram media item. Best-effort: callers treat failure as non-fatal.
func (s *SocialService) getInstagramPermalink(ctx context.Context, mediaID string) (string, error) {
	url := fmt.Sprintf("%s/%s?fields=permalink&access_token=%s", graphAPIBase, mediaID, s.cfg.AccessToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("build instagram permalink request: %w", err)
	}

	var out struct {
		Permalink string `json:"permalink"`
	}
	if err := s.doGraphRequest(req, &out); err != nil {
		return "", fmt.Errorf("instagram permalink: %w", err)
	}
	return out.Permalink, nil
}

func (s *SocialService) postInstagram(ctx context.Context, imageURL, caption string) (string, error) {
	containerURL := fmt.Sprintf("%s/%s/media", graphAPIBase, s.cfg.InstagramUser)
	containerBody := formValues(map[string]string{
		"image_url":    imageURL,
		"caption":      caption,
		"access_token": s.cfg.AccessToken,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, containerURL, containerBody)
	if err != nil {
		return "", fmt.Errorf("build instagram container request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var container struct {
		ID string `json:"id"`
	}
	if err := s.doGraphRequest(req, &container); err != nil {
		return "", fmt.Errorf("instagram container: %w", err)
	}

	publishURL := fmt.Sprintf("%s/%s/media_publish", graphAPIBase, s.cfg.InstagramUser)
	publishBody := formValues(map[string]string{
		"creation_id":  container.ID,
		"access_token": s.cfg.AccessToken,
	})
	req, err = http.NewRequestWithContext(ctx, http.MethodPost, publishURL, publishBody)
	if err != nil {
		return "", fmt.Errorf("build instagram publish request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var media struct {
		ID string `json:"id"`
	}
	if err := s.doGraphRequest(req, &media); err != nil {
		return "", fmt.Errorf("instagram publish: %w", err)
	}
	return media.ID, nil
}

func (s *SocialService) doGraphRequest(req *http.Request, out any) error {
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	var gErr graphError
	if err := json.Unmarshal(body, &gErr); err == nil && gErr.Error.Message != "" {
		return fmt.Errorf("%s", gErr.Error.Message)
	}

	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("http %d: %s", resp.StatusCode, body)
	}

	return json.Unmarshal(body, out)
}

func formValues(fields map[string]string) io.Reader {
	values := url.Values{}
	for k, v := range fields {
		values.Set(k, v)
	}
	return bytes.NewReader([]byte(values.Encode()))
}
