package services

import (
	"regexp"
	"strings"
)

// This file is the single home for everything that understands checklist
// lines inside a note's content and the hidden "{task:UUID}" marker that
// links a checklist line to a task. All functions here are pure (string in,
// string out) so they can be unit tested without a database.
//
// A checklist line is a markdown/org list item with a checkbox:
//
//	- [ ] Buy filter
//	- [x] Buy filter {task:0b1c...}
//	+ [X] org-style done item
//	  1. [ ] indented / ordered item
//
// Lines inside fenced code blocks (``` or ~~~) are ignored.
//
// Item indexes used throughout are zero-based LINE indexes into
// strings.Split(content, "\n") — the same index the frontend already uses
// when toggling a checklist line. The marker, not the index, is what ties a
// line to a task afterwards, so later edits that shift lines never break the
// link.

// ChecklistItem is one parsed checklist line.
type ChecklistItem struct {
	// LineIndex is the zero-based line number of the item in the content.
	LineIndex int
	// Text is the item text with any task marker removed and trimmed.
	Text string
	// Checked is true for "[x]" / "[X]".
	Checked bool
	// TaskID is the linked task's id, or "" when the line has no marker.
	TaskID string
}

var (
	// checklistLineRe splits a checklist line into: prefix up to and
	// including "[", the check mark, the "]" plus following spaces, and the
	// rest of the line (item text, possibly ending in a task marker).
	checklistLineRe = regexp.MustCompile(`^(\s*(?:[-+]|\d+[.)])\s+\[)([ xX])(\]\s+)(.*)$`)

	// taskMarkerRe matches a trailing " {task:UUID}" marker.
	taskMarkerRe = regexp.MustCompile(`\s*\{task:([0-9A-Fa-f]{8}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{12})\}\s*$`)
)

// checklistLine is the decomposed form of one checklist line.
type checklistLine struct {
	prefix string // everything up to and including "["
	mark   string // " ", "x" or "X"
	infix  string // "]" plus the spaces before the text
	text   string // item text WITHOUT marker, right-trimmed
	taskID string // marker id or ""
	cr     string // "\r" when the line ended in CRLF, else ""
}

func (l checklistLine) checked() bool {
	return l.mark == "x" || l.mark == "X"
}

func (l checklistLine) render() string {
	text := l.text
	if l.taskID != "" {
		if text != "" {
			text += " "
		}
		text += "{task:" + l.taskID + "}"
	}
	return l.prefix + l.mark + l.infix + text + l.cr
}

// parseChecklistLine decomposes line, reporting false when it is not a
// checklist line.
func parseChecklistLine(line string) (checklistLine, bool) {
	body := strings.TrimSuffix(line, "\r")
	cr := line[len(body):]
	m := checklistLineRe.FindStringSubmatch(body)
	if m == nil {
		return checklistLine{}, false
	}
	text := m[4]
	taskID := ""
	if mm := taskMarkerRe.FindStringSubmatch(text); mm != nil {
		taskID = strings.ToLower(mm[1])
		text = taskMarkerRe.ReplaceAllString(text, "")
	}
	return checklistLine{
		prefix: m[1],
		mark:   m[2],
		infix:  m[3],
		text:   strings.TrimRight(text, " \t"),
		taskID: taskID,
		cr:     cr,
	}, true
}

// codeLines returns, for each line, whether it is a fence delimiter or sits
// inside a fenced code block. An unterminated fence runs to the end.
func codeLines(lines []string) []bool {
	inCode := make([]bool, len(lines))
	fence := ""
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		marker := ""
		if strings.HasPrefix(trimmed, "```") {
			marker = "```"
		} else if strings.HasPrefix(trimmed, "~~~") {
			marker = "~~~"
		}
		switch {
		case fence == "" && marker != "":
			fence = marker
			inCode[i] = true
		case fence != "":
			inCode[i] = true
			if marker == fence {
				fence = ""
			}
		}
	}
	return inCode
}

// ParseChecklist returns every checklist item in content, in document order.
func ParseChecklist(content string) []ChecklistItem {
	lines := strings.Split(content, "\n")
	inCode := codeLines(lines)
	items := []ChecklistItem{}
	for i, line := range lines {
		if inCode[i] {
			continue
		}
		cl, ok := parseChecklistLine(line)
		if !ok {
			continue
		}
		items = append(items, ChecklistItem{
			LineIndex: i,
			Text:      cl.text,
			Checked:   cl.checked(),
			TaskID:    cl.taskID,
		})
	}
	return items
}

// FindChecklistItem returns the checklist item at lineIndex, if that line is
// a checklist item outside a code fence.
func FindChecklistItem(content string, lineIndex int) (ChecklistItem, bool) {
	for _, it := range ParseChecklist(content) {
		if it.LineIndex == lineIndex {
			return it, true
		}
	}
	return ChecklistItem{}, false
}

// FindTaskIDs returns the distinct task ids marked in content, in order of
// first appearance.
func FindTaskIDs(content string) []string {
	ids := []string{}
	seen := map[string]bool{}
	for _, it := range ParseChecklist(content) {
		if it.TaskID == "" || seen[it.TaskID] {
			continue
		}
		seen[it.TaskID] = true
		ids = append(ids, it.TaskID)
	}
	return ids
}

// AttachTaskMarker writes " {task:taskID}" at the end of the checklist item
// on line lineIndex, replacing any marker already there. It reports false
// (content unchanged) when that line is not a checklist item. Attaching the
// same id twice is a no-op.
func AttachTaskMarker(content string, lineIndex int, taskID string) (string, bool) {
	lines := strings.Split(content, "\n")
	if lineIndex < 0 || lineIndex >= len(lines) {
		return content, false
	}
	if codeLines(lines)[lineIndex] {
		return content, false
	}
	cl, ok := parseChecklistLine(lines[lineIndex])
	if !ok {
		return content, false
	}
	cl.taskID = strings.ToLower(taskID)
	lines[lineIndex] = cl.render()
	return strings.Join(lines, "\n"), true
}

// SetTaskChecked sets the check state of every checklist line marked with
// taskID. changed is false when no such line exists or every one already had
// the requested state, so callers can skip a pointless write.
func SetTaskChecked(content, taskID string, checked bool) (updated string, changed bool) {
	want := strings.ToLower(taskID)
	lines := strings.Split(content, "\n")
	inCode := codeLines(lines)
	for i, line := range lines {
		if inCode[i] {
			continue
		}
		cl, ok := parseChecklistLine(line)
		if !ok || cl.taskID != want || cl.checked() == checked {
			continue
		}
		if checked {
			cl.mark = "x"
		} else {
			cl.mark = " "
		}
		lines[i] = cl.render()
		changed = true
	}
	if !changed {
		return content, false
	}
	return strings.Join(lines, "\n"), true
}

// NewlyCheckedTaskIDs returns the ids of marked items that were unchecked in
// oldContent and are checked in newContent — i.e. only markers whose state
// actually flipped to checked. A marker that did not exist before (a freshly
// pasted line) is not reported.
func NewlyCheckedTaskIDs(oldContent, newContent string) []string {
	wasChecked := map[string]bool{}
	for _, it := range ParseChecklist(oldContent) {
		if it.TaskID == "" {
			continue
		}
		// A task id on several lines counts as checked only if all are.
		prev, seen := wasChecked[it.TaskID]
		if seen {
			wasChecked[it.TaskID] = prev && it.Checked
		} else {
			wasChecked[it.TaskID] = it.Checked
		}
	}
	ids := []string{}
	reported := map[string]bool{}
	for _, it := range ParseChecklist(newContent) {
		if it.TaskID == "" || !it.Checked || reported[it.TaskID] {
			continue
		}
		prev, existed := wasChecked[it.TaskID]
		if existed && !prev {
			reported[it.TaskID] = true
			ids = append(ids, it.TaskID)
		}
	}
	return ids
}
