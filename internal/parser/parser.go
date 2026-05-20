// Package parser splits the Agno llms-full.txt bundle into individual
// documentation pages. Each page starts with a top-level markdown heading
// ("# Title") immediately followed by a "Source: <url>" line. The body
// continues until the next such header (or EOF).
//
// llms-full.txt is plain markdown — no HTML, no entity encoding — so parsing
// is a linear scan with one regexp per heading; no DOM, no tag stripper.
package parser

import (
	"bufio"
	"crypto/md5"
	"fmt"
	"io"
	"regexp"
	"strings"
)

// Page is one documentation page extracted from llms-full.txt.
type Page struct {
	ID           string   // md5(URL) — stable across syncs
	URL          string   // canonical docs URL from the "Source:" line
	Title        string   // first-line heading
	Section      string   // derived from URL path (e.g. "agent-os/approvals")
	Slug         string   // last URL segment
	Content      string   // full markdown body (heading + Source line stripped)
	Headings     []string // ## and ### headings extracted from body
	CodeExamples []string // contents of fenced ``` blocks
}

var (
	titleRE   = regexp.MustCompile(`^#\s+(.+)$`)
	sourceRE  = regexp.MustCompile(`^Source:\s+(\S+)\s*$`)
	headingRE = regexp.MustCompile(`^#{2,3}\s+(.+)$`)
	fenceRE   = regexp.MustCompile("^```")
)

// Parse reads llms-full.txt content and yields every page found.
func Parse(r io.Reader) ([]*Page, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1024*1024), 64*1024*1024)

	var pages []*Page
	var cur *Page
	var body strings.Builder
	var inFence bool
	var fenceBuf strings.Builder

	flush := func() {
		if cur == nil {
			return
		}
		cur.Content = strings.TrimSpace(body.String())
		pages = append(pages, cur)
		cur = nil
		body.Reset()
		inFence = false
		fenceBuf.Reset()
	}

	for scanner.Scan() {
		line := scanner.Text()

		// Page boundary: "# Title" followed by "Source:" on the next line.
		// We need a 2-line lookahead, but bufio.Scanner is forward-only —
		// so we use a small state machine: when we see "# Title" we tentatively
		// open a page; if the very next line is "Source:", commit; otherwise
		// it was just an in-body heading and we restore it to the body.
		if cur == nil || (!inFence && titleRE.MatchString(line)) {
			if cur != nil && !inFence && titleRE.MatchString(line) {
				// Peek next line by reading ahead.
				if !scanner.Scan() {
					// EOF mid-header; bail.
					body.WriteString(line)
					body.WriteString("\n")
					break
				}
				next := scanner.Text()
				if m := sourceRE.FindStringSubmatch(next); m != nil {
					flush()
					cur = newPage(titleRE.FindStringSubmatch(line)[1], m[1])
					continue
				}
				// Not a real page boundary — keep both lines in body.
				body.WriteString(line)
				body.WriteString("\n")
				body.WriteString(next)
				body.WriteString("\n")
				continue
			}
			if cur == nil {
				if m := titleRE.FindStringSubmatch(line); m != nil {
					if !scanner.Scan() {
						break
					}
					next := scanner.Text()
					if sm := sourceRE.FindStringSubmatch(next); sm != nil {
						cur = newPage(m[1], sm[1])
						continue
					}
				}
				continue // skip preamble noise before first page
			}
		}

		// Track fenced code blocks for example extraction.
		if fenceRE.MatchString(line) {
			if inFence {
				code := strings.TrimSpace(fenceBuf.String())
				if len(code) >= 10 {
					cur.CodeExamples = append(cur.CodeExamples, code)
				}
				fenceBuf.Reset()
				inFence = false
			} else {
				inFence = true
			}
			body.WriteString(line)
			body.WriteString("\n")
			continue
		}
		if inFence {
			fenceBuf.WriteString(line)
			fenceBuf.WriteString("\n")
			body.WriteString(line)
			body.WriteString("\n")
			continue
		}

		// Collect h2/h3 headings outside code blocks.
		if m := headingRE.FindStringSubmatch(line); m != nil {
			h := strings.TrimSpace(m[1])
			if h != "" && len(h) < 200 {
				cur.Headings = append(cur.Headings, h)
			}
		}

		body.WriteString(line)
		body.WriteString("\n")
	}
	flush()

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning llms-full.txt: %w", err)
	}
	return pages, nil
}

func newPage(title, url string) *Page {
	return &Page{
		ID:      fmt.Sprintf("%x", md5.Sum([]byte(url))),
		URL:     url,
		Title:   strings.TrimSpace(title),
		Section: sectionFromURL(url),
		Slug:    slugFromURL(url),
	}
}

// sectionFromURL extracts a 1-3 segment section identifier from the URL path.
// e.g. https://docs.agno.com/agent-os/approvals/overview → "agent-os/approvals"
func sectionFromURL(u string) string {
	path := strings.TrimPrefix(u, "https://docs.agno.com/")
	path = strings.TrimPrefix(path, "https://docs.agno.com")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		return "root"
	}
	if len(parts) == 1 {
		return parts[0]
	}
	limit := len(parts) - 1
	if limit > 3 {
		limit = 3
	}
	return strings.Join(parts[:limit], "/")
}

// slugFromURL returns the last non-empty path segment.
func slugFromURL(u string) string {
	parts := strings.Split(strings.TrimRight(u, "/"), "/")
	if len(parts) == 0 {
		return u
	}
	return parts[len(parts)-1]
}

// DetectLanguage guesses the language of a fenced code block from its first
// line (Agno annotates many blocks with ```python or ```bash) or content.
func DetectLanguage(code string) string {
	first := code
	if i := strings.IndexByte(code, '\n'); i >= 0 {
		first = code[:i]
	}
	lower := strings.ToLower(strings.TrimSpace(first))
	switch {
	case strings.HasPrefix(lower, "python"), strings.Contains(lower, "from agno"), strings.Contains(lower, "import agno"):
		return "python"
	case strings.HasPrefix(lower, "bash"), strings.HasPrefix(lower, "sh "), strings.Contains(lower, "curl "):
		return "bash"
	case strings.HasPrefix(lower, "json"), strings.HasPrefix(lower, "{"):
		return "json"
	case strings.HasPrefix(lower, "yaml"), strings.HasPrefix(lower, "yml"):
		return "yaml"
	case strings.HasPrefix(lower, "typescript"), strings.HasPrefix(lower, "ts "):
		return "typescript"
	case strings.HasPrefix(lower, "javascript"), strings.HasPrefix(lower, "js "):
		return "javascript"
	}
	lowerAll := strings.ToLower(code)
	switch {
	case strings.Contains(lowerAll, "from agno"), strings.Contains(lowerAll, "def "), strings.Contains(lowerAll, "print("):
		return "python"
	case strings.Contains(lowerAll, "curl "):
		return "bash"
	}
	return "text"
}
