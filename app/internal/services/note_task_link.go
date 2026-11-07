package services

import (
	"context"
	"errors"
	"fmt"
	"log"

	"life-base-api/app/internal/models"
)

// Notes <-> TODO linking. A checklist item in a note can be turned into a
// `todo` task (tasks.tasks.note_id points back at the note, and a hidden
// "{task:UUID}" marker on the checklist line points at the task). From then
// on the two stay in sync, best-effort, in both directions:
//
//   - completing the task ticks the checklist line  (TaskService.Complete
//     -> NoteTaskLinkService.TickNoteChecklist)
//   - ticking the line through a note update completes the task
//     (NoteService.Update -> NoteTaskLinkService.CompleteLinkedTasks)
//
// Neither direction may fail the operation that triggered it, and neither
// may re-trigger the other: every write made by this file runs under a
// context that carries the skip flag below, and both hooks return
// immediately when it is set. Both hooks are also idempotent on their own
// (SetTaskChecked reports no change for an already-ticked line, and
// completed tasks are skipped), so the flag is a second safety net.

// ErrNoteNotFound is returned when a note id does not resolve to a note.
var ErrNoteNotFound = errors.New("note not found")

type noteTaskSyncSkipKey struct{}

func withNoteTaskSyncSkipped(ctx context.Context) context.Context {
	return context.WithValue(ctx, noteTaskSyncSkipKey{}, true)
}

func noteTaskSyncSkipped(ctx context.Context) bool {
	v, _ := ctx.Value(noteTaskSyncSkipKey{}).(bool)
	return v
}

// NoteTaskSyncer is what NoteService calls after a note update to complete
// the tasks whose checklist lines were just ticked.
type NoteTaskSyncer interface {
	CompleteLinkedTasks(ctx context.Context, noteID string, taskIDs []string)
}

// TaskNoteSyncer is what TaskService calls after completing a task that has
// a note_id, to tick the matching checklist line.
type TaskNoteSyncer interface {
	TickNoteChecklist(ctx context.Context, task *models.Task)
}

// NoteTaskLinkService implements both directions of the sync plus the
// "make task from checklist item" action. It depends on the concrete note
// and task services; they hold it back only through the two interfaces
// above, which is what breaks the construction cycle (see Registry).
type NoteTaskLinkService struct {
	notes *NoteService
	tasks *TaskService
}

// NewNoteTaskLinkService constructs a NoteTaskLinkService.
func NewNoteTaskLinkService(notes *NoteService, tasks *TaskService) *NoteTaskLinkService {
	return &NoteTaskLinkService{notes: notes, tasks: tasks}
}

// SetTaskSync installs the hook NoteService.Update calls after saving.
func (s *NoteService) SetTaskSync(sync NoteTaskSyncer) {
	s.taskSync = sync
}

// SetNoteSync installs the hook TaskService.Complete calls after completing.
func (s *TaskService) SetNoteSync(sync TaskNoteSyncer) {
	s.noteSync = sync
}

// syncLinkedTasks completes the tasks behind checklist lines that this
// update flipped from unchecked to checked. Called by NoteService.Update
// after the note was saved; never returns an error.
func (s *NoteService) syncLinkedTasks(ctx context.Context, noteID string, previous *models.Note, fields map[string]any) {
	if s.taskSync == nil || noteTaskSyncSkipped(ctx) {
		return
	}
	next, ok := fields["content"].(string)
	if !ok || previous == nil || previous.Content == nil {
		return
	}
	ids := NewlyCheckedTaskIDs(*previous.Content, next)
	if len(ids) == 0 {
		return
	}
	s.taskSync.CompleteLinkedTasks(ctx, noteID, ids)
}

// CompleteLinkedTasks implements NoteTaskSyncer.
func (s *NoteTaskLinkService) CompleteLinkedTasks(ctx context.Context, noteID string, taskIDs []string) {
	if noteTaskSyncSkipped(ctx) {
		return
	}
	ctx = withNoteTaskSyncSkipped(ctx)
	for _, id := range taskIDs {
		task, err := s.tasks.FindByID(ctx, id)
		if err != nil {
			log.Printf("note-task sync: find task %s: %v", id, err)
			continue
		}
		// Ignore markers that point at nothing, at a task from another note,
		// or at a task that is already done.
		if task == nil || task.NoteID == nil || *task.NoteID != noteID || task.CompletedAt != nil {
			continue
		}
		if _, err := s.tasks.Complete(ctx, id); err != nil {
			log.Printf("note-task sync: complete task %s from note %s: %v", id, noteID, err)
		}
	}
}

// TickNoteChecklist implements TaskNoteSyncer.
func (s *NoteTaskLinkService) TickNoteChecklist(ctx context.Context, task *models.Task) {
	if task == nil || task.NoteID == nil || noteTaskSyncSkipped(ctx) {
		return
	}
	ctx = withNoteTaskSyncSkipped(ctx)
	note, err := s.notes.FindByID(ctx, *task.NoteID)
	if err != nil {
		log.Printf("note-task sync: find note %s: %v", *task.NoteID, err)
		return
	}
	if note == nil || note.Content == nil {
		return
	}
	updated, changed := SetTaskChecked(*note.Content, task.ID, true)
	if !changed {
		return
	}
	if _, err := s.notes.Update(ctx, note.ID, map[string]any{"content": updated}); err != nil {
		log.Printf("note-task sync: tick checklist in note %s for task %s: %v", note.ID, task.ID, err)
	}
}

// CreateTaskFromChecklistItem turns the checklist item on line lineIndex of
// the note into a `todo` task and writes the {task:UUID} marker onto that
// line (snapshotting a note version like any other note update). If the
// line is already ticked the new task is created already completed.
func (s *NoteTaskLinkService) CreateTaskFromChecklistItem(ctx context.Context, noteID string, lineIndex int, in models.ChecklistTaskInput) (*models.ChecklistTaskResult, error) {
	note, err := s.notes.FindByID(ctx, noteID)
	if err != nil {
		return nil, fmt.Errorf("find note: %w", err)
	}
	if note == nil {
		return nil, ErrNoteNotFound
	}
	if note.Content == nil {
		return nil, fmt.Errorf("note has no checklist")
	}
	item, ok := FindChecklistItem(*note.Content, lineIndex)
	if !ok {
		return nil, fmt.Errorf("no checklist item on line %d", lineIndex)
	}
	if item.TaskID != "" {
		return nil, fmt.Errorf("checklist item is already linked to a task")
	}
	if item.Text == "" {
		return nil, fmt.Errorf("checklist item has no text")
	}

	input := models.TaskInput{
		Category: models.TaskTodo,
		Title:    item.Text,
		Priority: in.Priority,
		NoteID:   &noteID,
	}
	if in.DueDate != nil && *in.DueDate != "" {
		input.DueDate = in.DueDate
	}
	skipCtx := withNoteTaskSyncSkipped(ctx)
	task, err := s.tasks.Create(skipCtx, input)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, fmt.Errorf("create task: empty response")
	}

	content, ok := AttachTaskMarker(*note.Content, lineIndex, task.ID)
	if !ok {
		s.discardTask(skipCtx, task.ID)
		return nil, fmt.Errorf("no checklist item on line %d", lineIndex)
	}
	updatedNote, err := s.notes.Update(skipCtx, noteID, map[string]any{"content": content})
	if err != nil {
		s.discardTask(skipCtx, task.ID)
		return nil, fmt.Errorf("write task marker: %w", err)
	}

	if item.Checked {
		done, err := s.tasks.Complete(skipCtx, task.ID)
		if err != nil {
			log.Printf("note-task sync: complete new task %s for ticked item: %v", task.ID, err)
		} else if done != nil {
			task = done
		}
	}
	return &models.ChecklistTaskResult{Task: task, Note: updatedNote}, nil
}

// discardTask removes a just-created task when the note write that was
// supposed to link it failed, so no unlinked orphan is left behind.
func (s *NoteTaskLinkService) discardTask(ctx context.Context, taskID string) {
	if err := s.tasks.Delete(ctx, taskID); err != nil {
		log.Printf("note-task sync: delete orphan task %s: %v", taskID, err)
	}
}
