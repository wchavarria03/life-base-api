package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"life-base-api/app/internal/auth"
)

// RequireRole rejects the request with 403 unless the caller's JWT
// user_role claim (see Auth) is one of roles. Must run after Auth.
func RequireRole(roles ...string) gin.HandlerFunc {
	allowed := make(map[string]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}

	return func(c *gin.Context) {
		if !allowed[auth.RoleFromContext(c.Request.Context())] {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient role"})
			return
		}
		c.Next()
	}
}
