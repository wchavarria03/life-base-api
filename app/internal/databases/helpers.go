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

func addHeaders(req *http.Request, apiKey, bearer, prefer string, schema ...string) {
	req.Header.Set("apikey", apiKey)
	req.Header.Set("Authorization", "Bearer "+bearer)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if prefer != "" {
		req.Header.Set("Prefer", prefer)
	}
	// PostgREST only serves schemas explicitly exposed in the project's API
	// settings; Accept-Profile/Content-Profile pick which one a request
	// targets. Omitted (the common case) means the default "public" schema.
	if len(schema) > 0 && schema[0] != "" {
		req.Header.Set("Accept-Profile", schema[0])
		req.Header.Set("Content-Profile", schema[0])
	}
}

// EqID builds the common `?id=eq.<id>` filter as properly-encoded url.Values,
// for Patch/Delete callers that only filter by a single id.
func EqID(id string) url.Values {
	return url.Values{"id": []string{"eq." + id}}
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

// Get sends an authenticated GET request to path with query params and decodes the JSON
// response into T. An optional schema targets a non-public PostgREST-exposed schema.
func Get[T any](ctx context.Context, c *SupabaseClient, path string, params url.Values, schema ...string) (T, error) {
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
	addHeaders(req, apiKey, bearer, "", schema...)

	return decode[T](c.HTTPClient.Do(req))
}

// Post sends an authenticated POST to path with body marshaled as JSON and decodes the
// response into T. An optional schema targets a non-public PostgREST-exposed schema.
func Post[T any](ctx context.Context, c *SupabaseClient, path string, body any, prefer string, schema ...string) (T, error) {
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
	addHeaders(req, apiKey, bearer, prefer, schema...)

	return decode[T](c.HTTPClient.Do(req))
}

// Patch sends an authenticated PATCH to path with query params and a body marshaled as
// JSON, and decodes the response into T. Params are properly URL-encoded — callers must
// not hand-concatenate filter values (e.g. an id) into path, since PostgREST filter
// syntax in an unescaped value can inject extra query parameters. An optional schema
// targets a non-public PostgREST-exposed schema.
func Patch[T any](ctx context.Context, c *SupabaseClient, path string, params url.Values, body any, prefer string, schema ...string) (T, error) {
	var zero T

	data, err := json.Marshal(body)
	if err != nil {
		return zero, fmt.Errorf("marshal body: %w", err)
	}

	rawURL := c.BaseURL + path
	if len(params) > 0 {
		rawURL += "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, rawURL, bytes.NewReader(data))
	if err != nil {
		return zero, fmt.Errorf("build request: %w", err)
	}
	apiKey, bearer := resolveKeys(ctx, c)
	addHeaders(req, apiKey, bearer, prefer, schema...)

	return decode[T](c.HTTPClient.Do(req))
}

// Delete sends an authenticated DELETE to path with query params and discards the
// response body. See Patch's doc comment for why params must be passed as url.Values
// rather than concatenated into path. An optional schema targets a non-public
// PostgREST-exposed schema.
func Delete(ctx context.Context, c *SupabaseClient, path string, params url.Values, schema ...string) error {
	rawURL := c.BaseURL + path
	if len(params) > 0 {
		rawURL += "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, rawURL, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	apiKey, bearer := resolveKeys(ctx, c)
	addHeaders(req, apiKey, bearer, "", schema...)
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
