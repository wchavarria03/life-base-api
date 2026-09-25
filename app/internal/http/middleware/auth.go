package middleware

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"life-base-api/app/internal/auth"
)

// jwksRefreshInterval bounds how long a rotated Supabase signing key can be
// unrecognized for. Verification falls back to an on-demand refetch on
// failure regardless, so this is a ceiling, not the only recovery path.
const jwksRefreshInterval = 10 * time.Minute

// jwksMinRefetchInterval throttles the on-demand refetch triggered by a
// verification failure, so a flood of invalid tokens can't be used to spam
// the JWKS endpoint.
const jwksMinRefetchInterval = 10 * time.Second

type supabaseClaims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

type jwks struct {
	Keys []jwk `json:"keys"`
}

type jwk struct {
	Alg string `json:"alg"`
	Crv string `json:"crv"`
	X   string `json:"x"`
	Y   string `json:"y"`
}

func fetchECPublicKey(jwksURL string) (*ecdsa.PublicKey, error) {
	resp, err := http.Get(jwksURL) //nolint:noctx
	if err != nil {
		return nil, fmt.Errorf("fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	var keys jwks
	if err := json.NewDecoder(resp.Body).Decode(&keys); err != nil {
		return nil, fmt.Errorf("decode JWKS: %w", err)
	}

	for _, k := range keys.Keys {
		if k.Alg == "ES256" || k.Crv == "P-256" {
			xBytes, err := base64.RawURLEncoding.DecodeString(k.X)
			if err != nil {
				return nil, fmt.Errorf("decode x: %w", err)
			}
			yBytes, err := base64.RawURLEncoding.DecodeString(k.Y)
			if err != nil {
				return nil, fmt.Errorf("decode y: %w", err)
			}
			return &ecdsa.PublicKey{
				Curve: elliptic.P256(),
				X:     new(big.Int).SetBytes(xBytes),
				Y:     new(big.Int).SetBytes(yBytes),
			}, nil
		}
	}

	return nil, fmt.Errorf("no ES256 key found in JWKS")
}

// jwksKeyStore holds the current JWKS-fetched ES256 key and refreshes it
// periodically plus on-demand when a verification fails, so a rotated
// Supabase signing key recovers without a process restart.
type jwksKeyStore struct {
	jwksURL string

	mu          sync.RWMutex
	key         *ecdsa.PublicKey
	lastRefetch time.Time
}

func newJWKSKeyStore(jwksURL string) *jwksKeyStore {
	s := &jwksKeyStore{jwksURL: jwksURL}
	if jwksURL == "" {
		return s
	}

	s.refresh()

	go func() {
		ticker := time.NewTicker(jwksRefreshInterval)
		defer ticker.Stop()
		for range ticker.C {
			s.refresh()
		}
	}()

	return s
}

func (s *jwksKeyStore) refresh() {
	key, err := fetchECPublicKey(s.jwksURL)
	s.mu.Lock()
	s.lastRefetch = time.Now()
	if err == nil {
		s.key = key
	}
	s.mu.Unlock()
}

// refetchOnFailure is the debounced counterpart to refresh, called from the
// request path after a verification failure. It skips the fetch if one
// already happened recently, so a flood of invalid tokens can't be used to
// spam the JWKS endpoint.
func (s *jwksKeyStore) refetchOnFailure() {
	s.mu.RLock()
	tooSoon := time.Since(s.lastRefetch) < jwksMinRefetchInterval
	s.mu.RUnlock()
	if tooSoon {
		return
	}
	s.refresh()
}

func (s *jwksKeyStore) get() *ecdsa.PublicKey {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.key
}

// Auth validates Supabase JWTs using ES256 (asymmetric, verified via JWKS).
// issuer, when non-empty, is required to match the token's "iss" claim
// (Supabase's GoTrue issuer, typically "<SUPABASE_URL>/auth/v1"); the
// token's audience must be "authenticated", matching every Supabase-issued
// user access token.
func Auth(jwksURL, issuer string) gin.HandlerFunc {
	store := newJWKSKeyStore(jwksURL)

	parserOpts := []jwt.ParserOption{jwt.WithAudience("authenticated")}
	if issuer != "" {
		parserOpts = append(parserOpts, jwt.WithIssuer(issuer))
	}

	verify := func(tokenStr string) (*supabaseClaims, error) {
		claims := &supabaseClaims{}
		_, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
			switch t.Method.(type) {
			case *jwt.SigningMethodECDSA:
				if key := store.get(); key != nil {
					return key, nil
				}
				return nil, fmt.Errorf("no EC key available for ES256")
			default:
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
		}, parserOpts...)
		return claims, err
	}

	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}

		tokenStr := strings.TrimPrefix(header, "Bearer ")

		claims, err := verify(tokenStr)
		if err != nil {
			// The stored key may be stale (e.g. Supabase rotated it since our
			// last periodic refresh) — refetch once and retry before failing.
			store.refetchOnFailure()
			claims, err = verify(tokenStr)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
				return
			}
		}

		userID, err := claims.GetSubject()
		if err != nil || userID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing subject claim"})
			return
		}

		ctx := auth.WithUser(c.Request.Context(), tokenStr, userID)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
