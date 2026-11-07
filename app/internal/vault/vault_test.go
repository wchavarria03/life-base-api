package vault

import (
	"strings"
	"testing"
)

func TestValidateNotePath(t *testing.T) {
	long := strings.Repeat("a", MaxPathLen) + ".org"
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"simple org", "second-brain/foo.org", false},
		{"simple md", "notes/bar.md", false},
		{"uppercase ext", "notes/BAR.MD", false},
		{"unicode name", "notes/ñandú.org", false},
		{"empty", "", true},
		{"parent traversal", "../etc/passwd.org", true},
		{"embedded traversal", "a/../b.org", true},
		{"dot segment", "a/./b.org", true},
		{"absolute", "/etc/x.org", true},
		{"backslash", `a\b.org`, true},
		{"nul", "a\x00b.org", true},
		{"newline", "a\nb.org", true},
		{"windows drive", "C:foo.org", true},
		{"bad extension", "a/b.txt", true},
		{"pdf", "5-Health/report.pdf", true},
		{"no extension", "a/b", true},
		{"hidden dir", ".git/config.org", true},
		{"emacs lock", "a/.#b.org", true},
		{"empty segment", "a//b.org", true},
		{"trailing slash", "a/b.org/", true},
		{"too long", long, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateNotePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateNotePath(%q) err=%v wantErr=%v", tt.path, err, tt.wantErr)
			}
		})
	}
}

func TestValidateAttachmentPath(t *testing.T) {
	tests := []struct {
		path    string
		wantErr bool
	}{
		{"attachments/a.png", false},
		{"attachments/a.JPG", false},
		{"attachments/a.webp", false},
		{"attachments/a.svg", true},
		{"attachments/a.org", true},
		{"../a.png", true},
	}
	for _, tt := range tests {
		if err := ValidateAttachmentPath(tt.path); (err != nil) != tt.wantErr {
			t.Errorf("ValidateAttachmentPath(%q) err=%v wantErr=%v", tt.path, err, tt.wantErr)
		}
	}
}

func TestIsSensitivePath(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"5-Health/labs.org", true},
		{"5-health/labs.org", true},
		{"legacy-para/3-Resources/3-Finances/tax.org", true},
		{"legacy-para/3-Resources/other/x.org", false},
		{"second-brain/5-Health-notes.org", false},
		{"x/5-Health/a.org", false},
	}
	for _, tt := range tests {
		if got := IsSensitivePath(tt.path); got != tt.want {
			t.Errorf("IsSensitivePath(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestResolveSensitive(t *testing.T) {
	yes, no := true, false
	tests := []struct {
		name               string
		declared, pathRule bool
		override           *bool
		want               bool
	}{
		{"nothing", false, false, nil, false},
		{"path only", false, true, nil, true},
		{"override off beats path", false, true, &no, false},
		{"override on", false, false, &yes, true},
		{"declared beats override off", true, false, &no, true},
		{"declared and path", true, true, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ResolveSensitive(tt.declared, tt.pathRule, tt.override); got != tt.want {
				t.Fatalf("got %v want %v", got, tt.want)
			}
		})
	}
}

func TestParseOrgMetadata(t *testing.T) {
	src := ":PROPERTIES:\n:ID: abc-123\n:SENSITIVE: t\n:END:\n#+title: Synthetic Note\n#+filetags: :alpha:sensitive:\n\n* Heading :tag:\nSee [[id:def-456][Other note]] and [[id:def-456]].\n"
	doc := ParseOrg(src)
	if doc.Title != "Synthetic Note" {
		t.Errorf("title = %q", doc.Title)
	}
	if doc.ID != "abc-123" {
		t.Errorf("id = %q", doc.ID)
	}
	if !doc.Sensitive {
		t.Errorf("expected sensitive")
	}
	if len(doc.FileTags) != 2 || doc.FileTags[0] != "alpha" {
		t.Errorf("filetags = %v", doc.FileTags)
	}
	if len(doc.Links) != 1 || doc.Links[0].Target != "def-456" || doc.Links[0].Label != "Other note" {
		t.Errorf("links = %+v", doc.Links)
	}
}

func TestParseOrgSensitiveVariants(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want bool
	}{
		{"filetag", "#+title: x\n#+filetags: :sensitive:\n", true},
		{"filetag space separated", "#+filetags: a sensitive\n", true},
		{"property", ":PROPERTIES:\n:ID: x1\n:SENSITIVE: t\n:END:\n", true},
		{"property false", ":PROPERTIES:\n:ID: x1\n:SENSITIVE: nil\n:END:\n", false},
		{"heading property ignored", "* H\n:PROPERTIES:\n:SENSITIVE: t\n:END:\n", false},
		{"plain", "#+title: x\n", false},
		{"tag substring", "#+filetags: :insensitive:\n", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseOrg(tt.src).Sensitive; got != tt.want {
				t.Fatalf("got %v want %v", got, tt.want)
			}
		})
	}
}

func TestParseNoteMarkdown(t *testing.T) {
	m := ParseNote("md", "---\ntitle: \"Hello\"\nsensitive: true\nid: m-1\n---\n# Ignored\nbody", "a/b.md")
	if m.Title != "Hello" || !m.Sensitive || m.OrgID != "m-1" {
		t.Fatalf("meta = %+v", m)
	}
	m = ParseNote("md", "# First heading\ntext", "a/b.md")
	if m.Title != "First heading" || m.Sensitive {
		t.Fatalf("meta = %+v", m)
	}
	m = ParseNote("org", "no title here", "dir/my_file_name.org")
	if m.Title != "my file name" {
		t.Fatalf("fallback title = %q", m.Title)
	}
}

func TestRenderHTML(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		opts     RenderOptions
		contains []string
		absent   []string
	}{
		{
			name:     "escapes raw html",
			src:      "hello <script>alert(1)</script> & <img src=x onerror=alert(1)>",
			contains: []string{"&lt;script&gt;", "&amp;", "&lt;img"},
			absent:   []string{"<script", "<img"},
		},
		{
			name:     "html export block dropped",
			src:      "#+begin_export html\n<script>alert(1)</script>\n#+end_export\nafter",
			contains: []string{"<p>after</p>"},
			absent:   []string{"script", "alert"},
		},
		{
			name:     "property drawer hidden",
			src:      ":PROPERTIES:\n:ID: secret-id\n:END:\n#+title: T\n\nbody",
			contains: []string{"<p>body</p>"},
			absent:   []string{"secret-id", "PROPERTIES"},
		},
		{
			name:     "javascript link neutralized",
			src:      "[[javascript:alert(1)][click]]",
			contains: []string{"click"},
			absent:   []string{"href", "javascript"},
		},
		{
			name:     "https link",
			src:      "[[https://example.com/a?b=1&c=2][site]]",
			contains: []string{`href="https://example.com/a?b=1&amp;c=2"`, `rel="noopener noreferrer nofollow"`, ">site</a>"},
		},
		{
			name:     "quote in url cannot break attribute",
			src:      `[[https://example.com/"onmouseover="x][a]]`,
			contains: []string{"a"},
			absent:   []string{`onmouseover="x`, "<a "},
		},
		{
			name:     "id link app",
			src:      "[[id:abc-1][Other]]",
			contains: []string{`class="lb-note-link"`, `data-org-id="abc-1"`, ">Other</a>"},
		},
		{
			name:     "id link share is plain text",
			src:      "[[id:abc-1][Other]]",
			opts:     RenderOptions{Share: true},
			contains: []string{"Other"},
			absent:   []string{"<a ", "abc-1"},
		},
		{
			name:     "image is inert placeholder",
			src:      "[[file:attachments/pic.png][a pic]]",
			contains: []string{`<button type="button" class="lb-embed" data-kind="image" data-src="attachments/pic.png">a pic</button>`},
			absent:   []string{"<img"},
		},
		{
			name:     "attachment prefix",
			src:      "[[attachment:attachments/pic.jpg]]",
			contains: []string{`data-kind="image"`, `data-src="attachments/pic.jpg"`},
		},
		{
			name:   "image traversal rejected",
			src:    "[[file:../../etc/pic.png]]",
			absent: []string{"lb-embed", "<img"},
		},
		{
			name:   "image share is text",
			src:    "[[file:attachments/pic.png][a pic]]",
			opts:   RenderOptions{Share: true},
			absent: []string{"lb-embed", "<img", "<button"},
		},
		{
			name:     "youtube placeholder",
			src:      "[[https://www.youtube.com/watch?v=dQw4w9WgXcQ][clip]]",
			contains: []string{`data-kind="video"`, `data-src="https://www.youtube.com/watch?v=dQw4w9WgXcQ"`},
			absent:   []string{"<iframe", "<video"},
		},
		{
			name:     "mp4 placeholder",
			src:      "https://example.com/v/clip.mp4",
			contains: []string{`data-kind="video"`},
		},
		{
			name:     "video in share is plain link",
			src:      "[[https://youtu.be/dQw4w9WgXcQ][clip]]",
			opts:     RenderOptions{Share: true},
			contains: []string{"<a href="},
			absent:   []string{"lb-embed"},
		},
		{
			name:     "checklist",
			src:      "- [ ] open\n- [X] done\n  - nested item",
			contains: []string{`data-state="open"`, `data-state="done"`, "<ul><li>", "nested item"},
		},
		{
			name:     "src block escaped",
			src:      "#+begin_src html\n<b>x</b>\n#+end_src",
			contains: []string{`data-lang="html"`, "&lt;b&gt;x&lt;/b&gt;"},
			absent:   []string{"<b>"},
		},
		{
			name:     "emphasis",
			src:      "this is *bold* and /italic/ and =verb<x>= but a/b/c stays",
			contains: []string{"<strong>bold</strong>", "<em>italic</em>", "<code>verb&lt;x&gt;</code>", "a/b/c"},
		},
		{
			name:     "headings and table",
			src:      "* Top\n** Sub\n| a | b |\n|---+---|\n| 1 | 2 |",
			contains: []string{"<h1>Top</h1>", "<h2>Sub</h2>", "<th>a</th>", "<td>2</td>"},
		},
		{
			name:     "attribute injection through id",
			src:      `[[id:x" onclick="y][L]]`,
			absent:   []string{"onclick", "<a "},
			contains: []string{"L"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RenderOrg(tt.src, tt.opts)
			for _, want := range tt.contains {
				if !strings.Contains(got, want) {
					t.Errorf("output missing %q:\n%s", want, got)
				}
			}
			for _, bad := range tt.absent {
				if strings.Contains(got, bad) {
					t.Errorf("output contains forbidden %q:\n%s", bad, got)
				}
			}
		})
	}
}

func TestSafeHref(t *testing.T) {
	tests := []struct {
		href string
		want bool
	}{
		{"https://example.com", true},
		{"http://example.com/a?b=c", true},
		{"mailto:a@example.com", true},
		{"/relative/path", true},
		{"relative", true},
		{"javascript:alert(1)", false},
		{"JaVaScRiPt:alert(1)", false},
		{"data:text/html;base64,AAAA", false},
		{"vbscript:x", false},
		{"//evil.example", false},
		{"https://", false},
		{"https://a.b/ c", false},
		{"https://a.b/\"x", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := SafeHref(tt.href); got != tt.want {
			t.Errorf("SafeHref(%q) = %v, want %v", tt.href, got, tt.want)
		}
	}
}

func TestIsVideoURL(t *testing.T) {
	tests := []struct {
		u    string
		want bool
	}{
		{"https://www.youtube.com/watch?v=abc", true},
		{"https://youtu.be/abc", true},
		{"https://vimeo.com/123", true},
		{"https://example.com/a.mp4", true},
		{"http://youtu.be/abc", false},
		{"https://example.com/a.html", false},
		{"https://evilyoutube.com/x", false},
	}
	for _, tt := range tests {
		if got := IsVideoURL(tt.u); got != tt.want {
			t.Errorf("IsVideoURL(%q) = %v, want %v", tt.u, got, tt.want)
		}
	}
}
