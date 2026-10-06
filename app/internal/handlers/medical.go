package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"life-base-api/app/internal/models"
)

const maxMedicalFileBytes = 25 << 20 // 25 MB

// NewMedicalProfileHandler constructs a MedicalProfileHandler.
func NewMedicalProfileHandler(svc MedicalProfileManager) *MedicalProfileHandler {
	return &MedicalProfileHandler{svc: svc}
}

// List handles GET /v1/medical-profiles.
func (h *MedicalProfileHandler) List(c *gin.Context) {
	profiles, err := h.svc.List(c.Request.Context())
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, profiles)
}

// Create handles POST /v1/medical-profiles.
func (h *MedicalProfileHandler) Create(c *gin.Context) {
	input, ok := bindJSON[models.MedicalProfileInput](c)
	if !ok {
		return
	}
	profile, err := h.svc.Create(c.Request.Context(), input)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, profile)
}

// Update handles PATCH /v1/medical-profiles/:id.
func (h *MedicalProfileHandler) Update(c *gin.Context) {
	fields, ok := bindJSON[map[string]any](c)
	if !ok {
		return
	}
	profile, err := h.svc.Update(c.Request.Context(), c.Param("id"), fields)
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, profile)
}

// Delete handles DELETE /v1/medical-profiles/:id.
func (h *MedicalProfileHandler) Delete(c *gin.Context) {
	if err := h.svc.Delete(c.Request.Context(), c.Param("id")); err != nil {
		internalError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ListAccess handles GET /v1/medical-profiles/:id/access.
func (h *MedicalProfileHandler) ListAccess(c *gin.Context) {
	grants, err := h.svc.ListAccess(c.Request.Context(), c.Param("id"))
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, grants)
}

type grantAccessRequest struct {
	Email string                   `json:"email"`
	Role  models.MedicalAccessRole `json:"role"`
}

// GrantAccess handles POST /v1/medical-profiles/:id/access.
func (h *MedicalProfileHandler) GrantAccess(c *gin.Context) {
	req, ok := bindJSON[grantAccessRequest](c)
	if !ok {
		return
	}
	grant, err := h.svc.GrantAccess(c.Request.Context(), c.Param("id"), req.Email, req.Role)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, grant)
}

// RevokeAccess handles DELETE /v1/medical-profiles/:id/access/:grantId.
func (h *MedicalProfileHandler) RevokeAccess(c *gin.Context) {
	if err := h.svc.RevokeAccess(c.Request.Context(), c.Param("grantId")); err != nil {
		internalError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// Timeline handles GET /v1/medical-profiles/:id/timeline?attribute=.
func (h *MedicalProfileHandler) Timeline(c *gin.Context) {
	attribute := c.Query("attribute")
	if attribute == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "attribute query param is required"})
		return
	}
	points, err := h.svc.Timeline(c.Request.Context(), c.Param("id"), attribute)
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, points)
}

// NewMedicalRecordHandler constructs a MedicalRecordHandler.
func NewMedicalRecordHandler(svc MedicalRecordManager) *MedicalRecordHandler {
	return &MedicalRecordHandler{svc: svc}
}

// List handles GET /v1/medical-records?profile_id=.
func (h *MedicalRecordHandler) List(c *gin.Context) {
	profileID := c.Query("profile_id")
	if profileID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "profile_id query param is required"})
		return
	}
	records, err := h.svc.ListByProfile(c.Request.Context(), profileID)
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, records)
}

// Create handles POST /v1/medical-records.
func (h *MedicalRecordHandler) Create(c *gin.Context) {
	input, ok := bindJSON[models.MedicalRecordInput](c)
	if !ok {
		return
	}
	record, err := h.svc.Create(c.Request.Context(), input)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, record)
}

// Update handles PATCH /v1/medical-records/:id.
func (h *MedicalRecordHandler) Update(c *gin.Context) {
	fields, ok := bindJSON[map[string]any](c)
	if !ok {
		return
	}
	record, err := h.svc.Update(c.Request.Context(), c.Param("id"), fields)
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, record)
}

// Delete handles DELETE /v1/medical-records/:id.
func (h *MedicalRecordHandler) Delete(c *gin.Context) {
	if err := h.svc.Delete(c.Request.Context(), c.Param("id")); err != nil {
		internalError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ListFiles handles GET /v1/medical-records/:id/files.
func (h *MedicalRecordHandler) ListFiles(c *gin.Context) {
	files, err := h.svc.ListFiles(c.Request.Context(), c.Param("id"))
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, files)
}

// UploadFile handles POST /v1/medical-records/:id/files — multipart upload
// with a "file" field.
func (h *MedicalRecordHandler) UploadFile(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxMedicalFileBytes)

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file field is required"})
		return
	}
	defer func() { _ = file.Close() }()

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	row, err := h.svc.UploadFile(c.Request.Context(), c.Param("id"), file, header.Filename, contentType)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, row)
}

// DownloadFile handles GET /v1/medical-records/:id/files/:fileId/download.
func (h *MedicalRecordHandler) DownloadFile(c *gin.Context) {
	f, data, err := h.svc.DownloadFile(c.Request.Context(), c.Param("fileId"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}
	contentType := "application/octet-stream"
	if f.ContentType != nil {
		contentType = *f.ContentType
	}
	c.Header("Content-Disposition", fmt.Sprintf(`inline; filename=%q`, f.FileName))
	c.Data(http.StatusOK, contentType, data)
}

// DeleteFile handles DELETE /v1/medical-records/:id/files/:fileId.
func (h *MedicalRecordHandler) DeleteFile(c *gin.Context) {
	if err := h.svc.DeleteFile(c.Request.Context(), c.Param("fileId")); err != nil {
		internalError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
