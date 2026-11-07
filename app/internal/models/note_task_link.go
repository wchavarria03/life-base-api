package models

// ChecklistTaskInput is the optional request body of
// POST /v1/notes/:id/checklist/:index/task.
type ChecklistTaskInput struct {
	DueDate  *string `json:"due_date,omitempty"`
	Priority string  `json:"priority,omitempty"`
}

// ChecklistTaskResult is the response of that endpoint: the new task plus
// the note as rewritten with the hidden {task:UUID} marker, so the client
// can refresh without a second request.
type ChecklistTaskResult struct {
	Task *Task `json:"task"`
	Note *Note `json:"note"`
}
