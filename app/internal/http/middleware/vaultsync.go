package middleware

import (
	"crypto/sha256"
	"crypto/subtle"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	vaultSyncRateWindow = time.Minute
	// vaultSyncRateMax bounds worker traffic, failed attempts included. The
	// worker polls every few seconds at most; this is a brute-force speed bump.
	vaultSyncRateMax = 240
	vaultSyncMaxBody = 8 << 20
)

// VaultSyncAuth guards the /v1/internal routes used by the homelab sync
// worker. It is a static bearer service token (env VAULT_SYNC_TOKEN), compared
// in constant time (digests of fixed length, so length does not leak). When
// the token is empty the routes are disabled and answer 404. The worker acts
// for the single user configured in VAULT_SYNC_USER_ID — the token carries no
// user identity of its own.
func VaultSyncAuth(token string) gin.HandlerFunc {
	enabled := token != ""
	var want [sha256.Size]byte
	if enabled {
		want = sha256.Sum256([]byte(token))
	}

	var mu sync.Mutex
	count := 0
	windowStart := time.Now()

	return func(c *gin.Context) {
		if !enabled {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}

		mu.Lock()
		now := time.Now()
		if now.Sub(windowStart) > vaultSyncRateWindow {
			windowStart = now
			count = 0
		}
		count++
		exceeded := count > vaultSyncRateMax
		mu.Unlock()
		if exceeded {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			return
		}

		const prefix = "Bearer "
		header := c.GetHeader("Authorization")
		if len(header) <= len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		got := sha256.Sum256([]byte(header[len(prefix):]))
		if subtle.ConstantTimeCompare(got[:], want[:]) != 1 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, vaultSyncMaxBody)
		c.Next()
	}
}
