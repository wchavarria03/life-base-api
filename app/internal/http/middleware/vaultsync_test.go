package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestVaultSyncAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name   string
		token  string
		header string
		want   int
	}{
		{"disabled when unset", "", "Bearer anything", http.StatusNotFound},
		{"missing header", "s3cret", "", http.StatusUnauthorized},
		{"wrong scheme", "s3cret", "Basic s3cret", http.StatusUnauthorized},
		{"wrong token", "s3cret", "Bearer nope", http.StatusUnauthorized},
		{"empty bearer", "s3cret", "Bearer ", http.StatusUnauthorized},
		{"correct token", "s3cret", "Bearer s3cret", http.StatusOK},
		{"scheme is case-insensitive", "s3cret", "bearer s3cret", http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.Use(VaultSyncAuth(tt.token))
			r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

			req := httptest.NewRequest(http.MethodGet, "/x", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != tt.want {
				t.Fatalf("status = %d, want %d", w.Code, tt.want)
			}
		})
	}
}
