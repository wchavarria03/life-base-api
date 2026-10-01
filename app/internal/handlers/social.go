package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"life-base-api/app/internal/models"
	"life-base-api/app/internal/services"
)

const (
	maxSocialImageBytes    = 10 << 20 // 10 MB
	defaultSocialListLimit = 50
	maxSocialListLimit     = 200
)

// NewSocialHandler constructs a SocialHandler.
func NewSocialHandler(svc SocialPoster, captions CaptionManager) *SocialHandler {
	return &SocialHandler{svc: svc, captions: captions}
}

// Create handles POST /v1/social/posts — multipart upload with an "image"
// file field (required), an optional "caption" text field, an optional
// "force" field ("true" to bypass the duplicate-filename warning),
// optional "post_facebook"/"post_instagram" fields ("false" to skip that
// network — both default to true when absent, so existing callers that
// don't send these fields keep posting to both), and optional repeated
// "category_ids" fields to tag the post.
func (h *SocialHandler) Create(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxSocialImageBytes)

	file, header, err := c.Request.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "image field is required"})
		return
	}
	defer file.Close()

	var caption *string
	if v := c.PostForm("caption"); v != "" {
		caption = &v
	}
	var igCaption *string
	if v := c.PostForm("caption_instagram"); v != "" {
		igCaption = &v
	}
	force := c.PostForm("force") == "true"
	toFacebook := c.PostForm("post_facebook") != "false"
	toInstagram := c.PostForm("post_instagram") != "false"
	categoryIDs := c.PostFormArray("category_ids")

	post, err := h.svc.PostImageWithCaptions(c.Request.Context(), file, header.Filename, caption, igCaption, force, toFacebook, toInstagram)
	if err != nil {
		if errors.Is(err, services.ErrDuplicateSocialPost) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	if len(categoryIDs) > 0 {
		// Best-effort: tagging is metadata, a failure here shouldn't undo an
		// otherwise-successful post.
		if err := h.captions.SetPostCategories(c.Request.Context(), post.ID, categoryIDs); err == nil {
			post.CategoryIDs = categoryIDs
		}
	}

	c.JSON(http.StatusCreated, post)
}

// CreateManual handles POST /v1/social/posts/manual — multipart upload for a
// post with no Graph API call: either a draft (post_status=draft, not yet
// posted or scheduled) or a historical post logged after the fact
// (post_status=logged, optionally with the live Facebook/Instagram
// permalinks). Required fields: "image", "caption", "post_status"
// ("draft"|"logged"). "posted_at" (RFC3339) is required for "logged".
func (h *SocialHandler) CreateManual(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxSocialImageBytes)

	file, header, err := c.Request.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "image field is required"})
		return
	}
	defer func() { _ = file.Close() }()

	caption := c.PostForm("caption")
	if caption == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "caption is required"})
		return
	}
	var igCaption *string
	if v := c.PostForm("caption_instagram"); v != "" {
		igCaption = &v
	}

	status := models.SocialPostLifecycle(c.PostForm("post_status"))
	toFacebook := c.PostForm("post_facebook") == "true"
	toInstagram := c.PostForm("post_instagram") == "true"

	in := services.ManualPostInput{
		FacebookCaption:  caption,
		InstagramCaption: igCaption,
		PostFacebook:     toFacebook,
		PostInstagram:    toInstagram,
		Status:           status,
	}
	if v := c.PostForm("facebook_permalink"); v != "" {
		in.FacebookPermalink = &v
	}
	if v := c.PostForm("instagram_permalink"); v != "" {
		in.InstagramPermalink = &v
	}
	if status == models.SocialPostLifecycleLogged {
		v := c.PostForm("posted_at")
		postedAt, err := time.Parse(time.RFC3339, v)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "posted_at (RFC3339) is required to log an already-posted post"})
			return
		}
		in.PostedAt = &postedAt
	}

	post, err := h.svc.CreateManual(c.Request.Context(), file, header.Filename, in)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	categoryIDs := c.PostFormArray("category_ids")
	if len(categoryIDs) > 0 {
		if err := h.captions.SetPostCategories(c.Request.Context(), post.ID, categoryIDs); err == nil {
			post.CategoryIDs = categoryIDs
		}
	}

	c.JSON(http.StatusCreated, post)
}

// UpdatePost handles PATCH /v1/social/posts/:id — partial edit of a
// tracked post's caption(s), posted date, permalinks, categories, and
// (multipart, optional "image" field) its stored image. Editing a post that
// isn't a draft never touches the live Facebook/Instagram post — only our
// tracking record — and marks the row `edited`.
func (h *SocialHandler) UpdatePost(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxSocialImageBytes)
	_ = c.Request.ParseMultipartForm(maxSocialImageBytes)

	in := services.UpdatePostInput{}
	if v := c.PostForm("caption"); v != "" {
		in.Caption = &v
	}
	if v := c.PostForm("caption_instagram"); v != "" {
		in.CaptionInstagram = &v
	}
	if v := c.PostForm("facebook_permalink"); v != "" {
		in.FacebookPermalink = &v
	}
	if v := c.PostForm("instagram_permalink"); v != "" {
		in.InstagramPermalink = &v
	}
	if v := c.PostForm("posted_at"); v != "" {
		postedAt, err := time.Parse(time.RFC3339, v)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "posted_at must be RFC3339"})
			return
		}
		in.PostedAt = &postedAt
	}
	if file, header, err := c.Request.FormFile("image"); err == nil {
		defer func() { _ = file.Close() }()
		in.Image = file
		in.ImageFilename = header.Filename
	}

	post, err := h.svc.UpdatePost(c.Request.Context(), c.Param("id"), in)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	// The edit form always re-submits the full category selection (including
	// empty, meaning "no categories"), so this is always a full replace.
	ids := c.PostFormArray("category_ids")
	if err := h.captions.SetPostCategories(c.Request.Context(), post.ID, ids); err == nil {
		post.CategoryIDs = ids
	}

	c.JSON(http.StatusOK, post)
}

// MarkDraftPosted handles POST /v1/social/posts/:id/mark-posted — flips a
// draft to logged once it's been posted by hand outside the app (e.g. an
// Instagram/Facebook Story, which this app has no way to post to directly).
// JSON body: post_facebook, post_instagram (bool), posted_at (RFC3339,
// required), facebook_permalink/instagram_permalink (optional).
func (h *SocialHandler) MarkDraftPosted(c *gin.Context) {
	var req struct {
		PostFacebook       bool    `json:"post_facebook"`
		PostInstagram      bool    `json:"post_instagram"`
		PostedAt           string  `json:"posted_at" binding:"required"`
		FacebookPermalink  *string `json:"facebook_permalink,omitempty"`
		InstagramPermalink *string `json:"instagram_permalink,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	postedAt, err := time.Parse(time.RFC3339, req.PostedAt)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "posted_at must be RFC3339"})
		return
	}

	post, err := h.svc.MarkDraftPosted(c.Request.Context(), c.Param("id"), services.MarkDraftPostedInput{
		PostFacebook:       req.PostFacebook,
		PostInstagram:      req.PostInstagram,
		PostedAt:           postedAt,
		FacebookPermalink:  req.FacebookPermalink,
		InstagramPermalink: req.InstagramPermalink,
	})
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, post)
}

// List handles GET /v1/social/posts?limit=&offset=&status=.
func (h *SocialHandler) List(c *gin.Context) {
	limit := defaultSocialListLimit
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	if limit > maxSocialListLimit {
		limit = maxSocialListLimit
	}

	offset := 0
	if v := c.Query("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}

	var status *models.SocialPostStatus
	if v := c.Query("status"); v != "" {
		s := models.SocialPostStatus(v)
		status = &s
	}

	posts, err := h.svc.List(c.Request.Context(), limit, offset, status)
	if err != nil {
		internalError(c, err)
		return
	}

	if catsByPost, err := h.captions.ListPostCategoryIDs(c.Request.Context()); err == nil {
		for _, p := range posts {
			p.CategoryIDs = catsByPost[p.ID]
		}
	}

	c.JSON(http.StatusOK, posts)
}

// RetryInstagram handles POST /v1/social/posts/:id/retry-instagram.
func (h *SocialHandler) RetryInstagram(c *gin.Context) {
	post, err := h.svc.RetryInstagram(c.Request.Context(), c.Param("id"))
	if err != nil {
		if errors.Is(err, services.ErrInstagramNotRetryable) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, post)
}

// Delete handles DELETE /v1/social/posts/:id — removes the history row only,
// does not touch the live Facebook/Instagram post.
func (h *SocialHandler) Delete(c *gin.Context) {
	if err := h.svc.Delete(c.Request.Context(), c.Param("id")); err != nil {
		internalError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
