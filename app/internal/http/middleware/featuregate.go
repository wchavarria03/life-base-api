package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"life-base-api/app/internal/auth"
)

// PageChecker reports whether an account may use at least one of the page
// keys (role matrix plus per-account overrides).
type PageChecker interface {
	PageAllowedAny(ctx context.Context, userID, role string, keys []string) (bool, error)
}

var financeKeys = []string{"finance", "accounts", "loans", "envelopes", "reminders", "transfers", "work-hours", "budgets", "import", "categories"}

// featureRoutes maps an API path prefix to the page keys that unlock it. A
// request is allowed when the account has ANY listed key, because several
// pages share data endpoints (e.g. documents back attachments on notes,
// bikes and medical records). Prefixes not listed here (preferences, /me,
// kids, admin, share resolution, ...) are not feature-gated; /v1/admin has
// its own role check and /v1/kids its own child-profile checks.
var featureRoutes = []struct {
	prefix string
	keys   []string
}{
	{"/v1/accounts", financeKeys},
	{"/v1/transactions", financeKeys},
	{"/v1/reports", financeKeys},
	{"/v1/transfers", financeKeys},
	{"/v1/import", financeKeys},
	{"/v1/categories", financeKeys},
	{"/v1/category-rules", financeKeys},
	{"/v1/salary-profile", financeKeys},
	{"/v1/envelopes", []string{"envelopes"}},
	{"/v1/budgets", []string{"budgets"}},
	{"/v1/reminders", []string{"reminders"}},
	{"/v1/social", []string{"social", "social-captions"}},
	{"/v1/caption-", []string{"social-captions", "social"}},
	{"/v1/bikes", []string{"bikes"}},
	{"/v1/gear", []string{"gear"}},
	{"/v1/bottles", []string{"bottles"}},
	{"/v1/supplies", []string{"supplies"}},
	{"/v1/tasks", []string{"home", "household", "house", "todo", "notes"}},
	{"/v1/task-lists", []string{"home", "household", "house", "todo"}},
	{"/v1/shared-lists", []string{"home", "household", "house", "todo"}},
	{"/v1/house-timers", []string{"house", "home"}},
	{"/v1/notes", []string{"notes"}},
	{"/v1/documents", []string{"documents", "notes", "bikes", "medical", "home"}},
	{"/v1/share-links", []string{"notes", "bikes", "medical"}},
	{"/v1/dogs", []string{"dogs"}},
	{"/v1/dog-", []string{"dogs"}},
	{"/v1/medical-", []string{"medical"}},
}

// keysForPath returns the page keys guarding path, or nil when the path is
// not feature-gated. Prefix matches respect path-segment boundaries so
// "/v1/notes" does not capture "/v1/notesfoo".
func keysForPath(path string) []string {
	for _, r := range featureRoutes {
		if !strings.HasPrefix(path, r.prefix) {
			continue
		}
		if strings.HasSuffix(r.prefix, "-") || len(path) == len(r.prefix) || path[len(r.prefix)] == '/' {
			return r.keys
		}
	}
	return nil
}

// FeatureGate refuses requests to a feature the caller's account has turned
// off, so disabling a page in the admin view also closes its API. Must run
// after Auth. An error resolving access fails closed.
func FeatureGate(checker PageChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		keys := keysForPath(c.Request.URL.Path)
		if keys == nil {
			c.Next()
			return
		}
		ctx := c.Request.Context()
		ok, err := checker.PageAllowedAny(ctx, auth.UserIDFromContext(ctx), auth.RoleFromContext(ctx), keys)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "could not verify feature access"})
			return
		}
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "feature disabled for this account"})
			return
		}
		c.Next()
	}
}
