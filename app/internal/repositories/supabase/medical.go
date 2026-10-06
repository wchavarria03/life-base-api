package supabase

import (
	"context"
	"net/url"

	"life-base-api/app/internal/databases"
	"life-base-api/app/internal/models"
)

// MedicalProfileRepository persists medical_profiles rows.
type MedicalProfileRepository struct {
	client *databases.SupabaseClient
}

// NewMedicalProfileRepository constructs a MedicalProfileRepository.
func NewMedicalProfileRepository(client *databases.SupabaseClient) *MedicalProfileRepository {
	return &MedicalProfileRepository{client: client}
}

// List returns every profile visible to the caller (own + granted, via RLS).
func (r *MedicalProfileRepository) List(ctx context.Context) ([]*models.MedicalProfile, error) {
	return databases.Get[[]*models.MedicalProfile](ctx, r.client, "/rest/v1/medical_profiles",
		url.Values{"order": []string{"created_at.asc"}})
}

// FindByID looks up one profile (RLS-scoped).
func (r *MedicalProfileRepository) FindByID(ctx context.Context, id string) (*models.MedicalProfile, error) {
	return databases.First(databases.Get[[]*models.MedicalProfile](ctx, r.client, "/rest/v1/medical_profiles", url.Values{
		"id": []string{"eq." + id}, "limit": []string{"1"},
	}))
}

// Create inserts a new profile.
func (r *MedicalProfileRepository) Create(ctx context.Context, input models.MedicalProfileInput) (*models.MedicalProfile, error) {
	return databases.First(databases.Post[[]*models.MedicalProfile](ctx, r.client, "/rest/v1/medical_profiles", input, "return=representation"))
}

// Update patches a profile.
func (r *MedicalProfileRepository) Update(ctx context.Context, id string, fields map[string]any) (*models.MedicalProfile, error) {
	return databases.First(databases.Patch[[]*models.MedicalProfile](ctx, r.client, "/rest/v1/medical_profiles", databases.EqID(id), fields, "return=representation"))
}

// Delete removes a profile (and, via cascade, its records/access grants).
func (r *MedicalProfileRepository) Delete(ctx context.Context, id string) error {
	return databases.Delete(ctx, r.client, "/rest/v1/medical_profiles", databases.EqID(id))
}

// MedicalAccessRepository persists medical_profile_access rows.
type MedicalAccessRepository struct {
	client *databases.SupabaseClient
}

// NewMedicalAccessRepository constructs a MedicalAccessRepository.
func NewMedicalAccessRepository(client *databases.SupabaseClient) *MedicalAccessRepository {
	return &MedicalAccessRepository{client: client}
}

// ListByProfile returns every grant on a profile (owner-only via RLS).
func (r *MedicalAccessRepository) ListByProfile(ctx context.Context, profileID string) ([]*models.MedicalProfileAccess, error) {
	return databases.Get[[]*models.MedicalProfileAccess](ctx, r.client, "/rest/v1/medical_profile_access", url.Values{
		"profile_id": []string{"eq." + profileID}, "order": []string{"created_at.asc"},
	})
}

// Grant inserts or replaces a grant (unique on profile_id+email).
func (r *MedicalAccessRepository) Grant(ctx context.Context, a *models.MedicalProfileAccess) (*models.MedicalProfileAccess, error) {
	return databases.First(databases.Post[[]*models.MedicalProfileAccess](ctx, r.client,
		"/rest/v1/medical_profile_access?on_conflict=profile_id,email", a,
		"resolution=merge-duplicates,return=representation"))
}

// Revoke removes a grant.
func (r *MedicalAccessRepository) Revoke(ctx context.Context, id string) error {
	return databases.Delete(ctx, r.client, "/rest/v1/medical_profile_access", databases.EqID(id))
}

// MedicalRecordRepository persists medical_records rows.
type MedicalRecordRepository struct {
	client *databases.SupabaseClient
}

// NewMedicalRecordRepository constructs a MedicalRecordRepository.
func NewMedicalRecordRepository(client *databases.SupabaseClient) *MedicalRecordRepository {
	return &MedicalRecordRepository{client: client}
}

// ListByProfile returns a profile's records, oldest first (timeline order).
func (r *MedicalRecordRepository) ListByProfile(ctx context.Context, profileID string) ([]*models.MedicalRecord, error) {
	return databases.Get[[]*models.MedicalRecord](ctx, r.client, "/rest/v1/medical_records", url.Values{
		"profile_id": []string{"eq." + profileID}, "order": []string{"record_date.asc"},
	})
}

// FindByID looks up one record (RLS-scoped).
func (r *MedicalRecordRepository) FindByID(ctx context.Context, id string) (*models.MedicalRecord, error) {
	return databases.First(databases.Get[[]*models.MedicalRecord](ctx, r.client, "/rest/v1/medical_records", url.Values{
		"id": []string{"eq." + id}, "limit": []string{"1"},
	}))
}

// Create inserts a new record.
func (r *MedicalRecordRepository) Create(ctx context.Context, input models.MedicalRecordInput) (*models.MedicalRecord, error) {
	return databases.First(databases.Post[[]*models.MedicalRecord](ctx, r.client, "/rest/v1/medical_records", input, "return=representation"))
}

// Update patches a record.
func (r *MedicalRecordRepository) Update(ctx context.Context, id string, fields map[string]any) (*models.MedicalRecord, error) {
	return databases.First(databases.Patch[[]*models.MedicalRecord](ctx, r.client, "/rest/v1/medical_records", databases.EqID(id), fields, "return=representation"))
}

// Delete removes a record (and, via cascade, its attached files).
func (r *MedicalRecordRepository) Delete(ctx context.Context, id string) error {
	return databases.Delete(ctx, r.client, "/rest/v1/medical_records", databases.EqID(id))
}

// MedicalRecordFileRepository persists medical_record_files rows.
type MedicalRecordFileRepository struct {
	client *databases.SupabaseClient
}

// NewMedicalRecordFileRepository constructs a MedicalRecordFileRepository.
func NewMedicalRecordFileRepository(client *databases.SupabaseClient) *MedicalRecordFileRepository {
	return &MedicalRecordFileRepository{client: client}
}

// ListByRecord returns a record's attached files.
func (r *MedicalRecordFileRepository) ListByRecord(ctx context.Context, recordID string) ([]*models.MedicalRecordFile, error) {
	return databases.Get[[]*models.MedicalRecordFile](ctx, r.client, "/rest/v1/medical_record_files", url.Values{
		"record_id": []string{"eq." + recordID}, "order": []string{"created_at.asc"},
	})
}

// FindByID looks up one file row (RLS-scoped).
func (r *MedicalRecordFileRepository) FindByID(ctx context.Context, id string) (*models.MedicalRecordFile, error) {
	return databases.First(databases.Get[[]*models.MedicalRecordFile](ctx, r.client, "/rest/v1/medical_record_files", url.Values{
		"id": []string{"eq." + id}, "limit": []string{"1"},
	}))
}

// Create inserts a new file row.
func (r *MedicalRecordFileRepository) Create(ctx context.Context, f *models.MedicalRecordFile) (*models.MedicalRecordFile, error) {
	return databases.First(databases.Post[[]*models.MedicalRecordFile](ctx, r.client, "/rest/v1/medical_record_files", f, "return=representation"))
}

// Delete removes a file row.
func (r *MedicalRecordFileRepository) Delete(ctx context.Context, id string) error {
	return databases.Delete(ctx, r.client, "/rest/v1/medical_record_files", databases.EqID(id))
}

// MedicalAttributeDefRepository persists medical_attribute_defs rows.
type MedicalAttributeDefRepository struct {
	client *databases.SupabaseClient
}

// NewMedicalAttributeDefRepository constructs a MedicalAttributeDefRepository.
func NewMedicalAttributeDefRepository(client *databases.SupabaseClient) *MedicalAttributeDefRepository {
	return &MedicalAttributeDefRepository{client: client}
}

// List returns every attribute def the caller owns.
func (r *MedicalAttributeDefRepository) List(ctx context.Context) ([]*models.MedicalAttributeDef, error) {
	return databases.Get[[]*models.MedicalAttributeDef](ctx, r.client, "/rest/v1/medical_attribute_defs",
		url.Values{"order": []string{"attr_key.asc"}})
}

// Upsert creates or replaces a def (unique on user_id+attr_key).
func (r *MedicalAttributeDefRepository) Upsert(ctx context.Context, d *models.MedicalAttributeDef) (*models.MedicalAttributeDef, error) {
	return databases.First(databases.Post[[]*models.MedicalAttributeDef](ctx, r.client,
		"/rest/v1/medical_attribute_defs?on_conflict=user_id,attr_key", d,
		"resolution=merge-duplicates,return=representation"))
}

// MedicalMedicationRepository persists medical_medications rows.
type MedicalMedicationRepository struct {
	client *databases.SupabaseClient
}

// NewMedicalMedicationRepository constructs a MedicalMedicationRepository.
func NewMedicalMedicationRepository(client *databases.SupabaseClient) *MedicalMedicationRepository {
	return &MedicalMedicationRepository{client: client}
}

// ListByProfile returns a profile's medications.
func (r *MedicalMedicationRepository) ListByProfile(ctx context.Context, profileID string) ([]*models.MedicalMedication, error) {
	return databases.Get[[]*models.MedicalMedication](ctx, r.client, "/rest/v1/medical_medications", url.Values{
		"profile_id": []string{"eq." + profileID}, "order": []string{"created_at.asc"},
	})
}

// Create inserts a new medication.
func (r *MedicalMedicationRepository) Create(ctx context.Context, input models.MedicalMedicationInput) (*models.MedicalMedication, error) {
	return databases.First(databases.Post[[]*models.MedicalMedication](ctx, r.client, "/rest/v1/medical_medications", input, "return=representation"))
}

// Update patches a medication.
func (r *MedicalMedicationRepository) Update(ctx context.Context, id string, fields map[string]any) (*models.MedicalMedication, error) {
	return databases.First(databases.Patch[[]*models.MedicalMedication](ctx, r.client, "/rest/v1/medical_medications", databases.EqID(id), fields, "return=representation"))
}

// Delete removes a medication.
func (r *MedicalMedicationRepository) Delete(ctx context.Context, id string) error {
	return databases.Delete(ctx, r.client, "/rest/v1/medical_medications", databases.EqID(id))
}
