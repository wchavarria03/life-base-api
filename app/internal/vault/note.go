package vault

import (
	"path"
	"strings"
)

// Meta is what the sync API needs to know about one vault file.
type Meta struct {
	Title string
	OrgID string
	// Sensitive is true only when the file itself declares it (org filetag or
	// :SENSITIVE: property, markdown front-matter). Path rules are separate.
	Sensitive bool
	Links     []Link
}

// ParseNote extracts metadata from a note's content. relPath is the
// vault-relative path, used only for the fallback title.
func ParseNote(format, content, relPath string) Meta {
	var m Meta
	if format == "org" {
		doc := ParseOrg(content)
		m = Meta{Title: doc.Title, OrgID: doc.ID, Sensitive: doc.Sensitive, Links: doc.Links}
	} else {
		m = parseMarkdownMeta(content)
	}
	m.Title = strings.TrimSpace(m.Title)
	if m.Title == "" {
		base := path.Base(relPath)
		base = strings.TrimSuffix(base, path.Ext(base))
		m.Title = strings.ReplaceAll(base, "_", " ")
	}
	if len(m.Title) > 300 {
		m.Title = m.Title[:300]
		// Do not leave a truncated multi-byte rune behind.
		m.Title = strings.ToValidUTF8(m.Title, "")
	}
	return m
}

// parseMarkdownMeta reads a leading "---" front-matter block (flat key: value
// pairs only) for title / id / sensitive, and falls back to the first
// "# heading" for the title.
func parseMarkdownMeta(content string) Meta {
	var m Meta
	content = strings.ReplaceAll(content, "\r\n", "\n")
	lines := strings.Split(content, "\n")
	start := 0
	if len(lines) > 0 && strings.TrimSpace(lines[0]) == "---" {
		for j := 1; j < len(lines); j++ {
			t := strings.TrimSpace(lines[j])
			if t == "---" {
				start = j + 1
				break
			}
			colon := strings.Index(t, ":")
			if colon < 0 {
				continue
			}
			key := strings.ToLower(strings.TrimSpace(t[:colon]))
			val := strings.Trim(strings.TrimSpace(t[colon+1:]), `"'`)
			switch key {
			case "title":
				m.Title = val
			case "id":
				if ValidOrgID(val) {
					m.OrgID = val
				}
			case "sensitive":
				switch strings.ToLower(val) {
				case "true", "yes", "t", "1":
					m.Sensitive = true
				}
			}
		}
	}
	if m.Title == "" {
		for _, l := range lines[start:] {
			if strings.HasPrefix(l, "# ") {
				m.Title = strings.TrimSpace(l[2:])
				break
			}
		}
	}
	return m
}
