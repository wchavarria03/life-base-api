package services

import (
	"reflect"
	"testing"
)

const (
	testTaskA = "0b1c2d3e-1111-4222-8333-444455556666"
	testTaskB = "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee"
)

func TestParseChecklist(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []ChecklistItem
	}{
		{"empty", "", []ChecklistItem{}},
		{"no checklist", "hello\n- bullet\nworld", []ChecklistItem{}},
		{
			"markdown and org marks",
			"- [ ] one\n- [x] two\n- [X] three",
			[]ChecklistItem{
				{LineIndex: 0, Text: "one", Checked: false},
				{LineIndex: 1, Text: "two", Checked: true},
				{LineIndex: 2, Text: "three", Checked: true},
			},
		},
		{
			"marker is stripped from text",
			"intro\n- [ ] Buy filter {task:" + testTaskA + "}",
			[]ChecklistItem{{LineIndex: 1, Text: "Buy filter", Checked: false, TaskID: testTaskA}},
		},
		{
			"indented, plus and ordered bullets",
			"  - [ ] a\n+ [x] b\n1. [ ] c",
			[]ChecklistItem{
				{LineIndex: 0, Text: "a"},
				{LineIndex: 1, Text: "b", Checked: true},
				{LineIndex: 2, Text: "c"},
			},
		},
		{
			"fenced code ignored",
			"- [ ] real\n```\n- [ ] code\n```\n~~~\n- [x] tilde\n~~~\n- [ ] after",
			[]ChecklistItem{
				{LineIndex: 0, Text: "real"},
				{LineIndex: 7, Text: "after"},
			},
		},
		{
			"unterminated fence runs to end",
			"- [ ] a\n```\n- [ ] b",
			[]ChecklistItem{{LineIndex: 0, Text: "a"}},
		},
		{
			"crlf",
			"- [x] done\r\n- [ ] todo\r\n",
			[]ChecklistItem{
				{LineIndex: 0, Text: "done", Checked: true},
				{LineIndex: 1, Text: "todo"},
			},
		},
		{"not a checkbox", "- [] nope\n- [y] nope\n[ ] nope", []ChecklistItem{}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ParseChecklist(tc.content)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("ParseChecklist() = %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestAttachTaskMarker(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		line       int
		want       string
		wantOK     bool
		idempotent bool
	}{
		{"plain item", "a\n- [ ] Buy filter\nb", 1, "a\n- [ ] Buy filter {task:" + testTaskA + "}\nb", true, true},
		{"checked item keeps mark", "- [X] done", 0, "- [X] done {task:" + testTaskA + "}", true, true},
		{"replaces other marker", "- [ ] x {task:" + testTaskB + "}", 0, "- [ ] x {task:" + testTaskA + "}", true, true},
		{"preserves crlf", "- [ ] x\r\n- [ ] y\r\n", 0, "- [ ] x {task:" + testTaskA + "}\r\n- [ ] y\r\n", true, true},
		{"non checklist line", "just text", 0, "just text", false, false},
		{"out of range", "- [ ] a", 5, "- [ ] a", false, false},
		{"negative", "- [ ] a", -1, "- [ ] a", false, false},
		{"inside code fence", "```\n- [ ] a\n```", 1, "```\n- [ ] a\n```", false, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := AttachTaskMarker(tc.content, tc.line, testTaskA)
			if ok != tc.wantOK || got != tc.want {
				t.Fatalf("AttachTaskMarker() = (%q, %v), want (%q, %v)", got, ok, tc.want, tc.wantOK)
			}
			if tc.idempotent {
				again, ok2 := AttachTaskMarker(got, tc.line, testTaskA)
				if !ok2 || again != got {
					t.Fatalf("second AttachTaskMarker() = (%q, %v), want unchanged %q", again, ok2, got)
				}
			}
		})
	}
}

func TestSetTaskChecked(t *testing.T) {
	marked := "- [ ] a {task:" + testTaskA + "}\n- [ ] b {task:" + testTaskB + "}"
	tests := []struct {
		name        string
		content     string
		taskID      string
		checked     bool
		want        string
		wantChanged bool
	}{
		{"check one", marked, testTaskA, true, "- [x] a {task:" + testTaskA + "}\n- [ ] b {task:" + testTaskB + "}", true},
		{"already checked", "- [X] a {task:" + testTaskA + "}", testTaskA, true, "- [X] a {task:" + testTaskA + "}", false},
		{"uncheck org mark", "- [X] a {task:" + testTaskA + "}", testTaskA, false, "- [ ] a {task:" + testTaskA + "}", true},
		{"unknown task", marked, "00000000-0000-4000-8000-000000000000", true, marked, false},
		{"uppercase id matches", marked, "0B1C2D3E-1111-4222-8333-444455556666", true, "- [x] a {task:" + testTaskA + "}\n- [ ] b {task:" + testTaskB + "}", true},
		{"ignored in code fence", "```\n- [ ] a {task:" + testTaskA + "}\n```", testTaskA, true, "```\n- [ ] a {task:" + testTaskA + "}\n```", false},
		{"same id on two lines", "- [ ] a {task:" + testTaskA + "}\n- [ ] a2 {task:" + testTaskA + "}", testTaskA, true, "- [x] a {task:" + testTaskA + "}\n- [x] a2 {task:" + testTaskA + "}", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, changed := SetTaskChecked(tc.content, tc.taskID, tc.checked)
			if got != tc.want || changed != tc.wantChanged {
				t.Fatalf("SetTaskChecked() = (%q, %v), want (%q, %v)", got, changed, tc.want, tc.wantChanged)
			}
			again, changed2 := SetTaskChecked(got, tc.taskID, tc.checked)
			if changed2 || again != got {
				t.Fatalf("SetTaskChecked not idempotent: (%q, %v)", again, changed2)
			}
		})
	}
}

func TestFindTaskIDs(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []string
	}{
		{"none", "- [ ] a", []string{}},
		{"two distinct", "- [ ] a {task:" + testTaskA + "}\n- [x] b {task:" + testTaskB + "}", []string{testTaskA, testTaskB}},
		{"dedupes", "- [ ] a {task:" + testTaskA + "}\n- [x] b {task:" + testTaskA + "}", []string{testTaskA}},
		{"marker on non checklist line ignored", "text {task:" + testTaskA + "}", []string{}},
		{"code fence ignored", "```\n- [ ] a {task:" + testTaskA + "}\n```", []string{}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := FindTaskIDs(tc.content)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("FindTaskIDs() = %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestNewlyCheckedTaskIDs(t *testing.T) {
	open := "- [ ] a {task:" + testTaskA + "}\n- [ ] b {task:" + testTaskB + "}"
	tests := []struct {
		name string
		old  string
		new  string
		want []string
	}{
		{"tick one", open, "- [x] a {task:" + testTaskA + "}\n- [ ] b {task:" + testTaskB + "}", []string{testTaskA}},
		{"tick both", open, "- [x] a {task:" + testTaskA + "}\n- [X] b {task:" + testTaskB + "}", []string{testTaskA, testTaskB}},
		{"no change", open, open, []string{}},
		{"already checked before", "- [x] a {task:" + testTaskA + "}", "- [x] a {task:" + testTaskA + "}", []string{}},
		{"unticked", "- [x] a {task:" + testTaskA + "}", open, []string{}},
		{"new line pasted checked", "", "- [x] a {task:" + testTaskA + "}", []string{}},
		{"line moved and ticked", open, "intro\n- [ ] b {task:" + testTaskB + "}\n- [x] a {task:" + testTaskA + "}", []string{testTaskA}},
		{"unmarked tick ignored", "- [ ] z", "- [x] z", []string{}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := NewlyCheckedTaskIDs(tc.old, tc.new)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("NewlyCheckedTaskIDs() = %#v, want %#v", got, tc.want)
			}
		})
	}
}
