package supabase

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	"life-base-api/app/internal/databases"
)

// StorageRepository uploads files to Supabase Storage. It always uses the
// service-role key — the Go backend is the trusted intermediary here (the
// browser never talks to Supabase Storage directly), same trust boundary
// already used for CLI imports.
type StorageRepository struct {
	client *databases.SupabaseClient
}

// NewStorageRepository constructs a StorageRepository.
func NewStorageRepository(client *databases.SupabaseClient) *StorageRepository {
	return &StorageRepository{client: client}
}

// UploadObject uploads data to bucket/path, upserting if it already exists,
// and returns its public URL. The bucket must be public for the returned
// URL to be usable without an auth header (e.g. by Instagram's Graph API).
func (r *StorageRepository) UploadObject(ctx context.Context, bucket, path string, data []byte, contentType string) (string, error) {
	url := fmt.Sprintf("%s/storage/v1/object/%s/%s", r.client.BaseURL, bucket, path)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("build storage request: %w", err)
	}
	req.Header.Set("apikey", r.client.APIKey)
	req.Header.Set("Authorization", "Bearer "+r.client.APIKey)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("x-upsert", "true")

	resp, err := r.client.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("upload to storage: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= http.StatusBadRequest {
		return "", fmt.Errorf("storage upload: http %d", resp.StatusCode)
	}

	return fmt.Sprintf("%s/storage/v1/object/public/%s/%s", r.client.BaseURL, bucket, path), nil
}

// DownloadObject fetches bucket/path's raw bytes — used for private
// buckets (e.g. "documents") that have no usable public URL, where the Go
// backend is the only thing allowed to read the object directly.
func (r *StorageRepository) DownloadObject(ctx context.Context, bucket, path string) ([]byte, error) {
	url := fmt.Sprintf("%s/storage/v1/object/%s/%s", r.client.BaseURL, bucket, path)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build storage download request: %w", err)
	}
	req.Header.Set("apikey", r.client.APIKey)
	req.Header.Set("Authorization", "Bearer "+r.client.APIKey)

	resp, err := r.client.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download from storage: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("storage download: http %d", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read storage response: %w", err)
	}
	return data, nil
}

// DeleteObject removes bucket/path. Best-effort by convention at call
// sites — a stray orphaned file is harmless, so callers generally don't
// fail an otherwise-successful operation (like deleting a row) over this.
func (r *StorageRepository) DeleteObject(ctx context.Context, bucket, path string) error {
	url := fmt.Sprintf("%s/storage/v1/object/%s/%s", r.client.BaseURL, bucket, path)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("build storage delete request: %w", err)
	}
	req.Header.Set("apikey", r.client.APIKey)
	req.Header.Set("Authorization", "Bearer "+r.client.APIKey)

	resp, err := r.client.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("delete from storage: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("storage delete: http %d", resp.StatusCode)
	}
	return nil
}
