package middleware

import "github.com/gin-gonic/gin"

// SecurityHeaders sets a small set of defensive response headers. This is a
// JSON API (no HTML rendered here), so the main value is nosniff — but
// they're cheap and standard practice regardless of what a browser client
// does with the response. Strict-Transport-Security is ignored by browsers
// on a plain-HTTP connection, so it's safe to always send.
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "no-referrer")
		c.Header("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		c.Next()
	}
}
