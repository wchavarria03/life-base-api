package services

import (
	"context"
	"fmt"

	"life-base-api/app/internal/auth"
	"life-base-api/app/internal/models"
	supabaserepo "life-base-api/app/internal/repositories/supabase"
	"life-base-api/app/internal/vault"
)

// protectedNoteFields can never be set through the generic PATCH /v1/notes/:id
// body: they are owned by the vault sync pipeline (or the dedicated
// /sensitive and /resolve-conflict endpoints), and letting a client set e.g.
// vault_path would let it steer which file the sync worker writes.
var protectedNoteFields = []string{
	"id", "user_id", "created_at", "updated_at",
	"source", "vault_path", "org_id", "content_format", "vault_commit",
	"pending_write", "conflict", "deleted_at", "sensitive", "sensitive_override",
	"search_vec", "rendered_html",
}

type NoteRepository interface {
	crudRepository[models.Note, models.NoteInput]
}

type NoteService struct {
	notes    NoteRepository
	versions *supabaserepo.NoteVersionRepository

	// taskSync completes linked tasks when a checklist line gets ticked;
	// nil until the registry wires it (see note_task_link.go).
	taskSync NoteTaskSyncer
}

// NewNoteService constructs a NoteService.
func NewNoteService(notes NoteRepository, versions *supabaserepo.NoteVersionRepository) *NoteService {
	return &NoteService{notes: notes, versions: versions}
}

func (s *NoteService) List(ctx context.Context) ([]*models.Note, error) {
	return s.notes.List(ctx)
}

// FindByID returns a note; org notes also get a server-sanitized
// RenderedHTML (never stored).
func (s *NoteService) FindByID(ctx context.Context, id string) (*models.Note, error) {
	note, err := s.notes.FindByID(ctx, id)
	if err != nil || note == nil {
		return note, err
	}
	if note.ContentFormat == "org" && note.Content != nil {
		note.RenderedHTML = vault.RenderOrg(*note.Content, vault.RenderOptions{})
	}
	return note, nil
}

func (s *NoteService) Create(ctx context.Context, input models.NoteInput) (*models.Note, error) {
	if auth.UserIDFromContext(ctx) == "" {
		return nil, fmt.Errorf("no authenticated user")
	}
	if input.Title == "" {
		return nil, fmt.Errorf("title is required")
	}
	return s.notes.Create(ctx, input)
}

// Update patches a note, first archiving its current title/content as a
// new version — so every edit (e.g. a nutritionist's updated plan) leaves
// the prior one visible in history.
func (s *NoteService) Update(ctx context.Context, id string, fields map[string]any) (*models.Note, error) {
	for _, k := range protectedNoteFields {
		delete(fields, k)
	}
	if len(fields) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}
	current, err := s.notes.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("find note: %w", err)
	}
	if current != nil {
		applyVaultEditRules(current, fields)
		if len(fields) == 0 {
			return current, nil
		}
		existing, err := s.versions.ListByNote(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("list note versions: %w", err)
		}
		next := 1
		if len(existing) > 0 {
			next = existing[0].VersionNumber + 1
		}
		if _, err := s.versions.Create(ctx, &models.NoteVersion{
			NoteID: id, Title: current.Title, Content: current.Content, VersionNumber: next,
		}); err != nil {
			return nil, fmt.Errorf("archive note version: %w", err)
		}
	}
	updated, err := s.notes.Update(ctx, id, fields)
	if err != nil {
		return nil, err
	}
	s.syncLinkedTasks(ctx, id, current, fields)
	return updated, nil
}

// applyVaultEditRules adds the vault-sync side effects of an in-app edit:
//   - a content/title edit of a vault-sourced note marks it pending_write so
//     the homelab worker writes it back to git;
//   - for vault notes the title lives inside the file, so it is re-derived
//     from the new content instead of being set directly;
//   - if the new content declares itself sensitive, the flag is turned on.
func applyVaultEditRules(current *models.Note, fields map[string]any) {
	content, hasContent := fields["content"].(string)
	if current.Source == "vault" {
		// The title lives inside the file; ignore a directly-sent title.
		delete(fields, "title")
	}
	if !hasContent {
		return
	}
	format := current.ContentFormat
	if format == "" {
		format = "md"
	}
	path := ""
	if current.VaultPath != nil {
		path = *current.VaultPath
	}
	meta := vault.ParseNote(format, content, path)
	if meta.Sensitive {
		fields["sensitive"] = true
	}
	if current.Source == "vault" {
		fields["title"] = meta.Title
		fields["pending_write"] = true
	}
}

// ListVersions returns a note's archived prior versions, newest first.
func (s *NoteService) ListVersions(ctx context.Context, noteID string) ([]*models.NoteVersion, error) {
	return s.versions.ListByNote(ctx, noteID)
}

func (s *NoteService) Delete(ctx context.Context, id string) error {
	return s.notes.Delete(ctx, id)
}
