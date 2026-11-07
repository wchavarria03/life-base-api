package vault

import (
	"regexp"
	"strings"
)

// BlockKind identifies the kind of a parsed block.
type BlockKind int

// Block kinds produced by ParseOrg.
const (
	BlockParagraph BlockKind = iota
	BlockHeading
	BlockList
	BlockSrc
	BlockExample
	BlockQuote
	BlockTable
	BlockRule
)

// ListItem is one list entry; nesting is expressed through Indent.
type ListItem struct {
	Indent   int
	Ordered  bool
	Checkbox string // "", " " (open), "X" (done) or "-" (partial)
	Text     string
}

// Block is one parsed top-level element of an org document.
type Block struct {
	Kind  BlockKind
	Level int      // heading level (number of stars)
	Text  string   // heading title or paragraph text
	Lang  string   // src block language
	Lines []string // src/example/quote/table raw lines
	Items []ListItem
}

// Link is an org-roam style [[id:...][label]] link found in the body.
type Link struct {
	Target string // the org id (without the "id:" prefix)
	Label  string
}

// Doc is the parse result for one org file.
type Doc struct {
	Title     string
	FileTags  []string
	ID        string // file-level :ID: property
	Sensitive bool   // declared inside the file
	Blocks    []Block
	Links     []Link
}

var (
	orgIDRe       = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)
	headingRe     = regexp.MustCompile(`^(\*+)\s+(.*)$`)
	headingTagsRe = regexp.MustCompile(`\s+:[\w@#%:]+:\s*$`)
	listItemRe    = regexp.MustCompile(`^(\s*)([-+*]|[0-9]+[.)])\s+(.*)$`)
	ruleRe        = regexp.MustCompile(`^\s*-{5,}\s*$`)
	drawerRe      = regexp.MustCompile(`^:[A-Za-z0-9_-]+:$`)
	checkboxRe    = regexp.MustCompile(`^\[([ Xx-])\]\s+(.*)$`)
)

// ValidOrgID reports whether s is a safe org id (it is used in query filters
// and HTML attributes, so the alphabet is deliberately narrow).
func ValidOrgID(s string) bool {
	return orgIDRe.MatchString(s)
}

// ParseOrg parses an org-mode document. It never fails: unknown constructs
// are skipped or kept as paragraph text.
func ParseOrg(src string) *Doc {
	src = strings.ReplaceAll(src, "\r\n", "\n")
	src = strings.ReplaceAll(src, "\r", "\n")
	lines := strings.Split(src, "\n")

	doc := &Doc{}
	seenHeading := false
	var para []string

	flushPara := func() {
		if len(para) > 0 {
			doc.Blocks = append(doc.Blocks, Block{Kind: BlockParagraph, Text: strings.Join(para, " ")})
			para = nil
		}
	}

	i := 0
	for i < len(lines) {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		lower := strings.ToLower(trimmed)

		switch {
		case trimmed == "":
			flushPara()
			i++

		case strings.HasPrefix(lower, "#+begin_"):
			flushPara()
			// The "#+begin_" prefix is ASCII, so slicing trimmed by its length
			// is safe even if the rest of the line has multi-byte characters.
			name := trimmed[len("#+begin_"):]
			kind := strings.ToLower(name)
			rest := ""
			if sp := strings.IndexAny(name, " \t"); sp >= 0 {
				kind = strings.ToLower(name[:sp])
				rest = strings.TrimSpace(name[sp:])
			}
			j := i + 1
			var body []string
			for j < len(lines) && !strings.HasPrefix(strings.ToLower(strings.TrimSpace(lines[j])), "#+end_") {
				body = append(body, lines[j])
				j++
			}
			switch kind {
			case "src":
				lang := rest
				if sp := strings.IndexAny(lang, " \t"); sp >= 0 {
					lang = lang[:sp]
				}
				doc.Blocks = append(doc.Blocks, Block{Kind: BlockSrc, Lang: lang, Lines: body})
			case "example":
				doc.Blocks = append(doc.Blocks, Block{Kind: BlockExample, Lines: body})
			case "quote":
				doc.Blocks = append(doc.Blocks, Block{Kind: BlockQuote, Lines: body})
			default:
				// export, comment, html, ... : dropped entirely. This is what
				// keeps raw HTML from ever being passed through.
			}
			i = j + 1

		case strings.HasPrefix(lower, "#+"):
			flushPara()
			parseKeyword(doc, trimmed)
			i++

		case trimmed == "#" || strings.HasPrefix(trimmed, "# "):
			// comment line
			flushPara()
			i++

		case drawerRe.MatchString(trimmed):
			end := findDrawerEnd(lines, i+1)
			if end < 0 {
				para = append(para, trimmed)
				i++
				break
			}
			flushPara()
			if strings.EqualFold(trimmed, ":PROPERTIES:") && !seenHeading {
				parseFileProperties(doc, lines[i+1:end])
			}
			i = end + 1

		default:
			if m := headingRe.FindStringSubmatch(line); m != nil {
				flushPara()
				seenHeading = true
				title := headingTagsRe.ReplaceAllString(m[2], "")
				doc.Blocks = append(doc.Blocks, Block{Kind: BlockHeading, Level: len(m[1]), Text: strings.TrimSpace(title)})
				i++
				break
			}
			if ruleRe.MatchString(line) {
				flushPara()
				doc.Blocks = append(doc.Blocks, Block{Kind: BlockRule})
				i++
				break
			}
			if strings.HasPrefix(trimmed, "|") {
				flushPara()
				j := i
				var rows []string
				for j < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[j]), "|") {
					rows = append(rows, strings.TrimSpace(lines[j]))
					j++
				}
				doc.Blocks = append(doc.Blocks, Block{Kind: BlockTable, Lines: rows})
				i = j
				break
			}
			if item, ok := parseListItem(line); ok {
				flushPara()
				items, next := collectList(lines, i, item)
				doc.Blocks = append(doc.Blocks, Block{Kind: BlockList, Items: items})
				i = next
				break
			}
			para = append(para, trimmed)
			i++
		}
	}
	flushPara()

	collectLinks(doc)
	if doc.Title == "" {
		for _, b := range doc.Blocks {
			if b.Kind == BlockHeading {
				doc.Title = b.Text
				break
			}
		}
	}
	return doc
}

func parseKeyword(doc *Doc, trimmed string) {
	rest := trimmed[2:]
	colon := strings.Index(rest, ":")
	if colon < 0 {
		return
	}
	key := strings.ToLower(strings.TrimSpace(rest[:colon]))
	val := strings.TrimSpace(rest[colon+1:])
	switch key {
	case "title":
		if doc.Title == "" {
			doc.Title = val
		}
	case "filetags":
		tags := strings.FieldsFunc(val, func(r rune) bool { return r == ':' || r == ' ' || r == '\t' })
		doc.FileTags = append(doc.FileTags, tags...)
		for _, t := range tags {
			if strings.EqualFold(t, "sensitive") {
				doc.Sensitive = true
			}
		}
	}
}

func findDrawerEnd(lines []string, from int) int {
	for j := from; j < len(lines); j++ {
		if strings.EqualFold(strings.TrimSpace(lines[j]), ":END:") {
			return j
		}
		// A drawer cannot contain a heading; bail out so a stray ":foo:"
		// line does not swallow the document.
		if strings.HasPrefix(lines[j], "*") && headingRe.MatchString(lines[j]) {
			return -1
		}
	}
	return -1
}

func parseFileProperties(doc *Doc, body []string) {
	for _, l := range body {
		t := strings.TrimSpace(l)
		if !strings.HasPrefix(t, ":") {
			continue
		}
		idx := strings.Index(t[1:], ":")
		if idx < 0 {
			continue
		}
		key := strings.ToUpper(t[1 : 1+idx])
		val := strings.TrimSpace(t[idx+2:])
		switch key {
		case "ID":
			if ValidOrgID(val) {
				doc.ID = val
			}
		case "SENSITIVE":
			switch strings.ToLower(val) {
			case "t", "true", "yes", "1":
				doc.Sensitive = true
			}
		}
	}
}

func parseListItem(line string) (ListItem, bool) {
	m := listItemRe.FindStringSubmatch(line)
	if m == nil {
		return ListItem{}, false
	}
	indent := len(m[1])
	bullet := m[2]
	// A "*" at column 0 is a heading, not a bullet.
	if bullet == "*" && indent == 0 {
		return ListItem{}, false
	}
	item := ListItem{
		Indent:  indent,
		Ordered: bullet[0] >= '0' && bullet[0] <= '9',
		Text:    m[3],
	}
	if cb := checkboxRe.FindStringSubmatch(item.Text); cb != nil {
		item.Checkbox = strings.ToUpper(cb[1])
		item.Text = cb[2]
	}
	return item, true
}

func collectList(lines []string, start int, first ListItem) ([]ListItem, int) {
	items := []ListItem{first}
	i := start + 1
	for i < len(lines) {
		line := lines[i]
		if strings.TrimSpace(line) == "" {
			// A single blank line followed by another item continues the list.
			if i+1 < len(lines) {
				if _, ok := parseListItem(lines[i+1]); ok {
					i++
					continue
				}
			}
			break
		}
		if item, ok := parseListItem(line); ok {
			items = append(items, item)
			i++
			continue
		}
		// Continuation line: must be indented deeper than the current item.
		cur := &items[len(items)-1]
		if leadingSpaces(line) > cur.Indent {
			cur.Text += " " + strings.TrimSpace(line)
			i++
			continue
		}
		break
	}
	return items, i
}

func leadingSpaces(s string) int {
	n := 0
	for n < len(s) && (s[n] == ' ' || s[n] == '\t') {
		n++
	}
	return n
}

// collectLinks gathers [[id:...]] links from every rendered text region
// (never from src/example blocks or drawers), de-duplicated by target.
func collectLinks(doc *Doc) {
	seen := map[string]int{}
	add := func(text string) {
		for i := 0; i < len(text); i++ {
			if text[i] != '[' {
				continue
			}
			target, label, end, ok := parseBracketLink(text, i)
			if !ok {
				continue
			}
			i = end - 1
			if !strings.HasPrefix(target, "id:") {
				continue
			}
			id := target[3:]
			if !ValidOrgID(id) {
				continue
			}
			if idx, dup := seen[id]; dup {
				if doc.Links[idx].Label == "" {
					doc.Links[idx].Label = label
				}
				continue
			}
			seen[id] = len(doc.Links)
			doc.Links = append(doc.Links, Link{Target: id, Label: label})
		}
	}
	for _, b := range doc.Blocks {
		switch b.Kind {
		case BlockHeading, BlockParagraph:
			add(b.Text)
		case BlockList:
			for _, it := range b.Items {
				add(it.Text)
			}
		case BlockTable, BlockQuote:
			for _, l := range b.Lines {
				add(l)
			}
		}
	}
}

// parseBracketLink parses "[[target]]" or "[[target][label]]" starting at s[i].
// end is the index just past the closing brackets.
func parseBracketLink(s string, i int) (target, label string, end int, ok bool) {
	if !strings.HasPrefix(s[i:], "[[") {
		return "", "", 0, false
	}
	rest := s[i+2:]
	j := strings.Index(rest, "]")
	if j <= 0 {
		return "", "", 0, false
	}
	target = rest[:j]
	if strings.ContainsAny(target, "[\n") {
		return "", "", 0, false
	}
	after := rest[j+1:]
	switch {
	case strings.HasPrefix(after, "]"):
		return target, "", i + j + 4, true
	case strings.HasPrefix(after, "["):
		k := strings.Index(after[1:], "]]")
		if k < 0 {
			return "", "", 0, false
		}
		return target, after[1 : 1+k], i + j + k + 6, true
	}
	return "", "", 0, false
}
