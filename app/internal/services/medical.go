package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"

	"life-base-api/app/internal/auth"
	"life-base-api/app/internal/models"
	supabaserepo "life-base-api/app/internal/repositories/supabase"
)

const medicalFilesBucket = "medical-files"

// MedicalProfileService manages medical profiles and their access grants.
type MedicalProfileService struct {
	profiles *supabaserepo.MedicalProfileRepository
	access   *supabaserepo.MedicalAccessRepository
	records  *supabaserepo.MedicalRecordRepository
}

// NewMedicalProfileService constructs a MedicalProfileService.
func NewMedicalProfileService(profiles *supabaserepo.MedicalProfileRepository, access *supabaserepo.MedicalAccessRepository, records *supabaserepo.MedicalRecordRepository) *MedicalProfileService {
	return &MedicalProfileService{profiles: profiles, access: access, records: records}
}

// List returns every profile visible to the caller — their own plus any
// granted to their login email.
func (s *MedicalProfileService) List(ctx context.Context) ([]*models.MedicalProfile, error) {
	return s.profiles.List(ctx)
}

// FindByID looks up a single profile, RLS-scoped.
func (s *MedicalProfileService) FindByID(ctx context.Context, id string) (*models.MedicalProfile, error) {
	return s.profiles.FindByID(ctx, id)
}

// Create adds a new profile owned by the caller.
func (s *MedicalProfileService) Create(ctx context.Context, input models.MedicalProfileInput) (*models.MedicalProfile, error) {
	userID := auth.UserIDFromContext(ctx)
	if userID == "" {
		return nil, fmt.Errorf("no authenticated user")
	}
	if input.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	input.OwnerUserID = userID
	profile, err := s.profiles.Create(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("create medical profile: %w", err)
	}
	return profile, nil
}

// Update patches a profile (owner or editor, enforced by RLS).
func (s *MedicalProfileService) Update(ctx context.Context, id string, fields map[string]any) (*models.MedicalProfile, error) {
	profile, err := s.profiles.Update(ctx, id, fields)
	if err != nil {
		return nil, fmt.Errorf("update medical profile: %w", err)
	}
	return profile, nil
}

// Delete removes a profile (owner only, enforced by RLS).
func (s *MedicalProfileService) Delete(ctx context.Context, id string) error {
	return s.profiles.Delete(ctx, id)
}

// ListAccess returns a profile's granted accounts (owner only, via RLS).
func (s *MedicalProfileService) ListAccess(ctx context.Context, profileID string) ([]*models.MedicalProfileAccess, error) {
	return s.access.ListByProfile(ctx, profileID)
}

// GrantAccess shares a profile with another gmail account by email —
// enforced off that account's JWT email claim at login, no invite flow.
func (s *MedicalProfileService) GrantAccess(ctx context.Context, profileID, email string, role models.MedicalAccessRole) (*models.MedicalProfileAccess, error) {
	if email == "" {
		return nil, fmt.Errorf("email is required")
	}
	if role != models.MedicalAccessViewer && role != models.MedicalAccessEditor {
		return nil, fmt.Errorf("invalid role: %s", role)
	}
	grant, err := s.access.Grant(ctx, &models.MedicalProfileAccess{ProfileID: profileID, Email: email, Role: role})
	if err != nil {
		return nil, fmt.Errorf("grant access: %w", err)
	}
	return grant, nil
}

// RevokeAccess removes a grant (owner only, via RLS).
func (s *MedicalProfileService) RevokeAccess(ctx context.Context, grantID string) error {
	return s.access.Revoke(ctx, grantID)
}

// Timeline extracts one attribute key's value from every record on a
// profile that has it set, in date order — the data behind the
// year-to-year comparison chart.
func (s *MedicalProfileService) Timeline(ctx context.Context, profileID, attributeKey string) ([]models.MedicalTimelinePoint, error) {
	records, err := s.records.ListByProfile(ctx, profileID)
	if err != nil {
		return nil, fmt.Errorf("list records: %w", err)
	}
	points := make([]models.MedicalTimelinePoint, 0, len(records))
	for _, r := range records {
		value, ok := r.Attributes[attributeKey]
		if !ok {
			continue
		}
		points = append(points, models.MedicalTimelinePoint{RecordID: r.ID, RecordDate: r.RecordDate, Value: value})
	}
	return points, nil
}

// MedicalRecordService manages medical records and their attached files.
type MedicalRecordService struct {
	records *supabaserepo.MedicalRecordRepository
	files   *supabaserepo.MedicalRecordFileRepository
	storage *supabaserepo.StorageRepository
	defs    *supabaserepo.MedicalAttributeDefRepository
	push    *PushService
	cfg     HouseTimerConfig // VAPID keys — same config shape as HouseTimerService, no email fields needed
}

// NewMedicalRecordService constructs a MedicalRecordService.
func NewMedicalRecordService(records *supabaserepo.MedicalRecordRepository, files *supabaserepo.MedicalRecordFileRepository, storage *supabaserepo.StorageRepository, defs *supabaserepo.MedicalAttributeDefRepository, push *PushService, cfg HouseTimerConfig) *MedicalRecordService {
	return &MedicalRecordService{records: records, files: files, storage: storage, defs: defs, push: push, cfg: cfg}
}

// ListByProfile returns a profile's records in date order.
func (s *MedicalRecordService) ListByProfile(ctx context.Context, profileID string) ([]*models.MedicalRecord, error) {
	return s.records.ListByProfile(ctx, profileID)
}

// FindByID looks up a single record, RLS-scoped.
func (s *MedicalRecordService) FindByID(ctx context.Context, id string) (*models.MedicalRecord, error) {
	return s.records.FindByID(ctx, id)
}

// Create adds a new record.
func (s *MedicalRecordService) Create(ctx context.Context, input models.MedicalRecordInput) (*models.MedicalRecord, error) {
	userID := auth.UserIDFromContext(ctx)
	if userID == "" {
		return nil, fmt.Errorf("no authenticated user")
	}
	if input.ProfileID == "" || input.Title == "" || input.RecordDate == "" {
		return nil, fmt.Errorf("profile_id, title, and record_date are required")
	}
	switch input.RecordType {
	case models.MedicalRecordExam, models.MedicalRecordVaccine, models.MedicalRecordImaging, models.MedicalRecordNoteType:
	case "":
		input.RecordType = models.MedicalRecordNoteType
	default:
		return nil, fmt.Errorf("invalid record_type: %s", input.RecordType)
	}
	input.CreatedBy = userID
	record, err := s.records.Create(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("create medical record: %w", err)
	}
	s.notifyOutOfRange(ctx, userID, record)
	return record, nil
}

// rangeRE matches "low - high" reference ranges; boundRE matches a single
// bound like "<100.00" or ">=126.00" — same shapes the frontend parses.
var rangeRE = regexp.MustCompile(`^(-?\d+(?:\.\d+)?)\s*-\s*(-?\d+(?:\.\d+)?)$`)
var boundRE = regexp.MustCompile(`^(<=|>=|<|>)\s*(-?\d+(?:\.\d+)?)$`)

// isOutOfRange reports whether value falls outside a reference_range
// string, or nil if the range isn't in a recognized shape.
func isOutOfRange(value any, rangeStr string) *bool {
	if rangeStr == "" {
		return nil
	}
	var num float64
	switch v := value.(type) {
	case float64:
		num = v
	case string:
		n, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return nil
		}
		num = n
	default:
		return nil
	}

	if m := rangeRE.FindStringSubmatch(rangeStr); m != nil {
		lo, _ := strconv.ParseFloat(m[1], 64)
		hi, _ := strconv.ParseFloat(m[2], 64)
		out := num < lo || num > hi
		return &out
	}
	if m := boundRE.FindStringSubmatch(rangeStr); m != nil {
		limit, _ := strconv.ParseFloat(m[2], 64)
		var out bool
		switch m[1] {
		case "<":
			out = !(num < limit)
		case "<=":
			out = !(num <= limit)
		case ">":
			out = !(num > limit)
		default:
			out = !(num >= limit)
		}
		return &out
	}
	return nil
}

// notifyOutOfRange best-effort push-notifies the record's owner when any of
// its attributes falls outside that attribute's known reference range.
// Never fails the Create — logs and returns on any error.
func (s *MedicalRecordService) notifyOutOfRange(ctx context.Context, userID string, record *models.MedicalRecord) {
	if s.cfg.VAPIDPublicKey == "" || s.cfg.VAPIDPrivateKey == "" || len(record.Attributes) == 0 {
		return
	}
	flagged, err := s.flaggedAttributeLabels(ctx, record)
	if err != nil {
		log.Printf("medical: list attribute defs for push check: %v", err)
		return
	}
	if len(flagged) == 0 {
		return
	}

	subs, err := s.push.ListByUserID(ctx, userID)
	if err != nil {
		log.Printf("medical: list push subscriptions: %v", err)
		return
	}
	body := fmt.Sprintf("%s: %s out of range", record.Title, strings.Join(flagged, ", "))
	payload, err := json.Marshal(map[string]string{"title": "Lab value out of range", "body": body})
	if err != nil {
		return
	}
	s.sendPushToAll(ctx, subs, payload)
}

// flaggedAttributeLabels returns the friendly label (or raw key) of every
// attribute on record whose value falls outside its known reference range.
func (s *MedicalRecordService) flaggedAttributeLabels(ctx context.Context, record *models.MedicalRecord) ([]string, error) {
	defs, err := s.defs.List(ctx)
	if err != nil {
		return nil, err
	}
	defByKey := make(map[string]*models.MedicalAttributeDef, len(defs))
	for _, d := range defs {
		defByKey[d.AttrKey] = d
	}

	var flagged []string
	for key, value := range record.Attributes {
		def := defByKey[key]
		if def == nil || def.ReferenceRange == nil {
			continue
		}
		if out := isOutOfRange(value, *def.ReferenceRange); out != nil && *out {
			label := key
			if def.Label != nil && *def.Label != "" {
				label = *def.Label
			}
			flagged = append(flagged, label)
		}
	}
	return flagged, nil
}

// sendPushToAll best-effort-sends payload to every subscription, pruning
// any that the push service reports as gone.
func (s *MedicalRecordService) sendPushToAll(ctx context.Context, subs []*models.PushSubscription, payload []byte) {
	for _, sub := range subs {
		resp, err := webpush.SendNotificationWithContext(ctx, payload, &webpush.Subscription{
			Endpoint: sub.Endpoint,
			Keys:     webpush.Keys{P256dh: sub.P256dh, Auth: sub.AuthKey},
		}, &webpush.Options{
			VAPIDPublicKey:  s.cfg.VAPIDPublicKey,
			VAPIDPrivateKey: s.cfg.VAPIDPrivateKey,
			Subscriber:      s.cfg.VAPIDSubject,
			TTL:             3600,
		})
		if err != nil {
			log.Printf("medical: send push: %v", err)
			continue
		}
		_ = resp.Body.Close()
		if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone {
			_ = s.push.Delete(ctx, sub.ID)
		}
	}
}

// Update patches a record.
func (s *MedicalRecordService) Update(ctx context.Context, id string, fields map[string]any) (*models.MedicalRecord, error) {
	record, err := s.records.Update(ctx, id, fields)
	if err != nil {
		return nil, fmt.Errorf("update medical record: %w", err)
	}
	return record, nil
}

// Delete removes a record and its attached files (storage objects
// best-effort — the row delete, which cascades to medical_record_files, is
// what matters for access).
func (s *MedicalRecordService) Delete(ctx context.Context, id string) error {
	files, err := s.files.ListByRecord(ctx, id)
	if err == nil {
		for _, f := range files {
			_ = s.storage.DeleteObject(ctx, medicalFilesBucket, f.StoragePath)
		}
	}
	return s.records.Delete(ctx, id)
}

// ListFiles returns a record's attached files.
func (s *MedicalRecordService) ListFiles(ctx context.Context, recordID string) ([]*models.MedicalRecordFile, error) {
	return s.files.ListByRecord(ctx, recordID)
}

// UploadFile stores file in the private medical-files bucket, under the
// owning profile's folder, and records its metadata.
func (s *MedicalRecordService) UploadFile(ctx context.Context, recordID string, file io.Reader, fileName, contentType string) (*models.MedicalRecordFile, error) {
	record, err := s.records.FindByID(ctx, recordID)
	if err != nil {
		return nil, fmt.Errorf("find record: %w", err)
	}
	if record == nil {
		return nil, fmt.Errorf("record not found")
	}

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("read uploaded file: %w", err)
	}

	path := fmt.Sprintf("%s/%d-%s", record.ProfileID, time.Now().UnixNano(), fileName)
	if _, err := s.storage.UploadObject(ctx, medicalFilesBucket, path, data, contentType); err != nil {
		return nil, fmt.Errorf("upload file: %w", err)
	}

	row, err := s.files.Create(ctx, &models.MedicalRecordFile{
		RecordID:    recordID,
		StoragePath: path,
		FileName:    fileName,
		ContentType: &contentType,
	})
	if err != nil {
		return nil, fmt.Errorf("record file metadata: %w", err)
	}
	return row, nil
}

// DownloadFile returns a file's raw bytes plus its metadata.
func (s *MedicalRecordService) DownloadFile(ctx context.Context, fileID string) (*models.MedicalRecordFile, []byte, error) {
	f, err := s.files.FindByID(ctx, fileID)
	if err != nil {
		return nil, nil, fmt.Errorf("find file: %w", err)
	}
	if f == nil {
		return nil, nil, fmt.Errorf("file not found")
	}
	data, err := s.storage.DownloadObject(ctx, medicalFilesBucket, f.StoragePath)
	if err != nil {
		return nil, nil, fmt.Errorf("download file: %w", err)
	}
	return f, data, nil
}

// DeleteFile removes a file's row and its stored object.
func (s *MedicalRecordService) DeleteFile(ctx context.Context, fileID string) error {
	f, err := s.files.FindByID(ctx, fileID)
	if err != nil {
		return fmt.Errorf("find file: %w", err)
	}
	if f == nil {
		return nil
	}
	if err := s.files.Delete(ctx, fileID); err != nil {
		return err
	}
	_ = s.storage.DeleteObject(ctx, medicalFilesBucket, f.StoragePath)
	return nil
}

// MedicalAttributeDefService manages per-user attribute label/description
// definitions.
type MedicalAttributeDefService struct {
	repo *supabaserepo.MedicalAttributeDefRepository
}

// NewMedicalAttributeDefService constructs a MedicalAttributeDefService.
func NewMedicalAttributeDefService(repo *supabaserepo.MedicalAttributeDefRepository) *MedicalAttributeDefService {
	return &MedicalAttributeDefService{repo: repo}
}

// List returns every def the caller owns.
func (s *MedicalAttributeDefService) List(ctx context.Context) ([]*models.MedicalAttributeDef, error) {
	return s.repo.List(ctx)
}

// Upsert creates or updates a def for the caller.
func (s *MedicalAttributeDefService) Upsert(ctx context.Context, input models.MedicalAttributeDefInput) (*models.MedicalAttributeDef, error) {
	userID := auth.UserIDFromContext(ctx)
	if userID == "" {
		return nil, fmt.Errorf("no authenticated user")
	}
	if input.AttrKey == "" {
		return nil, fmt.Errorf("attr_key is required")
	}
	def, err := s.repo.Upsert(ctx, &models.MedicalAttributeDef{
		UserID: userID, AttrKey: input.AttrKey, Label: input.Label, Description: input.Description, ReferenceRange: input.ReferenceRange,
	})
	if err != nil {
		return nil, fmt.Errorf("upsert attribute def: %w", err)
	}
	return def, nil
}

// MedicalMedicationService manages ongoing medications for a profile.
type MedicalMedicationService struct {
	repo *supabaserepo.MedicalMedicationRepository
}

// NewMedicalMedicationService constructs a MedicalMedicationService.
func NewMedicalMedicationService(repo *supabaserepo.MedicalMedicationRepository) *MedicalMedicationService {
	return &MedicalMedicationService{repo: repo}
}

// ListByProfile returns a profile's medications.
func (s *MedicalMedicationService) ListByProfile(ctx context.Context, profileID string) ([]*models.MedicalMedication, error) {
	return s.repo.ListByProfile(ctx, profileID)
}

// Create adds a new medication.
func (s *MedicalMedicationService) Create(ctx context.Context, input models.MedicalMedicationInput) (*models.MedicalMedication, error) {
	if input.ProfileID == "" || input.Name == "" {
		return nil, fmt.Errorf("profile_id and name are required")
	}
	med, err := s.repo.Create(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("create medication: %w", err)
	}
	return med, nil
}

// Update patches a medication.
func (s *MedicalMedicationService) Update(ctx context.Context, id string, fields map[string]any) (*models.MedicalMedication, error) {
	med, err := s.repo.Update(ctx, id, fields)
	if err != nil {
		return nil, fmt.Errorf("update medication: %w", err)
	}
	return med, nil
}

// Delete removes a medication.
func (s *MedicalMedicationService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

// labLineRE matches a lab report line ending in "<label> <value> <optional
// range>", e.g. "Hemoglobina 14.80 g/dL 13.50 - 17.90" or
// "Colesterol LDL 125.20 mg/dL Deseable: <100.00". Deliberately permissive
// — this feeds a review step, not a direct save.
var labLineRE = regexp.MustCompile(`^([A-Za-zÁÉÍÓÚÑáéíóúñ() /%-]{3,60}?)\s+(\d+\.\d+|\d+)\s*(?:\*\*|\+|-)?\s*([A-Za-z/%^0-9]*)\s*(.*)$`)

// ParseLabAttributes best-effort-parses lines of an extracted lab PDF's
// text into {label, value, range} suggestions for the user to review and
// rename before adding to a record. Never persisted directly.
func ParseLabAttributes(text string) []models.SuggestedAttribute {
	var out []models.SuggestedAttribute
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		m := labLineRE.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		label := strings.TrimSpace(m[1])
		if label == "" {
			continue
		}
		rangePart := strings.TrimSpace(m[4])
		out = append(out, models.SuggestedAttribute{Label: label, Value: m[2], Range: rangePart})
	}
	return out
}
