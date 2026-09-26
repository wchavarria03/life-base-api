package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"life-base-api/app/internal/auth"
)

const maxScheduledImageBytes = 10 << 20 // 10 MB

// NewScheduledPostHandler constructs a ScheduledPostHandler.
func NewScheduledPostHandler(svc ScheduledPostManager) *ScheduledPostHandler {
	return &ScheduledPostHandler{svc: svc}
}

// Create handles POST /v1/social/scheduled — multipart upload with an
// "image" file field, "caption_facebook" (required), optional
// "caption_instagram", "post_facebook"/"post_instagram" ("false" to skip),
// repeated "category_ids", and "scheduled_at" (RFC3339, required).
func (h *ScheduledPostHandler) Create(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxScheduledImageBytes)

	file, header, err := c.Request.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "image field is required"})
		return
	}
	defer func() { _ = file.Close() }()

	fbCaption := c.PostForm("caption_facebook")
	if fbCaption == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "caption_facebook is required"})
		return
	}
	var igCaption *string
	if v := c.PostForm("caption_instagram"); v != "" {
		igCaption = &v
	}

	scheduledAtStr := c.PostForm("scheduled_at")
	scheduledAt, err := time.Parse(time.RFC3339, scheduledAtStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "scheduled_at must be an RFC3339 timestamp"})
		return
	}

	toFacebook := c.PostForm("post_facebook") != "false"
	toInstagram := c.PostForm("post_instagram") != "false"
	categoryIDs := c.PostFormArray("category_ids")

	post, err := h.svc.Create(c.Request.Context(), file, header.Filename, fbCaption, igCaption, toFacebook, toInstagram, categoryIDs, scheduledAt)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, post)
}

// List handles GET /v1/social/scheduled.
func (h *ScheduledPostHandler) List(c *gin.Context) {
	posts, err := h.svc.List(c.Request.Context())
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, posts)
}

// Delete handles DELETE /v1/social/scheduled/:id — cancels a pending post.
func (h *ScheduledPostHandler) Delete(c *gin.Context) {
	if err := h.svc.Cancel(c.Request.Context(), c.Param("id")); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// CheckNow handles POST /v1/social/scheduled/check-now — sends the
// caller's own due scheduled posts immediately, without waiting for the
// hourly cron.
func (h *ScheduledPostHandler) CheckNow(c *gin.Context) {
	userID := auth.UserIDFromContext(c.Request.Context())
	sent, failed, err := h.svc.ProcessDue(c.Request.Context(), userID)
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"sent": sent, "failed": failed})
}
