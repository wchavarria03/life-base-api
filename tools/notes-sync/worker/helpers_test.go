package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestValidNotePath(t *testing.T) {
	tests := []struct {
		path    string
		wantErr bool
	}{
		{"second-brain/a.org", false},
		{"x/y/z.md", false},
		{"", true},
		{"../a.org", true},
		{"a/../b.org", true},
		{"/abs.org", true},
		{`a\b.org`, true},
		{"a\x00.org", true},
		{"a.txt", true},
		{"a.pdf", true},
		{".git/hooks/x.org", true},
		{"a//b.org", true},
		{"C:x.org", true},
		{strings.Repeat("a", 300) + ".org", true},
	}
	for _, tt := range tests {
		if err := validNotePath(tt.path); (err != nil) != tt.wantErr {
			t.Errorf("validNotePath(%q) err=%v wantErr=%v", tt.path, err, tt.wantErr)
		}
	}
}

func TestParseNameStatusZ(t *testing.T) {
	out := []byte("A\x00notes/new.org\x00M\x00notes/old.md\x00D\x00gone.org\x00")
	got := parseNameStatusZ(out)
	want := []change{{"A", "notes/new.org"}, {"M", "notes/old.md"}, {"D", "gone.org"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v want %+v", got, want)
	}
	if len(parseNameStatusZ(nil)) != 0 {
		t.Fatal("expected no changes for empty output")
	}
}

func TestFilterChanges(t *testing.T) {
	in := []change{
		{"A", "second-brain/a.org"},
		{"M", "second-brain/b.md"},
		{"D", "second-brain/c.org"},
		{"M", "5-Health/report.pdf"},
		{"A", "attachments/pic.png"},
		{"M", "org-roam.db"},
		{"A", ".DS_Store"},
		{"M", "../escape.org"},
		{"T", "typechange.org"},
	}
	keep, skipped := filterChanges(in)
	if len(keep) != 3 {
		t.Fatalf("keep = %+v", keep)
	}
	if len(skipped) != 6 {
		t.Fatalf("skipped = %v", skipped)
	}
}

func TestIsSyncableText(t *testing.T) {
	if !isSyncableText([]byte("héllo\n* heading")) {
		t.Error("utf-8 text should be syncable")
	}
	if isSyncableText([]byte("bin\x00ary")) {
		t.Error("NUL should be rejected")
	}
	if isSyncableText([]byte{0xff, 0xfe, 0xfd}) {
		t.Error("invalid utf-8 should be rejected")
	}
	if isSyncableText([]byte(strings.Repeat("a", maxContentSize+1))) {
		t.Error("oversize content should be rejected")
	}
}

func TestBatchItems(t *testing.T) {
	var items []upsertItem
	for i := 0; i < 5; i++ {
		items = append(items, upsertItem{VaultPath: "a.org", Content: strings.Repeat("x", 100)})
	}
	if got := batchItems(items, 2, 1<<20); len(got) != 3 || len(got[0]) != 2 || len(got[2]) != 1 {
		t.Errorf("by count: %v", got)
	}
	if got := batchItems(items, 100, 300); len(got) < 3 {
		t.Errorf("by bytes: got %d batches", len(got))
	}
	if got := batchItems(nil, 10, 10); len(got) != 0 {
		t.Errorf("empty: %v", got)
	}
}

func TestValidCommit(t *testing.T) {
	tests := map[string]bool{
		"abc1234": true, "0123456789abcdef0123456789abcdef01234567": true,
		"abc": false, "ABC1234": false, "xyz1234": false, "": false,
	}
	for in, want := range tests {
		if got := validCommit(in); got != want {
			t.Errorf("validCommit(%q) = %v want %v", in, got, want)
		}
	}
}

func TestSafeJoin(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "link")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if got, err := safeJoin(root, "notes/new.org"); err != nil || !strings.HasPrefix(got, root) {
		t.Errorf("plain path: %q %v", got, err)
	}
	for _, bad := range []string{"../x.org", "link/x.org", "/etc/passwd.org", "a.txt"} {
		if _, err := safeJoin(root, bad); err == nil {
			t.Errorf("safeJoin(%q) should fail", bad)
		}
	}
}

func TestSha256Hex(t *testing.T) {
	if got := sha256Hex(""); got != "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855" {
		t.Errorf("sha256 of empty = %s", got)
	}
}
