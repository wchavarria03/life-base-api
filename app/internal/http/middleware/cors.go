package middleware

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// CORS sets cross-origin headers so browser clients can call the API.
// Pass allowedOrigins as a list of origins, or []string{"*"} for development.
func CORS(allowedOrigins []string) gin.HandlerFunc {
	allowed := make(map[string]bool, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[o] = true
	}
	if allowed["*"] {
		log.Println("WARNING: ALLOWED_ORIGINS is \"*\" — any site can call this API from a browser. Fine for local dev; set a specific origin in production.")
	}

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if allowed["*"] || allowed[origin] {
			c.Header("Access-Control-Allow-Origin", origin)
		}
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
