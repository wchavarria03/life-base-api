package handlers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// internalError logs the underlying error server-side and responds with a
// generic message. The app talks to PostgREST directly, so err often embeds
// internal detail (table/column/constraint names, SQL hints) that shouldn't
// reach the client.
func internalError(c *gin.Context, err error) {
	log.Printf("internal error: %s %s: %v", c.Request.Method, c.FullPath(), err)
	c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
}
