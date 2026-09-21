package databases

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"life-base-api/app/internal/auth"
)

type postgrestError struct {
	Message string `json:"message"`
	Code    string `json:"code"`
}

func (e *postgrestError) Error() string {
	return fmt.Sprintf("postgrest %s: %s", e.Code, e.Message)
}

func addHeaders(req *http.Request, apiKey, bearer, prefer string) {
	req.Header.Set("apikey", apiKey)
	req.Header.Set("Authorization", "Bearer "+bearer)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if prefer != "" {
		req.Header.Set("Prefer", prefer)
	}
}

// resolveKeys returns the apiKey and bearer token to use for a request.
// If a user JWT is in context (web request), use anon key + user JWT so RLS applies.
// Otherwise fall back to service key (CLI import, bypasses RLS).
func resolveKeys(ctx context.Context, c *SupabaseClient) (apiKey, bearer string) {
	if userToken := auth.UserTokenFromContext(ctx); userToken != "" {
		return c.AnonKey, userToken
	}
	return c.APIKey, c.APIKey
}

// Get sends an authenticated GET request to path with query params and decodes the JSON response into T.
func Get[T any](ctx context.Context, c *SupabaseClient, path string, params url.Values) (T, error) {
	var zero T

	rawURL := c.BaseURL + path
	if len(params) > 0 {
		rawURL += "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return zero, fmt.Errorf("build request: %w", err)
	}
	apiKey, bearer := resolveKeys(ctx, c)
	addHeaders(req, apiKey, bearer, "")

	return decode[T](c.HTTPClient.Do(req))
}

// Post sends an authenticated POST to path with body marshaled as JSON and decodes the response into T.
func Post[T any](ctx context.Context, c *SupabaseClient, path string, body any, prefer string) (T, error) {
	var zero T

	data, err := json.Marshal(body)
	if err != nil {
		return zero, fmt.Errorf("marshal body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+path, bytes.NewReader(data))
	if err != nil {
		return zero, fmt.Errorf("build request: %w", err)
	}
	apiKey, bearer := resolveKeys(ctx, c)
	addHeaders(req, apiKey, bearer, prefer)

	return decode[T](c.HTTPClient.Do(req))
}

// Patch sends an authenticated PATCH to path with body marshaled as JSON and decodes the response into T.
func Patch[T any](ctx context.Context, c *SupabaseClient, path string, body any, prefer string) (T, error) {
	var zero T

	data, err := json.Marshal(body)
	if err != nil {
		return zero, fmt.Errorf("marshal body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, c.BaseURL+path, bytes.NewReader(data))
	if err != nil {
		return zero, fmt.Errorf("build request: %w", err)
	}
	apiKey, bearer := resolveKeys(ctx, c)
	addHeaders(req, apiKey, bearer, prefer)

	return decode[T](c.HTTPClient.Do(req))
}

// Delete sends an authenticated DELETE to path and discards the response body.
func Delete(ctx context.Context, c *SupabaseClient, path string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.BaseURL+path, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	apiKey, bearer := resolveKeys(ctx, c)
	addHeaders(req, apiKey, bearer, "")
	_, err = decode[struct{}](c.HTTPClient.Do(req))
	return err
}

// decode reads the HTTP response, checks for PostgREST errors, and unmarshals the body into T.
// It is shared by Get and Post to avoid duplicating response-handling logic.
func decode[T any](resp *http.Response, err error) (T, error) {
	var zero T
	if err != nil {
		return zero, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return zero, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= http.StatusBadRequest {
		var pgErr postgrestError
		if jsonErr := json.Unmarshal(body, &pgErr); jsonErr == nil && pgErr.Message != "" {
			return zero, &pgErr
		}
		return zero, fmt.Errorf("http %d: %s", resp.StatusCode, body)
	}

	if len(body) == 0 {
		return zero, nil
	}

	var result T
	if err := json.Unmarshal(body, &result); err != nil {
		return zero, fmt.Errorf("decode response: %w", err)
	}
	return result, nil
}
