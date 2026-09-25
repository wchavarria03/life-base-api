package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"life-base-api/app/internal/auth"
)

const (
	rateLimitWindow = time.Minute
	// rateLimitMax is generous for normal single-user use — it exists to
	// blunt a runaway client or abusive script, not to shape normal traffic.
	rateLimitMax = 300
)

type rateWindow struct {
	count int
	start time.Time
}

// RateLimit throttles requests per authenticated user rather than per IP:
// the app sits behind its hosting platform's proxy/load balancer, so
// client IPs aren't reliably distinguishable here (see also
// engine.SetTrustedProxies(nil) in router.go). userID comes from a
// verified JWT, so the key space is bounded by real Supabase users, not
// attacker-controlled — no cleanup sweep needed.
//
// In-memory only: correct for a single instance, not shared across
// horizontally-scaled replicas. Requests without an authenticated user
// (shouldn't happen — this runs after Auth) pass through unthrottled.
func RateLimit() gin.HandlerFunc {
	var mu sync.Mutex
	windows := make(map[string]*rateWindow)

	return func(c *gin.Context) {
		userID := auth.UserIDFromContext(c.Request.Context())
		if userID == "" {
			c.Next()
			return
		}

		mu.Lock()
		w, ok := windows[userID]
		now := time.Now()
		if !ok || now.Sub(w.start) > rateLimitWindow {
			w = &rateWindow{start: now}
			windows[userID] = w
		}
		w.count++
		exceeded := w.count > rateLimitMax
		mu.Unlock()

		if exceeded {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			return
		}
		c.Next()
	}
}
