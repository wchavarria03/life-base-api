package models

import "time"

// MedicalRecordType discriminates the kind of medical record.
type MedicalRecordType string

// MedicalRecordType values.
const (
	MedicalRecordExam     MedicalRecordType = "exam"
	MedicalRecordVaccine  MedicalRecordType = "vaccine"
	MedicalRecordImaging  MedicalRecordType = "imaging"
	MedicalRecordNoteType MedicalRecordType = "note"
)

// MedicalAccessRole is a grant's permission level.
type MedicalAccessRole string

// MedicalAccessRole values.
const (
	MedicalAccessViewer MedicalAccessRole = "viewer"
	MedicalAccessEditor MedicalAccessRole = "editor"
)

// MedicalProfile is the stored shape from medical_profiles — one person's
// medical history (self, a child, etc.), owned by the user who created it.
type MedicalProfile struct {
	ID           string    `json:"id,omitempty"`
	OwnerUserID  string    `json:"owner_user_id,omitempty"`
	Name         string    `json:"name"`
	Relationship *string   `json:"relationship,omitempty"`
	DOB          *string   `json:"dob,omitempty"`
	CreatedAt    time.Time `json:"created_at,omitempty"`
}

// MedicalProfileInput is the write shape for create/update.
type MedicalProfileInput struct {
	OwnerUserID  string  `json:"owner_user_id,omitempty"`
	Name         string  `json:"name,omitempty"`
	Relationship *string `json:"relationship,omitempty"`
	DOB          *string `json:"dob,omitempty"`
}

// MedicalProfileAccess is a grant of viewer/editor access to another gmail
// account, enforced off that account's JWT email claim at login — no
// invite/signup flow.
type MedicalProfileAccess struct {
	ID        string            `json:"id,omitempty"`
	ProfileID string            `json:"profile_id,omitempty"`
	Email     string            `json:"email"`
	Role      MedicalAccessRole `json:"role"`
	CreatedAt time.Time         `json:"created_at,omitempty"`
}

// MedicalProfileAccessInput is the write shape for granting access.
type MedicalProfileAccessInput struct {
	ProfileID string            `json:"profile_id,omitempty"`
	Email     string            `json:"email"`
	Role      MedicalAccessRole `json:"role"`
}

// MedicalRecord is the stored shape from medical_records — one exam,
// vaccine, imaging study, or free note, with structured attributes (e.g.
// {"weight_kg": 70.5}) for timeline charting.
type MedicalRecord struct {
	ID         string            `json:"id,omitempty"`
	ProfileID  string            `json:"profile_id,omitempty"`
	CreatedBy  string            `json:"created_by,omitempty"`
	RecordType MedicalRecordType `json:"record_type"`
	Title      string            `json:"title"`
	RecordDate string            `json:"record_date"`
	Attributes map[string]any    `json:"attributes"`
	// AttributeGroups optionally maps an attribute key to the section it
	// appeared under in the source exam (e.g. "Serie blanca"), for grouped
	// display. Keys with no entry fall back to an "Other" group in the UI.
	AttributeGroups map[string]string `json:"attribute_groups"`
	Notes           *string           `json:"notes,omitempty"`
	DoctorName      *string           `json:"doctor_name,omitempty"`
	Recommendations *string           `json:"recommendations,omitempty"`
	LinkedNoteID    *string           `json:"linked_note_id,omitempty"`
	CreatedAt       time.Time         `json:"created_at,omitempty"`
}

// MedicalRecordInput is the write shape for create/update.
type MedicalRecordInput struct {
	ProfileID       string            `json:"profile_id,omitempty"`
	CreatedBy       string            `json:"created_by,omitempty"`
	RecordType      MedicalRecordType `json:"record_type,omitempty"`
	Title           string            `json:"title,omitempty"`
	RecordDate      string            `json:"record_date,omitempty"`
	Attributes      map[string]any    `json:"attributes,omitempty"`
	AttributeGroups map[string]string `json:"attribute_groups,omitempty"`
	Notes           *string           `json:"notes,omitempty"`
	DoctorName      *string           `json:"doctor_name,omitempty"`
	Recommendations *string           `json:"recommendations,omitempty"`
	LinkedNoteID    *string           `json:"linked_note_id,omitempty"`
}

// MedicalRecordFile is one attached file (X-ray, lab PDF) on a record — the
// file itself lives in the private "medical-files" storage bucket.
type MedicalRecordFile struct {
	ID          string    `json:"id,omitempty"`
	RecordID    string    `json:"record_id,omitempty"`
	StoragePath string    `json:"storage_path"`
	FileName    string    `json:"file_name"`
	ContentType *string   `json:"content_type,omitempty"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
}

// MedicalTimelinePoint is one {date, value} sample of a single attribute
// key, extracted from a profile's records in date order.
type MedicalTimelinePoint struct {
	RecordID   string `json:"record_id"`
	RecordDate string `json:"record_date"`
	Value      any    `json:"value"`
}
