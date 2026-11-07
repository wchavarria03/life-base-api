package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"life-base-api/app/internal/models"
	"life-base-api/app/internal/services"
)

// vaultSyncBodyLimit caps internal sync request bodies (below the global
// 10 MB limit).
const vaultSyncBodyLimit = 8 << 20

// VaultManager is the subset of VaultService the handler needs.
type VaultManager interface {
	Upsert(ctx context.Context, req models.VaultUpsertRequest) ([]models.VaultUpsertResult, error)
	Pending(ctx context.Context) (*models.VaultPendingResponse, error)
	Ack(ctx context.Context, req models.VaultAckRequest) error
	SyncDone(ctx context.Context, commit string) error
	RequestSync(ctx context.Context) (*models.VaultSyncState, error)
	SyncStatus(ctx context.Context) (*models.VaultSyncState, error)
	SetSensitive(ctx context.Context, id string, value bool) (*models.Note, error)
	Backlinks(ctx context.Context, id string) ([]models.Backlink, error)
	Search(ctx context.Context, q string, limit int) ([]*models.NoteSearchHit, error)
	LookupByOrgID(ctx context.Context, orgID string) (*models.NoteRef, error)
	ResolveConflict(ctx context.Context, id, keep string) (*models.Note, error)
}

// VaultHandler serves the vault sync endpoints: the service-token protected
// /v1/internal/notes/* routes for the homelab worker and the user-facing
// /v1/notes/* additions (search, backlinks, sensitive flag, sync button).
type VaultHandler struct {
	svc VaultManager
}

// NewVaultHandler constructs a VaultHandler.
func NewVaultHandler(svc VaultManager) *VaultHandler {
	return &VaultHandler{svc: svc}
}

// vaultError maps service errors to HTTP responses.
func vaultError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrVaultDisabled):
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "vault sync is not configured"})
	case errors.Is(err, services.ErrNoteNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "note not found"})
	case errors.Is(err, services.ErrSensitiveDeclared):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error(), "code": "sensitive_declared_in_file"})
	default:
		internalError(c, err)
	}
}

// limitBody applies the sync body cap before binding.
func limitBody(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, vaultSyncBodyLimit)
}

// Upsert handles POST /v1/internal/notes/upsert.
func (h *VaultHandler) Upsert(c *gin.Context) {
	limitBody(c)
	req, ok := bindJSON[models.VaultUpsertRequest](c)
	if !ok {
		return
	}
	results, err := h.svc.Upsert(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, services.ErrVaultDisabled) {
			vaultError(c, err)
			return
		}
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"results": results})
}

// Pending handles GET /v1/internal/notes/pending.
func (h *VaultHandler) Pending(c *gin.Context) {
	resp, err := h.svc.Pending(c.Request.Context())
	if err != nil {
		vaultError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// Ack handles POST /v1/internal/notes/ack.
func (h *VaultHandler) Ack(c *gin.Context) {
	limitBody(c)
	req, ok := bindJSON[models.VaultAckRequest](c)
	if !ok {
		return
	}
	if err := h.svc.Ack(c.Request.Context(), req); err != nil {
		if errors.Is(err, services.ErrVaultDisabled) || errors.Is(err, services.ErrNoteNotFound) {
			vaultError(c, err)
			return
		}
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// SyncDone handles POST /v1/internal/notes/sync-done.
func (h *VaultHandler) SyncDone(c *gin.Context) {
	limitBody(c)
	req, ok := bindJSON[models.VaultSyncDoneRequest](c)
	if !ok {
		return
	}
	if err := h.svc.SyncDone(c.Request.Context(), req.Commit); err != nil {
		if errors.Is(err, services.ErrVaultDisabled) {
			vaultError(c, err)
			return
		}
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// RequestSync handles POST /v1/notes/sync-request.
func (h *VaultHandler) RequestSync(c *gin.Context) {
	state, err := h.svc.RequestSync(c.Request.Context())
	if err != nil {
		vaultError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, state)
}

// SyncStatus handles GET /v1/notes/sync-status.
func (h *VaultHandler) SyncStatus(c *gin.Context) {
	state, err := h.svc.SyncStatus(c.Request.Context())
	if err != nil {
		vaultError(c, err)
		return
	}
	c.JSON(http.StatusOK, state)
}

// SetSensitive handles PATCH /v1/notes/:id/sensitive.
func (h *VaultHandler) SetSensitive(c *gin.Context) {
	req, ok := bindJSON[struct {
		Sensitive *bool `json:"sensitive"`
	}](c)
	if !ok {
		return
	}
	if req.Sensitive == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sensitive (boolean) is required"})
		return
	}
	note, err := h.svc.SetSensitive(c.Request.Context(), c.Param("id"), *req.Sensitive)
	if err != nil {
		vaultError(c, err)
		return
	}
	c.JSON(http.StatusOK, note)
}

// Backlinks handles GET /v1/notes/:id/backlinks.
func (h *VaultHandler) Backlinks(c *gin.Context) {
	links, err := h.svc.Backlinks(c.Request.Context(), c.Param("id"))
	if err != nil {
		vaultError(c, err)
		return
	}
	c.JSON(http.StatusOK, links)
}

// Search handles GET /v1/notes/search?q=&limit=.
func (h *VaultHandler) Search(c *gin.Context) {
	limit, _ := strconv.Atoi(c.Query("limit"))
	hits, err := h.svc.Search(c.Request.Context(), c.Query("q"), limit)
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, hits)
}

// Lookup handles GET /v1/notes/lookup?org_id= (resolves an id: link).
func (h *VaultHandler) Lookup(c *gin.Context) {
	ref, err := h.svc.LookupByOrgID(c.Request.Context(), c.Query("org_id"))
	if err != nil {
		vaultError(c, err)
		return
	}
	c.JSON(http.StatusOK, ref)
}

// ResolveConflict handles POST /v1/notes/:id/resolve-conflict {"keep":"mine"|"vault"}.
func (h *VaultHandler) ResolveConflict(c *gin.Context) {
	req, ok := bindJSON[struct {
		Keep string `json:"keep"`
	}](c)
	if !ok {
		return
	}
	note, err := h.svc.ResolveConflict(c.Request.Context(), c.Param("id"), req.Keep)
	if err != nil {
		if errors.Is(err, services.ErrNoteNotFound) {
			vaultError(c, err)
			return
		}
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, note)
}
