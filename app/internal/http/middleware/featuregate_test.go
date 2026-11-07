package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type fakeChecker struct {
	allowed map[string]bool
	err     error
}

func (f fakeChecker) PageAllowedAny(_ context.Context, _, _ string, keys []string) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	for _, k := range keys {
		if f.allowed[k] {
			return true, nil
		}
	}
	return false, nil
}

func TestKeysForPath(t *testing.T) {
	tests := []struct {
		path    string
		gated   bool
		wantKey string
	}{
		{"/v1/bikes", true, "bikes"},
		{"/v1/bikes/summary", true, "bikes"},
		{"/v1/bikesfoo", false, ""},
		{"/v1/notes/abc/backlinks", true, "notes"},
		{"/v1/dog-recipient-types", true, "dogs"},
		{"/v1/medical-records/1", true, "medical"},
		{"/v1/caption-templates", true, "social-captions"},
		{"/v1/me", false, ""},
		{"/v1/preferences", false, ""},
		{"/v1/kids/profiles", false, ""},
		{"/v1/admin/users", false, ""},
	}
	for _, tt := range tests {
		keys := keysForPath(tt.path)
		if (keys != nil) != tt.gated {
			t.Errorf("%s gated=%v, want %v", tt.path, keys != nil, tt.gated)
			continue
		}
		if tt.gated && keys[0] != tt.wantKey {
			t.Errorf("%s first key %q, want %q", tt.path, keys[0], tt.wantKey)
		}
	}
}

func TestFeatureGate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name    string
		checker fakeChecker
		path    string
		want    int
	}{
		{"enabled feature", fakeChecker{allowed: map[string]bool{"bikes": true}}, "/v1/bikes", http.StatusOK},
		{"disabled feature", fakeChecker{allowed: map[string]bool{"notes": true}}, "/v1/bikes", http.StatusForbidden},
		{"shared endpoint, one key enough", fakeChecker{allowed: map[string]bool{"medical": true}}, "/v1/documents", http.StatusOK},
		{"ungated path ignores checker", fakeChecker{}, "/v1/me", http.StatusOK},
		{"lookup error fails closed", fakeChecker{err: errors.New("db down")}, "/v1/bikes", http.StatusServiceUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.Use(FeatureGate(tt.checker))
			r.Any("/*any", func(c *gin.Context) { c.Status(http.StatusOK) })
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, tt.path, nil))
			if w.Code != tt.want {
				t.Fatalf("status %d, want %d", w.Code, tt.want)
			}
		})
	}
}
