package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"life-base-api/app/internal/models"
	"life-base-api/app/internal/services"
)

const (
	maxSocialImageBytes    = 10 << 20 // 10 MB
	defaultSocialListLimit = 50
	maxSocialListLimit     = 200
)

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
	force := c.PostForm("force") == "true"
	toFacebook := c.PostForm("post_facebook") != "false"
	toInstagram := c.PostForm("post_instagram") != "false"
	categoryIDs := c.PostFormArray("category_ids")

	post, err := h.svc.PostImage(c.Request.Context(), file, header.Filename, caption, force, toFacebook, toInstagram)
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
