package services

import (
	"testing"

	"life-base-api/app/internal/models"
)

func TestGraphEdges(t *testing.T) {
	b := "b"
	links := []*models.GraphLinkRow{
		{FromNoteID: "a", ToNoteID: &b},
		{FromNoteID: "a", ToNoteID: &b},
		{FromNoteID: "a", ToOrgID: "org-c"},
		{FromNoteID: "b", ToOrgID: "x"},
		{FromNoteID: "a", ToOrgID: "org-a"},
		{FromNoteID: "gone", ToNoteID: &b},
	}
	byID := map[string]bool{"a": true, "b": true, "c": true}
	byOrg := map[string]string{"org-a": "a", "org-c": "c"}

	got := graphEdges(links, byID, byOrg)
	if len(got) != 2 {
		t.Fatalf("want 2 edges, got %d: %+v", len(got), got)
	}
	if got[0] != (models.GraphEdge{Source: "a", Target: "b"}) {
		t.Errorf("edge 0 = %+v", got[0])
	}
	if got[1] != (models.GraphEdge{Source: "a", Target: "c"}) {
		t.Errorf("edge 1 = %+v", got[1])
	}
}
