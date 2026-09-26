// Package middleware provides the Gin HTTP middleware chain the API server
// applies to incoming requests (auth, rate limiting, CORS, and so on).
package middleware

import (
	"context"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"life-base-api/app/internal/auth"
)

// AuditWriter persists one audit entry — implemented by
// *services.AuditLogService, kept as a narrow interface here so this
// low-level middleware package doesn't need to import the services package.
type AuditWriter interface {
	Create(ctx context.Context, method, path string, status int) error
}

// AuditLog logs every DELETE request with the acting user, resource path,
// and outcome, and (when writer is non-nil) persists it to the audit_logs
// table so it's queryable instead of living only in the process log
// stream. This is the one place all destructive actions funnel through, so
// it's a single choke point rather than instrumenting every handler
// individually. Authorization for the delete itself is decided by Postgres
// RLS, not this middleware — this only records who asked and what
// happened, for forensics if something is later contested.
func AuditLog(writer AuditWriter) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if c.Request.Method != http.MethodDelete {
			return
		}

		userID := auth.UserIDFromContext(c.Request.Context())
		path := c.FullPath()
		status := c.Writer.Status()
		log.Printf("audit: delete user=%s path=%s status=%d", userID, path, status)

		if writer == nil {
			return
		}
		if err := writer.Create(c.Request.Context(), c.Request.Method, path, status); err != nil {
			log.Printf("audit: failed to persist entry: %v", err)
		}
	}
}
