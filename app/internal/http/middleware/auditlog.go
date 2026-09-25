package middleware

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"life-base-api/app/internal/auth"
)

// AuditLog logs every DELETE request with the acting user, resource path,
// and outcome. This is the one place all destructive actions funnel
// through, so it's a single choke point rather than instrumenting every
// handler individually. Authorization for the delete itself is decided by
// Postgres RLS, not this middleware — this only records who asked and
// what happened, for forensics if something is later contested.
func AuditLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if c.Request.Method != http.MethodDelete {
			return
		}

		userID := auth.UserIDFromContext(c.Request.Context())
		log.Printf("audit: delete user=%s path=%s status=%d", userID, c.FullPath(), c.Writer.Status())
	}
}
