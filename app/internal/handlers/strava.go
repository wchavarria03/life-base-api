package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func NewStravaHandler(svc StravaManager) *StravaHandler {
	return &StravaHandler{svc: svc}
}

// Authorize handles GET /v1/bikes/strava/authorize?redirect_uri=...
func (h *StravaHandler) Authorize(c *gin.Context) {
	authURL, err := h.svc.Authorize(c.Request.Context(), c.Query("redirect_uri"))
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"auth_url": authURL})
}

// Callback handles GET /v1/bikes/strava/callback?code=&state=
func (h *StravaHandler) Callback(c *gin.Context) {
	conn, err := h.svc.Callback(c.Request.Context(), c.Query("code"), c.Query("state"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, conn)
}

// Status handles GET /v1/bikes/strava/status.
func (h *StravaHandler) Status(c *gin.Context) {
	conn, err := h.svc.Status(c.Request.Context())
	if err != nil {
		internalError(c, err)
		return
	}
	if conn == nil {
		c.JSON(http.StatusOK, gin.H{"connected": false})
		return
	}
	c.JSON(http.StatusOK, gin.H{"connected": true, "connection": conn})
}

// Disconnect handles DELETE /v1/bikes/strava.
func (h *StravaHandler) Disconnect(c *gin.Context) {
	if err := h.svc.Disconnect(c.Request.Context()); err != nil {
		internalError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// Activities handles GET /v1/bikes/strava/activities?after=&before=&page=&per_page=
// (after/before are unix timestamps) — returns a preview, nothing is saved.
func (h *StravaHandler) Activities(c *gin.Context) {
	after := parseUnixQuery(c.Query("after"))
	before := parseUnixQuery(c.Query("before"))
	page, _ := strconv.Atoi(c.Query("page"))
	perPage, _ := strconv.Atoi(c.Query("per_page"))

	activities, err := h.svc.FetchActivities(c.Request.Context(), after, before, page, perPage)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, activities)
}

func parseUnixQuery(v string) *time.Time {
	if v == "" {
		return nil
	}
	sec, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return nil
	}
	t := time.Unix(sec, 0)
	return &t
}
