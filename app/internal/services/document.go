package services

import (
	"context"
	"fmt"
	"io"
	"time"

	supabaserepo "life-base-api/app/internal/repositories/supabase"

	"life-base-api/app/internal/auth"
	"life-base-api/app/internal/models"
)

const documentsBucket = "documents"

// DocumentService manages the private document vault — upload, list,
// download, delete.
type DocumentService struct {
	repo    *supabaserepo.DocumentRepository
	storage *supabaserepo.StorageRepository
}

// NewDocumentService constructs a DocumentService.
func NewDocumentService(repo *supabaserepo.DocumentRepository, storage *supabaserepo.StorageRepository) *DocumentService {
	return &DocumentService{repo: repo, storage: storage}
}

// List returns every document the caller owns.
func (s *DocumentService) List(ctx context.Context) ([]*models.Document, error) {
	return s.repo.List(ctx)
}

// Upload stores file in the private documents bucket and records its
// metadata.
func (s *DocumentService) Upload(ctx context.Context, file io.Reader, title, fileName, contentType string, category *string) (*models.Document, error) {
	userID := auth.UserIDFromContext(ctx)
	if userID == "" {
		return nil, fmt.Errorf("no authenticated user")
	}
	if title == "" {
		return nil, fmt.Errorf("title is required")
	}

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("read uploaded file: %w", err)
	}

	path := fmt.Sprintf("%s/%d-%s", userID, time.Now().UnixNano(), fileName)
	if _, err := s.storage.UploadObject(ctx, documentsBucket, path, data, contentType); err != nil {
		return nil, fmt.Errorf("upload file: %w", err)
	}

	return s.repo.Create(ctx, models.DocumentInput{
		UserID:      userID,
		Title:       title,
		Category:    category,
		FileName:    fileName,
		ContentType: contentType,
		FileSize:    int64(len(data)),
		StoragePath: path,
	})
}

// Download returns a document's raw bytes plus its stored metadata —
// FindByID is RLS-scoped, so a document another user owns resolves to nil
// here before storage is ever touched.
func (s *DocumentService) Download(ctx context.Context, id string) (*models.Document, []byte, error) {
	doc, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, nil, fmt.Errorf("find document: %w", err)
	}
	if doc == nil {
		return nil, nil, fmt.Errorf("document not found")
	}
	data, err := s.storage.DownloadObject(ctx, documentsBucket, doc.StoragePath)
	if err != nil {
		return nil, nil, fmt.Errorf("download file: %w", err)
	}
	return doc, data, nil
}

// Delete removes a document's row and its stored file.
func (s *DocumentService) Delete(ctx context.Context, id string) error {
	doc, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("find document: %w", err)
	}
	if doc == nil {
		return nil
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	_ = s.storage.DeleteObject(ctx, documentsBucket, doc.StoragePath)
	return nil
}
