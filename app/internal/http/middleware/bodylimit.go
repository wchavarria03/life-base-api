package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// defaultMaxBodyBytes caps request bodies globally. Matches the limit the
// upload/social-image handlers already set on themselves — http.MaxBytesReader
// doesn't compose when wrapped twice (the smaller inner cap always wins), so
// this can't be smaller than their limit without breaking those endpoints.
const defaultMaxBodyBytes = 10 << 20 // 10 MB

// BodyLimit rejects request bodies over defaultMaxBodyBytes before they're
// read, so an oversized payload can't be used to exhaust server memory.
func BodyLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, defaultMaxBodyBytes)
		c.Next()
	}
}
