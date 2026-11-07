package services

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"life-base-api/app/internal/auth"
	"life-base-api/app/internal/models"
)

const (
	// signedURLTTL is how long a signed URL for an ordinary document lives.
	signedURLTTL = time.Hour
	// sensitiveSignedURLTTL is the much shorter lifetime for sensitive ones.
	sensitiveSignedURLTTL = 5 * time.Minute
)

// Errors returned by the document link and signed-URL operations.
var (
	ErrInvalidEntityType   = errors.New("invalid entity_type")
	ErrInvalidEntityID     = errors.New("invalid entity_id")
	ErrDocumentNotFound    = errors.New("document not found")
	ErrSigningDisabled     = errors.New("signed urls are not configured")
	ErrInvalidDocSignature = errors.New("invalid or expired signature")
)

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// validDocumentEntity reports whether t is an attachable entity type.
func validDocumentEntity(t string) bool {
	switch t {
	case models.DocumentEntityNote, models.DocumentEntityTaskLog, models.DocumentEntityBike,
		models.DocumentEntityComponent, models.DocumentEntityMedicalRecord:
		return true
	}
	return false
}

// AddLink attaches a document to an entity. The document lookup is
// RLS-scoped, so another user's document resolves as not found.
func (s *DocumentService) AddLink(ctx context.Context, documentID, entityType, entityID string) (*models.DocumentLink, error) {
	userID := auth.UserIDFromContext(ctx)
	if userID == "" {
		return nil, fmt.Errorf("no authenticated user")
	}
	if !validDocumentEntity(entityType) {
		return nil, ErrInvalidEntityType
	}
	if !uuidPattern.MatchString(entityID) {
		return nil, ErrInvalidEntityID
	}
	doc, err := s.repo.FindByID(ctx, documentID)
	if err != nil {
		return nil, fmt.Errorf("find document: %w", err)
	}
	if doc == nil {
		return nil, ErrDocumentNotFound
	}
	return s.links.Create(ctx, models.DocumentLinkInput{
		UserID:     userID,
		DocumentID: documentID,
		EntityType: entityType,
		EntityID:   entityID,
	})
}

// ListLinks returns every link of one document.
func (s *DocumentService) ListLinks(ctx context.Context, documentID string) ([]*models.DocumentLink, error) {
	return s.links.ListByDocument(ctx, documentID)
}

// RemoveLink detaches a document from an entity (the document is kept).
func (s *DocumentService) RemoveLink(ctx context.Context, documentID, linkID string) error {
	return s.links.Delete(ctx, documentID, linkID)
}

// ListByEntity returns the documents attached to one entity, newest first.
func (s *DocumentService) ListByEntity(ctx context.Context, entityType, entityID string) ([]*models.Document, error) {
	if !validDocumentEntity(entityType) {
		return nil, ErrInvalidEntityType
	}
	if !uuidPattern.MatchString(entityID) {
		return nil, ErrInvalidEntityID
	}
	links, err := s.links.ListByEntity(ctx, entityType, entityID)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(links))
	for _, l := range links {
		ids = append(ids, l.DocumentID)
	}
	return s.repo.ListByIDs(ctx, ids)
}

// SignedURL returns a short-lived, JWT-less URL path for a document the
// caller owns. Sensitive documents get a much shorter lifetime.
func (s *DocumentService) SignedURL(ctx context.Context, id string) (*models.SignedDocumentURL, error) {
	if len(s.signingSecret) == 0 {
		return nil, ErrSigningDisabled
	}
	doc, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("find document: %w", err)
	}
	if doc == nil {
		return nil, ErrDocumentNotFound
	}
	return s.signDocument(doc, time.Now()), nil
}

// signDocument builds the signed URL for doc as of now.
func (s *DocumentService) signDocument(doc *models.Document, now time.Time) *models.SignedDocumentURL {
	ttl := signedURLTTL
	if doc.Sensitive {
		ttl = sensitiveSignedURLTTL
	}
	expires := now.Add(ttl)
	exp := strconv.FormatInt(expires.Unix(), 10)
	return &models.SignedDocumentURL{
		URL:       "/public/documents/" + doc.ID + "?exp=" + exp + "&sig=" + s.docSignature(doc.ID, exp),
		ExpiresAt: expires,
	}
}

// docSignature is the hex HMAC-SHA256 of "id|exp".
func (s *DocumentService) docSignature(id, exp string) string {
	mac := hmac.New(sha256.New, s.signingSecret)
	mac.Write([]byte(id + "|" + exp))
	return hex.EncodeToString(mac.Sum(nil))
}

// verifySignature checks sig and expiry for a signed document URL.
func (s *DocumentService) verifySignature(id, exp, sig string, now time.Time) error {
	if len(s.signingSecret) == 0 {
		return ErrSigningDisabled
	}
	expUnix, err := strconv.ParseInt(exp, 10, 64)
	if err != nil || now.Unix() > expUnix {
		return ErrInvalidDocSignature
	}
	if !hmac.Equal([]byte(sig), []byte(s.docSignature(id, exp))) {
		return ErrInvalidDocSignature
	}
	return nil
}

// DownloadSigned serves a document to an unauthenticated caller holding a
// valid signature. With no user token in ctx the repository falls back to
// the service key, so ownership is established by the signature alone.
func (s *DocumentService) DownloadSigned(ctx context.Context, id, exp, sig string) (*models.Document, []byte, error) {
	if err := s.verifySignature(id, exp, sig, time.Now()); err != nil {
		return nil, nil, err
	}
	return s.Download(ctx, id)
}

// shareableAttachments returns the non-sensitive documents linked to a note,
// each with a signed URL. Sensitive documents are never included.
func (s *DocumentService) shareableAttachments(ctx context.Context, noteID string) []models.SharedAttachmentView {
	if len(s.signingSecret) == 0 {
		return nil
	}
	docs, err := s.ListByEntity(ctx, models.DocumentEntityNote, noteID)
	if err != nil {
		return nil // best-effort — a note without visible attachments is still a valid share
	}
	out := make([]models.SharedAttachmentView, 0, len(docs))
	for _, d := range docs {
		if d.Sensitive {
			continue
		}
		out = append(out, models.SharedAttachmentView{
			Title:       d.Title,
			ContentType: d.ContentType,
			URL:         s.signDocument(d, time.Now()).URL,
		})
	}
	return out
}
