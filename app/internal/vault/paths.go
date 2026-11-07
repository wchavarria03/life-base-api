// Package vault parses and renders the user's notes vault (org-roam .org and
// plain .md files kept in git) and enforces the path and sensitivity rules
// the sync API relies on. It is dependency-free on purpose: everything the
// renderer emits is built from HTML-escaped text plus a fixed set of tags
// written in this package, so there is no raw-HTML pass-through to sanitize.
// It can later be swapped for go-org + bluemonday (see docs/NOTES_SYNC.md).
package vault

import (
	"errors"
	"fmt"
	"path"
	"strings"
	"unicode/utf8"
)

// MaxPathLen is the longest vault-relative path the API accepts.
const MaxPathLen = 240

// SensitivePrefixes are vault folders whose notes are sensitive by default.
// Compared case-insensitively (the Macs use a case-insensitive filesystem).
var SensitivePrefixes = []string{
	"5-Health/",
	"legacy-para/3-Resources/3-Finances/",
}

var noteExts = map[string]bool{".org": true, ".md": true}

var attachmentExts = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".webp": true,
}

// ErrInvalidPath is wrapped by every path validation failure.
var ErrInvalidPath = errors.New("invalid vault path")

// ValidateNotePath checks a vault-relative path for a note file (.org/.md).
func ValidateNotePath(p string) error {
	return validatePath(p, noteExts)
}

// ValidateAttachmentPath checks a vault-relative path for an image attachment.
func ValidateAttachmentPath(p string) error {
	return validatePath(p, attachmentExts)
}

func validatePath(p string, exts map[string]bool) error {
	if err := validatePathShape(p); err != nil {
		return err
	}
	if err := validatePathSegments(p); err != nil {
		return err
	}
	if !exts[strings.ToLower(path.Ext(p))] {
		return fmt.Errorf("%w: extension not allowed", ErrInvalidPath)
	}
	return nil
}

// validatePathShape rejects empty, oversized, non-UTF-8, absolute and
// otherwise malformed path strings.
func validatePathShape(p string) error {
	if p == "" {
		return fmt.Errorf("%w: empty", ErrInvalidPath)
	}
	if len(p) > MaxPathLen {
		return fmt.Errorf("%w: longer than %d bytes", ErrInvalidPath, MaxPathLen)
	}
	if !utf8.ValidString(p) {
		return fmt.Errorf("%w: not valid UTF-8", ErrInvalidPath)
	}
	if strings.ContainsRune(p, 0) {
		return fmt.Errorf("%w: NUL byte", ErrInvalidPath)
	}
	if strings.Contains(p, "\\") {
		return fmt.Errorf("%w: backslash", ErrInvalidPath)
	}
	for _, r := range p {
		if r < 0x20 || r == 0x7f {
			return fmt.Errorf("%w: control character", ErrInvalidPath)
		}
	}
	if strings.HasPrefix(p, "/") {
		return fmt.Errorf("%w: absolute path", ErrInvalidPath)
	}
	// Windows drive letters ("C:foo") are never legitimate vault paths.
	if len(p) >= 2 && p[1] == ':' {
		return fmt.Errorf("%w: drive prefix", ErrInvalidPath)
	}
	return nil
}

// validatePathSegments rejects empty, relative and hidden path segments.
func validatePathSegments(p string) error {
	for _, seg := range strings.Split(p, "/") {
		switch {
		case seg == "":
			return fmt.Errorf("%w: empty segment", ErrInvalidPath)
		case seg == "." || seg == "..":
			return fmt.Errorf("%w: relative segment %q", ErrInvalidPath, seg)
		case strings.HasPrefix(seg, "."):
			// Covers .git, .claude, emacs lock files (.#foo.org) and dotfiles.
			return fmt.Errorf("%w: hidden segment %q", ErrInvalidPath, seg)
		}
	}
	return nil
}

// FormatForPath returns "org" or "md" for a validated note path.
func FormatForPath(p string) string {
	if strings.ToLower(path.Ext(p)) == ".org" {
		return "org"
	}
	return "md"
}

// IsSensitivePath reports whether p lives under a sensitive-by-default folder.
func IsSensitivePath(p string) bool {
	lp := strings.ToLower(p)
	for _, prefix := range SensitivePrefixes {
		if strings.HasPrefix(lp, strings.ToLower(prefix)) {
			return true
		}
	}
	return false
}

// ResolveSensitive combines the three inputs into the stored flag:
//   - a flag declared inside the file always wins and cannot be turned off
//     from the app (the file is the source of truth);
//   - otherwise an explicit in-app override (non-nil) wins over the path rule;
//   - otherwise the path rule decides.
func ResolveSensitive(fileDeclared, pathSensitive bool, override *bool) bool {
	if fileDeclared {
		return true
	}
	if override != nil {
		return *override
	}
	return pathSensitive
}
