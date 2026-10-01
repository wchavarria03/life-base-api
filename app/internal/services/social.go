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
	"strconv"
	"time"

	"life-base-api/app/internal/auth"
	"life-base-api/app/internal/models"
	supabaserepo "life-base-api/app/internal/repositories/supabase"
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

// socialManualImagesBucket is the same public bucket scheduled posts use —
// public so a logged/draft thumbnail is directly embeddable, and there's no
// reason to split tracking images across two buckets.
const socialManualImagesBucket = scheduledImagesBucket

// SocialService posts images to Facebook and Instagram via the Meta Graph
// API, persisting a history row for every attempt.
type SocialService struct {
	posts      SocialPostRepository
	storage    *supabaserepo.StorageRepository
	cfg        SocialConfig
	httpClient *http.Client
}

// NewSocialService constructs a SocialService.
func NewSocialService(posts SocialPostRepository, storage *supabaserepo.StorageRepository, cfg SocialConfig) *SocialService {
	return &SocialService{
		posts:      posts,
		storage:    storage,
		cfg:        cfg,
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
}

// ManualPostInput describes a draft or already-posted-elsewhere social post
// recorded with no Graph API call ever made.
type ManualPostInput struct {
	FacebookCaption    string
	InstagramCaption   *string
	PostFacebook       bool
	PostInstagram      bool
	Status             models.SocialPostLifecycle // draft | logged
	PostedAt           *time.Time                 // required for logged, ignored for draft
	FacebookPermalink  *string
	InstagramPermalink *string
}

// CreateManual uploads image to our own storage and inserts a social_posts
// row with no Graph API call — used both for "save as draft" (status
// draft, nothing has happened yet) and "log a post that already happened
// outside this app" (status logged, optionally with the live permalinks).
func (s *SocialService) CreateManual(ctx context.Context, file io.Reader, filename string, in ManualPostInput) (*models.SocialPost, error) {
	userID := auth.UserIDFromContext(ctx)
	if userID == "" {
		return nil, fmt.Errorf("no authenticated user")
	}
	if in.Status != models.SocialPostLifecycleDraft && in.Status != models.SocialPostLifecycleLogged {
		return nil, fmt.Errorf("invalid post_status %q", in.Status)
	}
	if in.Status == models.SocialPostLifecycleLogged && !in.PostFacebook && !in.PostInstagram {
		return nil, fmt.Errorf("select at least one network this post actually went to")
	}

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("read uploaded image: %w", err)
	}
	path := fmt.Sprintf("%s/manual/%d-%s", userID, time.Now().UnixNano(), filename)
	publicURL, err := s.storage.UploadObject(ctx, socialManualImagesBucket, path, data, contentTypeForFilename(filename))
	if err != nil {
		return nil, fmt.Errorf("upload image: %w", err)
	}

	fbStatus, igStatus := models.SocialPostSkipped, models.SocialPostSkipped
	var fbPermalink, igPermalink *string
	if in.Status == models.SocialPostLifecycleLogged {
		if in.PostFacebook {
			fbStatus = models.SocialPostSuccess
			fbPermalink = in.FacebookPermalink
		}
		if in.PostInstagram {
			igStatus = models.SocialPostSuccess
			igPermalink = in.InstagramPermalink
		}
	}

	return s.posts.Create(ctx, models.SocialPostInput{
		UserID:             userID,
		Filename:           filename,
		Caption:            in.FacebookCaption,
		CaptionInstagram:   in.InstagramCaption,
		FacebookStatus:     fbStatus,
		FacebookPhotoURL:   &publicURL,
		FacebookPermalink:  fbPermalink,
		InstagramStatus:    igStatus,
		InstagramPermalink: igPermalink,
		PostStatus:         in.Status,
		PostedAt:           in.PostedAt,
		ImageStoragePath:   &path,
		PostFacebook:       in.PostFacebook,
		PostInstagram:      in.PostInstagram,
	})
}

// UpdatePostInput is the partial-update shape for UpdatePost — nil fields
// are left unchanged. Image, when set, replaces the stored photo (and
// removes the previous object if we owned it); for a 'posted' row this only
// changes our tracking record, never the live Facebook/Instagram post.
type UpdatePostInput struct {
	Caption            *string
	CaptionInstagram   *string
	PostedAt           *time.Time
	FacebookPermalink  *string
	InstagramPermalink *string
	Image              io.Reader
	ImageFilename      string
}

// UpdatePost applies a partial edit to an existing social post. Editing any
// field on a post that isn't a draft (i.e. one that was actually posted or
// logged as posted) marks it `edited` — surfaced in the UI as a reminder
// that the live Facebook/Instagram post itself is untouched.
func (s *SocialService) UpdatePost(ctx context.Context, id string, in UpdatePostInput) (*models.SocialPost, error) {
	existing, err := s.posts.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, fmt.Errorf("post not found")
	}

	fields := map[string]any{}
	if in.Caption != nil {
		fields["caption"] = *in.Caption
	}
	if in.CaptionInstagram != nil {
		fields["caption_instagram"] = *in.CaptionInstagram
	}
	if in.PostedAt != nil {
		fields["posted_at"] = in.PostedAt.UTC().Format(time.RFC3339)
	}
	if in.FacebookPermalink != nil {
		fields["facebook_permalink"] = *in.FacebookPermalink
	}
	if in.InstagramPermalink != nil {
		fields["instagram_permalink"] = *in.InstagramPermalink
	}
	if in.Image != nil {
		data, err := io.ReadAll(in.Image)
		if err != nil {
			return nil, fmt.Errorf("read uploaded image: %w", err)
		}
		path := fmt.Sprintf("%s/manual/%d-%s", existing.UserID, time.Now().UnixNano(), in.ImageFilename)
		publicURL, err := s.storage.UploadObject(ctx, socialManualImagesBucket, path, data, contentTypeForFilename(in.ImageFilename))
		if err != nil {
			return nil, fmt.Errorf("upload image: %w", err)
		}
		if existing.ImageStoragePath != nil {
			_ = s.storage.DeleteObject(ctx, socialManualImagesBucket, *existing.ImageStoragePath)
		}
		fields["facebook_photo_url"] = publicURL
		fields["image_storage_path"] = path
	}
	if len(fields) > 0 && existing.PostStatus != models.SocialPostLifecycleDraft {
		fields["edited"] = true
	}
	if len(fields) == 0 {
		return existing, nil
	}
	return s.posts.Update(ctx, id, fields)
}

// MarkDraftPostedInput is the input for MarkDraftPosted.
type MarkDraftPostedInput struct {
	PostFacebook       bool
	PostInstagram      bool
	PostedAt           time.Time
	FacebookPermalink  *string
	InstagramPermalink *string
}

// MarkDraftPosted converts a draft into a logged post — used when a draft
// (e.g. an Instagram/Facebook Story, which this app can't post to directly)
// was posted by hand outside the app, and the user comes back to flag it.
// Only valid on a row whose post_status is still 'draft'.
func (s *SocialService) MarkDraftPosted(ctx context.Context, id string, in MarkDraftPostedInput) (*models.SocialPost, error) {
	existing, err := s.posts.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, fmt.Errorf("post not found")
	}
	if existing.PostStatus != models.SocialPostLifecycleDraft {
		return nil, fmt.Errorf("only a draft can be marked posted")
	}
	if !in.PostFacebook && !in.PostInstagram {
		return nil, fmt.Errorf("select at least one network this post actually went to")
	}

	fbStatus, igStatus := models.SocialPostSkipped, models.SocialPostSkipped
	if in.PostFacebook {
		fbStatus = models.SocialPostSuccess
	}
	if in.PostInstagram {
		igStatus = models.SocialPostSuccess
	}

	fields := map[string]any{
		"post_status":      models.SocialPostLifecycleLogged,
		"posted_at":        in.PostedAt.UTC().Format(time.RFC3339),
		"post_facebook":    in.PostFacebook,
		"post_instagram":   in.PostInstagram,
		"facebook_status":  fbStatus,
		"instagram_status": igStatus,
	}
	if in.FacebookPermalink != nil {
		fields["facebook_permalink"] = *in.FacebookPermalink
	}
	if in.InstagramPermalink != nil {
		fields["instagram_permalink"] = *in.InstagramPermalink
	}
	return s.posts.Update(ctx, id, fields)
}

// PostImage posts an image to the selected networks (toFacebook/toInstagram).
// Instagram always publishes from a Facebook-hosted photo URL, so the
// Facebook upload happens regardless of toFacebook — when toFacebook is
// false, it's uploaded unpublished (never becomes a public Facebook Page
// post), purely to get Instagram a URL to publish from. Every outcome —
// success, failure, or skipped (not selected) — is persisted; PostImage
// only returns an error if it couldn't even persist the attempt.
func (s *SocialService) PostImage(ctx context.Context, file io.Reader, filename string, customCaption *string, force, toFacebook, toInstagram bool) (*models.SocialPost, error) {
	return s.PostImageWithCaptions(ctx, file, filename, customCaption, nil, force, toFacebook, toInstagram)
}

// PostImageWithCaptions is PostImage with an optional distinct Instagram
// caption — when igCaption is nil or empty, both networks use the same
// (Facebook) caption, same as PostImage.
func (s *SocialService) PostImageWithCaptions(ctx context.Context, file io.Reader, filename string, fbCaption, igCaption *string, force, toFacebook, toInstagram bool) (*models.SocialPost, error) {
	userID := auth.UserIDFromContext(ctx)
	if userID == "" {
		return nil, fmt.Errorf("no authenticated user")
	}
	return s.postImageForUser(ctx, userID, file, filename, fbCaption, igCaption, force, toFacebook, toInstagram)
}

// postImageForUser is PostImageWithCaptions with the acting user supplied
// directly rather than read from context — used by the scheduled-posts
// sender, which runs outside any per-request user JWT.
func (s *SocialService) postImageForUser(ctx context.Context, userID string, file io.Reader, filename string, fbCaption, igCaption *string, force, toFacebook, toInstagram bool) (*models.SocialPost, error) {
	if s.cfg.AccessToken == "" {
		return nil, fmt.Errorf("META_ACCESS_TOKEN is not configured")
	}
	if !toFacebook && !toInstagram {
		return nil, fmt.Errorf("select at least one network to post to")
	}

	if err := s.checkDuplicate(ctx, filename, force); err != nil {
		return nil, err
	}

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("read uploaded image: %w", err)
	}

	caption := buildCaption(fbCaption)
	igText := caption
	if igCaption != nil && *igCaption != "" {
		igText = *igCaption
	}

	now := time.Now().UTC()
	input := models.SocialPostInput{
		UserID:        userID,
		Filename:      filename,
		Caption:       caption,
		PostStatus:    models.SocialPostLifecyclePosted,
		PostedAt:      &now,
		PostFacebook:  toFacebook,
		PostInstagram: toInstagram,
	}
	if igText != caption {
		input.CaptionInstagram = &igText
	}

	// Facebook upload always happens — even Instagram-only posts need the
	// resulting hosted photo URL. publish=toFacebook controls whether it
	// actually becomes a public Facebook Page post.
	postID, photoID, err := s.postFacebook(ctx, data, filename, caption, toFacebook)
	if err != nil {
		input.FacebookStatus, input.FacebookError = networkOutcome(toFacebook, err.Error())
		input.InstagramStatus, input.InstagramError = networkOutcome(toInstagram, "could not prepare image: "+err.Error())
		return s.posts.Create(ctx, input)
	}

	if toFacebook {
		input.FacebookStatus = models.SocialPostSuccess
		input.FacebookPostID = &postID
	} else {
		input.FacebookStatus, input.FacebookError = networkOutcome(false, "")
	}

	photoURL, permalink, err := s.getFacebookPhotoInfo(ctx, photoID)
	if err != nil {
		input.InstagramStatus, input.InstagramError = networkOutcome(toInstagram, err.Error())
		return s.posts.Create(ctx, input)
	}
	input.FacebookPhotoURL = &photoURL
	if toFacebook && permalink != "" {
		input.FacebookPermalink = &permalink
	}

	if !toInstagram {
		input.InstagramStatus, input.InstagramError = networkOutcome(false, "")
		return s.posts.Create(ctx, input)
	}

	s.fillInstagramOutcome(ctx, &input, photoURL, igText)
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
	// FacebookPhotoURL exists whenever the upload itself succeeded, whether
	// or not it was published (an Instagram-only post uploads unpublished
	// but still gets a usable URL) — that's the only real precondition here.
	if post.FacebookStatus == models.SocialPostFailed || post.FacebookPhotoURL == nil {
		return nil, fmt.Errorf("%w: facebook leg did not succeed — resubmit the image instead", ErrInstagramNotRetryable)
	}
	if post.InstagramStatus == models.SocialPostSuccess {
		return nil, fmt.Errorf("%w: instagram already posted", ErrInstagramNotRetryable)
	}

	igCaption := post.Caption
	if post.CaptionInstagram != nil && *post.CaptionInstagram != "" {
		igCaption = *post.CaptionInstagram
	}

	fields := map[string]any{}
	mediaID, err := s.postInstagram(ctx, *post.FacebookPhotoURL, igCaption)
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
	if existing, err := s.posts.FindByID(ctx, id); err == nil && existing != nil && existing.ImageStoragePath != nil {
		_ = s.storage.DeleteObject(ctx, socialManualImagesBucket, *existing.ImageStoragePath)
	}
	return s.posts.Delete(ctx, id)
}

func (s *SocialService) List(ctx context.Context, limit, offset int, status *models.SocialPostStatus) ([]*models.SocialPost, error) {
	return s.posts.List(ctx, limit, offset, status)
}

// fillInstagramOutcome publishes to Instagram from photoURL and records the
// result on input — success (with permalink, best-effort) or failure.
func (s *SocialService) fillInstagramOutcome(ctx context.Context, input *models.SocialPostInput, photoURL, caption string) {
	mediaID, err := s.postInstagram(ctx, photoURL, caption)
	if err != nil {
		input.InstagramStatus = models.SocialPostFailed
		msg := err.Error()
		input.InstagramError = &msg
		return
	}
	input.InstagramStatus = models.SocialPostSuccess
	input.InstagramMediaID = &mediaID
	if igPermalink, err := s.getInstagramPermalink(ctx, mediaID); err == nil && igPermalink != "" {
		input.InstagramPermalink = &igPermalink
	}
}

// checkDuplicate errors if filename was posted within duplicatePostWindow,
// unless force is set.
func (s *SocialService) checkDuplicate(ctx context.Context, filename string, force bool) error {
	if force {
		return nil
	}
	since := time.Now().Add(-duplicatePostWindow)
	existing, err := s.posts.FindRecentByFilename(ctx, filename, since)
	if err != nil {
		return fmt.Errorf("check for duplicate post: %w", err)
	}
	if len(existing) > 0 {
		return fmt.Errorf("%w: %q was posted at %s — pass force=true to post it again",
			ErrDuplicateSocialPost, filename, existing[0].CreatedAt.Format(time.RFC3339))
	}
	return nil
}

// networkOutcome returns the status/error pair for a network leg that
// didn't succeed: "skipped, not selected" when the caller didn't choose
// that network, or "failed, failedMsg" when they did but it errored.
func networkOutcome(selected bool, failedMsg string) (models.SocialPostStatus, *string) {
	if !selected {
		msg := "not selected"
		return models.SocialPostSkipped, &msg
	}
	return models.SocialPostFailed, &failedMsg
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
	body := weeklyCaptions[rand.New(rand.NewSource(n)).Intn(len(weeklyCaptions))] //nolint:gosec // deterministic by design (seeded from today's date), not a security context
	return fmt.Sprintf("%s\n\n\U0001F3A5 Live on YouTube: youtube.com/@wallyrides_cr\n\n%s", body, socialHashtags)
}

// graphError mirrors the {"error": {"message": "..."}} shape returned by
// the Meta Graph API on failure.
type graphError struct {
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (s *SocialService) postFacebook(ctx context.Context, imageData []byte, filename, caption string, publish bool) (postID, photoID string, err error) {
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	_ = w.WriteField("caption", caption)
	_ = w.WriteField("access_token", s.cfg.AccessToken)
	_ = w.WriteField("published", strconv.FormatBool(publish))
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

	if err := s.waitForContainerReady(ctx, container.ID); err != nil {
		return "", err
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

// waitForContainerReady polls an Instagram media container's status_code
// until it's FINISHED (ready to publish) or ERROR/EXPIRED, or times out.
// Meta downloads/validates the image asynchronously after container
// creation — publishing before it finishes fails with "Media ID is not
// available", which this avoids.
func (s *SocialService) waitForContainerReady(ctx context.Context, containerID string) error {
	statusURL := fmt.Sprintf("%s/%s?fields=status_code&access_token=%s", graphAPIBase, containerID, url.QueryEscape(s.cfg.AccessToken))
	const maxAttempts = 15
	for attempt := 0; attempt < maxAttempts; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, statusURL, nil)
		if err != nil {
			return fmt.Errorf("build container status request: %w", err)
		}
		var status struct {
			StatusCode string `json:"status_code"`
		}
		if err := s.doGraphRequest(req, &status); err != nil {
			return fmt.Errorf("instagram container status: %w", err)
		}
		switch status.StatusCode {
		case "FINISHED":
			return nil
		case "ERROR", "EXPIRED":
			return fmt.Errorf("instagram container failed to process (status %s)", status.StatusCode)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(1500 * time.Millisecond):
		}
	}
	return fmt.Errorf("instagram container did not finish processing in time")
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
