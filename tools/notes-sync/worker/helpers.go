package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// This file holds the pure helpers (no I/O beyond path resolution) so they
// can be unit tested. The path rules deliberately mirror
// app/internal/vault/paths.go: the API validates again, this is the
// worker's own first line of defense (it also decides what gets written to
// disk, which the API cannot see).

const (
	maxPathLen     = 240
	maxContentSize = 1 << 20 // keep in sync with vaultMaxContentBytes in the API
)

var noteExts = map[string]bool{".org": true, ".md": true}

// validNotePath rejects anything that is not a plain, relative .org/.md path.
func validNotePath(p string) error {
	if err := validPathShape(p); err != nil {
		return err
	}
	for _, seg := range strings.Split(p, "/") {
		if seg == "" || seg == "." || seg == ".." {
			return fmt.Errorf("bad segment %q", seg)
		}
		if strings.HasPrefix(seg, ".") {
			return fmt.Errorf("hidden segment %q", seg)
		}
	}
	if !noteExts[strings.ToLower(path.Ext(p))] {
		return errors.New("extension not allowed")
	}
	return nil
}

// validPathShape rejects empty, oversized, non-UTF-8, absolute and otherwise
// malformed path strings.
func validPathShape(p string) error {
	switch {
	case p == "":
		return errors.New("empty path")
	case len(p) > maxPathLen:
		return errors.New("path too long")
	case !utf8.ValidString(p):
		return errors.New("path is not valid UTF-8")
	case strings.ContainsRune(p, 0):
		return errors.New("NUL in path")
	case strings.Contains(p, "\\"):
		return errors.New("backslash in path")
	case strings.HasPrefix(p, "/"):
		return errors.New("absolute path")
	case len(p) >= 2 && p[1] == ':':
		return errors.New("drive prefix")
	}
	for _, r := range p {
		if r < 0x20 || r == 0x7f {
			return errors.New("control character in path")
		}
	}
	return nil
}

// change is one entry of `git diff --name-status`.
type change struct {
	Status string // "A", "M" or "D"
	Path   string
}

// parseNameStatusZ parses `git diff --name-status -z --no-renames` output:
// repeated  STATUS NUL PATH NUL.
func parseNameStatusZ(out []byte) []change {
	parts := strings.Split(string(out), "\x00")
	var changes []change
	for i := 0; i+1 < len(parts); i += 2 {
		status := strings.TrimSpace(parts[i])
		if status == "" {
			continue
		}
		changes = append(changes, change{Status: status[:1], Path: parts[i+1]})
	}
	return changes
}

// parseLsFilesZ parses `git ls-files -z` into "added" changes (full resync).
func parseLsFilesZ(out []byte) []change {
	var changes []change
	for _, p := range strings.Split(string(out), "\x00") {
		if p != "" {
			changes = append(changes, change{Status: "A", Path: p})
		}
	}
	return changes
}

// filterChanges keeps only changes the worker may sync: valid note paths with
// status A/M/D. Everything else (PDFs, images, dotfiles, org-roam.db, ...)
// is dropped and returned in skipped for logging.
func filterChanges(in []change) (keep []change, skipped []string) {
	for _, c := range in {
		switch c.Status {
		case "A", "M", "D":
		default:
			skipped = append(skipped, c.Path)
			continue
		}
		if err := validNotePath(c.Path); err != nil {
			skipped = append(skipped, c.Path)
			continue
		}
		keep = append(keep, c)
	}
	return keep, skipped
}

// isSyncableText reports whether file content may be sent to the API:
// valid UTF-8, no NUL bytes, within the size cap.
func isSyncableText(b []byte) bool {
	if len(b) > maxContentSize {
		return false
	}
	if !utf8.Valid(b) {
		return false
	}
	for _, c := range b {
		if c == 0 {
			return false
		}
	}
	return true
}

type upsertItem struct {
	VaultPath     string `json:"vault_path"`
	ContentFormat string `json:"content_format,omitempty"`
	Content       string `json:"content"`
	Commit        string `json:"commit"`
	Deleted       bool   `json:"deleted,omitempty"`
}

// formatFor returns "org" or "md" for a validated path.
func formatFor(p string) string {
	if strings.ToLower(path.Ext(p)) == ".org" {
		return "org"
	}
	return "md"
}

// batchItems splits items so no batch exceeds maxItems or roughly maxBytes
// of content.
func batchItems(items []upsertItem, maxItems, maxBytes int) [][]upsertItem {
	var batches [][]upsertItem
	var cur []upsertItem
	size := 0
	for _, it := range items {
		n := len(it.Content) + len(it.VaultPath) + 128
		if len(cur) > 0 && (len(cur) >= maxItems || size+n > maxBytes) {
			batches = append(batches, cur)
			cur, size = nil, 0
		}
		cur = append(cur, it)
		size += n
	}
	if len(cur) > 0 {
		batches = append(batches, cur)
	}
	return batches
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// safeJoin resolves a validated vault-relative path under root and makes
// sure neither the lexical result nor any existing parent (after resolving
// symlinks) escapes root.
func safeJoin(root, rel string) (string, error) {
	if err := validNotePath(rel); err != nil {
		return "", err
	}
	full := filepath.Join(root, filepath.FromSlash(rel))
	lexical, err := filepath.Rel(root, full)
	if err != nil || lexical == ".." || strings.HasPrefix(lexical, ".."+string(filepath.Separator)) {
		return "", errors.New("path escapes repository")
	}
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	// Walk up to the nearest existing ancestor and resolve it.
	dir := filepath.Dir(full)
	for {
		if _, statErr := os.Lstat(dir); statErr == nil {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("no existing ancestor")
		}
		dir = parent
	}
	realDir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return "", err
	}
	if realDir != realRoot && !strings.HasPrefix(realDir, realRoot+string(filepath.Separator)) {
		return "", errors.New("path escapes repository via symlink")
	}
	if fi, err := os.Lstat(full); err == nil && fi.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("refusing to write through a symlink")
	}
	return full, nil
}

// validCommit matches an abbreviated or full lowercase hex commit id.
func validCommit(c string) bool {
	if len(c) < 7 || len(c) > 64 {
		return false
	}
	for _, r := range c {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}
