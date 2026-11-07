package vault

import (
	"html"
	"net/url"
	"path"
	"strconv"
	"strings"
)

// RenderOptions tunes RenderHTML.
type RenderOptions struct {
	// Share renders for a public share view: links to other notes become
	// plain text, image embeds become plain text (no scoped image URLs yet,
	// see docs/SHARING_HARDENING.md) and video links become normal anchors.
	Share bool
}

// maxInlineDepth bounds nested emphasis recursion.
const maxInlineDepth = 4

// emphasisWindow bounds how far ahead a closing emphasis marker is searched
// for, keeping inline parsing linear-ish on pathological input.
const emphasisWindow = 300

// RenderOrg parses src and renders sanitized HTML.
func RenderOrg(src string, opts RenderOptions) string {
	return RenderHTML(ParseOrg(src), opts)
}

// RenderHTML renders a parsed document. The output only ever contains tags
// written in this file: h1-h6, p, ul, ol, li, pre, code, blockquote, table,
// thead, tbody, tr, th, td, hr, strong, em, u, del, a, span, button.
// All text and attribute values pass through html.EscapeString.
func RenderHTML(doc *Doc, opts RenderOptions) string {
	r := &renderer{opts: opts}
	var b strings.Builder
	for _, blk := range doc.Blocks {
		switch blk.Kind {
		case BlockHeading:
			lvl := blk.Level
			if lvl > 6 {
				lvl = 6
			}
			tag := "h" + strconv.Itoa(lvl)
			b.WriteString("<" + tag + ">" + r.inline(blk.Text, 0) + "</" + tag + ">\n")
		case BlockParagraph:
			b.WriteString("<p>" + r.inline(blk.Text, 0) + "</p>\n")
		case BlockList:
			r.list(&b, blk.Items)
		case BlockSrc:
			b.WriteString(`<pre class="lb-src"`)
			if lang := safeLang(blk.Lang); lang != "" {
				b.WriteString(` data-lang="` + lang + `"`)
			}
			b.WriteString("><code>" + html.EscapeString(strings.Join(blk.Lines, "\n")) + "</code></pre>\n")
		case BlockExample:
			b.WriteString(`<pre class="lb-example"><code>` + html.EscapeString(strings.Join(blk.Lines, "\n")) + "</code></pre>\n")
		case BlockQuote:
			b.WriteString("<blockquote><p>" + r.inline(strings.Join(trimAll(blk.Lines), " "), 0) + "</p></blockquote>\n")
		case BlockTable:
			r.table(&b, blk.Lines)
		case BlockRule:
			b.WriteString("<hr>\n")
		}
	}
	return b.String()
}

type renderer struct {
	opts RenderOptions
}

func trimAll(lines []string) []string {
	out := make([]string, 0, len(lines))
	for _, l := range lines {
		if t := strings.TrimSpace(l); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// safeLang keeps only characters that are inert inside an attribute.
func safeLang(l string) string {
	l = strings.ToLower(l)
	if len(l) > 20 {
		return ""
	}
	for _, c := range l {
		ok := (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '+' || c == '#' || c == '-' || c == '_'
		if !ok {
			return ""
		}
	}
	return l
}

type listLevel struct {
	indent int
	tag    string
}

func (r *renderer) list(b *strings.Builder, items []ListItem) {
	var stack []listLevel
	for _, it := range items {
		tag := "ul"
		if it.Ordered {
			tag = "ol"
		}
		for len(stack) > 0 && it.Indent < stack[len(stack)-1].indent {
			b.WriteString("</li></" + stack[len(stack)-1].tag + ">")
			stack = stack[:len(stack)-1]
		}
		if len(stack) > 0 && it.Indent == stack[len(stack)-1].indent {
			b.WriteString("</li>")
		} else {
			b.WriteString("<" + tag + ">")
			stack = append(stack, listLevel{indent: it.Indent, tag: tag})
		}
		b.WriteString("<li>")
		switch it.Checkbox {
		case "X":
			b.WriteString(`<span class="lb-check" data-state="done">&#9745;</span> `)
		case " ":
			b.WriteString(`<span class="lb-check" data-state="open">&#9744;</span> `)
		case "-":
			b.WriteString(`<span class="lb-check" data-state="partial">&#9680;</span> `)
		}
		b.WriteString(r.inline(it.Text, 0))
	}
	for len(stack) > 0 {
		b.WriteString("</li></" + stack[len(stack)-1].tag + ">")
		stack = stack[:len(stack)-1]
	}
	b.WriteString("\n")
}

func (r *renderer) table(b *strings.Builder, rows []string) {
	isSep := func(row string) bool {
		return strings.HasPrefix(row, "|-") || strings.HasPrefix(row, "|+")
	}
	b.WriteString("<table>")
	header := len(rows) > 1 && isSep(rows[1])
	for idx, row := range rows {
		if isSep(row) {
			continue
		}
		cell := "td"
		if header && idx == 0 {
			cell = "th"
			b.WriteString("<thead>")
		} else if header && idx == 2 {
			b.WriteString("<tbody>")
		}
		inner := strings.TrimSuffix(strings.TrimPrefix(row, "|"), "|")
		b.WriteString("<tr>")
		for _, c := range strings.Split(inner, "|") {
			b.WriteString("<" + cell + ">" + r.inline(strings.TrimSpace(c), 0) + "</" + cell + ">")
		}
		b.WriteString("</tr>")
		if header && idx == 0 {
			b.WriteString("</thead>")
		}
	}
	if header && len(rows) > 2 {
		b.WriteString("</tbody>")
	}
	b.WriteString("</table>\n")
}

func isSpaceByte(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

func emphasisTag(c byte) string {
	switch c {
	case '*':
		return "strong"
	case '/':
		return "em"
	case '_':
		return "u"
	case '+':
		return "del"
	case '=', '~':
		return "code"
	}
	return ""
}

// emphasisEnd returns the index of the closing marker for an emphasis
// opening at s[i], or -1. It applies org's boundary rules so text such as
// "a/b/c" or "2*3*4" is left alone.
func emphasisEnd(s string, i int) int {
	m := s[i]
	if i > 0 && !strings.ContainsRune(" \t\n({'\"-", rune(s[i-1])) {
		return -1
	}
	if i+2 >= len(s) || isSpaceByte(s[i+1]) || s[i+1] == m {
		return -1
	}
	limit := i + emphasisWindow
	if limit > len(s) {
		limit = len(s)
	}
	for j := i + 2; j < limit; j++ {
		if s[j] != m || isSpaceByte(s[j-1]) {
			continue
		}
		if j+1 == len(s) || strings.ContainsRune(" \t\n-.,;:!?')}\"", rune(s[j+1])) {
			return j
		}
	}
	return -1
}

func (r *renderer) inline(s string, depth int) string {
	var b strings.Builder
	plainStart := 0
	flush := func(upTo int) {
		if upTo > plainStart {
			b.WriteString(html.EscapeString(s[plainStart:upTo]))
		}
	}
	i := 0
	for i < len(s) {
		c := s[i]
		switch {
		case c == '[' && strings.HasPrefix(s[i:], "[["):
			if target, label, end, ok := parseBracketLink(s, i); ok {
				flush(i)
				b.WriteString(r.link(target, label))
				i = end
				plainStart = i
				continue
			}
		case c == 'h' && (strings.HasPrefix(s[i:], "http://") || strings.HasPrefix(s[i:], "https://")) &&
			(i == 0 || strings.ContainsRune(" \t\n({\"'", rune(s[i-1]))):
			end := urlEnd(s, i)
			flush(i)
			b.WriteString(r.link(s[i:end], ""))
			i = end
			plainStart = i
			continue
		case emphasisTag(c) != "" && depth < maxInlineDepth:
			if end := emphasisEnd(s, i); end > 0 {
				flush(i)
				tag := emphasisTag(c)
				inner := s[i+1 : end]
				if c == '=' || c == '~' {
					b.WriteString("<" + tag + ">" + html.EscapeString(inner) + "</" + tag + ">")
				} else {
					b.WriteString("<" + tag + ">" + r.inline(inner, depth+1) + "</" + tag + ">")
				}
				i = end + 1
				plainStart = i
				continue
			}
		}
		i++
	}
	flush(len(s))
	return b.String()
}

// urlEnd finds the end of a bare URL starting at s[i], trimming trailing
// punctuation that is almost certainly sentence punctuation.
func urlEnd(s string, i int) int {
	j := i
	for j < len(s) && !isSpaceByte(s[j]) && s[j] != '<' && s[j] != '>' && s[j] != '"' {
		j++
	}
	for j > i && strings.ContainsRune(".,;:!?)'", rune(s[j-1])) {
		j--
	}
	return j
}

// SafeHref reports whether u may be used as an href: http(s) with a host,
// mailto, or a plain relative reference. javascript:, data:, vbscript:,
// protocol-relative URLs and anything with control characters are rejected.
func SafeHref(u string) bool {
	if u == "" || len(u) > 2048 {
		return false
	}
	for _, c := range u {
		if c < 0x21 || c == 0x7f || c == '\\' || c == '<' || c == '>' || c == '"' {
			return false
		}
	}
	parsed, err := url.Parse(u)
	if err != nil {
		return false
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
		return parsed.Host != ""
	case "mailto":
		return parsed.Opaque != ""
	case "":
		return parsed.Host == "" && !strings.HasPrefix(u, "//")
	}
	return false
}

var videoHosts = map[string]bool{
	"youtube.com": true, "www.youtube.com": true, "m.youtube.com": true,
	"youtu.be": true, "vimeo.com": true, "www.vimeo.com": true,
}

// IsVideoURL reports whether u is an https link the UI may offer as an
// on-demand video embed (YouTube, Vimeo, or a direct .mp4).
func IsVideoURL(u string) bool {
	parsed, err := url.Parse(u)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return false
	}
	if videoHosts[strings.ToLower(parsed.Hostname())] {
		return true
	}
	return strings.HasSuffix(strings.ToLower(parsed.Path), ".mp4")
}

// attachmentPath normalises an org file:/attachment: target to a
// vault-relative image path, or returns "" if it is not a valid image.
func attachmentPath(target string) string {
	t := target
	switch {
	case strings.HasPrefix(t, "file:"):
		t = strings.TrimPrefix(t, "file:")
	case strings.HasPrefix(t, "attachment:"):
		t = strings.TrimPrefix(t, "attachment:")
	default:
		return ""
	}
	if idx := strings.Index(t, "::"); idx >= 0 {
		t = t[:idx]
	}
	t = strings.TrimPrefix(t, "./")
	if ValidateAttachmentPath(t) != nil {
		return ""
	}
	return t
}

func (r *renderer) link(target, label string) string {
	target = strings.TrimSpace(target)
	text := label
	switch {
	case strings.HasPrefix(target, "id:"):
		id := target[3:]
		if text == "" {
			text = "note"
		}
		if r.opts.Share || !ValidOrgID(id) {
			return html.EscapeString(text)
		}
		return `<a class="lb-note-link" href="/notes?org_id=` + html.EscapeString(id) +
			`" data-org-id="` + html.EscapeString(id) + `">` + html.EscapeString(text) + `</a>`

	case strings.HasPrefix(target, "file:") || strings.HasPrefix(target, "attachment:"):
		p := attachmentPath(target)
		if text == "" {
			text = path.Base(strings.TrimPrefix(strings.TrimPrefix(target, "file:"), "attachment:"))
		}
		if p == "" || r.opts.Share {
			return html.EscapeString(text)
		}
		// Inert placeholder: the frontend creates the <img> only on click.
		return `<button type="button" class="lb-embed" data-kind="image" data-src="` + html.EscapeString(p) +
			`">` + html.EscapeString(text) + `</button>`

	case strings.HasPrefix(target, "http://"), strings.HasPrefix(target, "https://"), strings.HasPrefix(target, "mailto:"):
		if !SafeHref(target) {
			if text == "" {
				text = target
			}
			return html.EscapeString(text)
		}
		if text == "" {
			text = target
		}
		if !r.opts.Share && IsVideoURL(target) {
			return `<button type="button" class="lb-embed" data-kind="video" data-src="` + html.EscapeString(target) +
				`">` + html.EscapeString(text) + `</button>`
		}
		return `<a href="` + html.EscapeString(target) + `" target="_blank" rel="noopener noreferrer nofollow">` +
			html.EscapeString(text) + `</a>`
	}
	// Anything else (heading links, custom ids, other file links, unknown
	// schemes such as javascript:) is rendered as inert text.
	if text == "" {
		text = target
	}
	return html.EscapeString(text)
}
