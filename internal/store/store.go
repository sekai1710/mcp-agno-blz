// Package store wraps SQLite + FTS5 for the Agno docs index. One table per
// content type (pages, examples) with a matching FTS5 virtual table kept in
// sync via INSERT/UPDATE triggers. Schema is created idempotently on Open.
package store

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"

	"github.com/sekai1710/agno-docs-pp-cli/internal/parser"
)

const schema = `
CREATE TABLE IF NOT EXISTS pages (
	id        TEXT PRIMARY KEY,
	url       TEXT NOT NULL UNIQUE,
	title     TEXT NOT NULL,
	section   TEXT NOT NULL,
	slug      TEXT NOT NULL,
	content   TEXT NOT NULL,
	headings  TEXT NOT NULL DEFAULT ''
);
CREATE VIRTUAL TABLE IF NOT EXISTS pages_fts USING fts5(
	title, section, slug, content, headings,
	content='pages', content_rowid='rowid',
	tokenize='porter unicode61'
);
CREATE TRIGGER IF NOT EXISTS pages_ai AFTER INSERT ON pages BEGIN
	INSERT INTO pages_fts(rowid, title, section, slug, content, headings)
	VALUES (new.rowid, new.title, new.section, new.slug, new.content, new.headings);
END;
CREATE TRIGGER IF NOT EXISTS pages_ad AFTER DELETE ON pages BEGIN
	INSERT INTO pages_fts(pages_fts, rowid, title, section, slug, content, headings)
	VALUES('delete', old.rowid, old.title, old.section, old.slug, old.content, old.headings);
END;
CREATE TRIGGER IF NOT EXISTS pages_au AFTER UPDATE ON pages BEGIN
	INSERT INTO pages_fts(pages_fts, rowid, title, section, slug, content, headings)
	VALUES('delete', old.rowid, old.title, old.section, old.slug, old.content, old.headings);
	INSERT INTO pages_fts(rowid, title, section, slug, content, headings)
	VALUES (new.rowid, new.title, new.section, new.slug, new.content, new.headings);
END;

CREATE TABLE IF NOT EXISTS examples (
	id         TEXT PRIMARY KEY,
	page_url   TEXT NOT NULL,
	page_title TEXT NOT NULL,
	language   TEXT NOT NULL,
	code       TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS examples_page_url_idx ON examples(page_url);
CREATE VIRTUAL TABLE IF NOT EXISTS examples_fts USING fts5(
	page_title, language, code,
	content='examples', content_rowid='rowid',
	tokenize='porter unicode61'
);
CREATE TRIGGER IF NOT EXISTS examples_ai AFTER INSERT ON examples BEGIN
	INSERT INTO examples_fts(rowid, page_title, language, code)
	VALUES (new.rowid, new.page_title, new.language, new.code);
END;
CREATE TRIGGER IF NOT EXISTS examples_ad AFTER DELETE ON examples BEGIN
	INSERT INTO examples_fts(examples_fts, rowid, page_title, language, code)
	VALUES ('delete', old.rowid, old.page_title, old.language, old.code);
END;

CREATE TABLE IF NOT EXISTS meta (
	key   TEXT PRIMARY KEY,
	value TEXT NOT NULL
);
`

// DB wraps an open SQLite connection with the agno-docs schema applied.
type DB struct {
	conn *sql.DB
	path string
}

// DefaultPath returns ~/.local/share/agno-docs-pp-cli/data.db (XDG-style).
func DefaultPath() string {
	if v := os.Getenv("AGNO_DOCS_DB"); v != "" {
		return v
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "agno-docs-pp-cli", "data.db")
}

// Open opens (or creates) the SQLite database at path and applies the schema.
func Open(ctx context.Context, path string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("creating db directory: %w", err)
	}
	dsn := path + "?_journal_mode=WAL&_busy_timeout=5000"
	conn, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening sqlite: %w", err)
	}
	if _, err := conn.ExecContext(ctx, schema); err != nil {
		conn.Close()
		return nil, fmt.Errorf("applying schema: %w", err)
	}
	return &DB{conn: conn, path: path}, nil
}

// Close releases the underlying connection.
func (d *DB) Close() error { return d.conn.Close() }

// Path returns the on-disk db path.
func (d *DB) Path() string { return d.path }

// Raw returns the underlying *sql.DB (used by sync to wrap operations in a tx).
func (d *DB) Raw() *sql.DB { return d.conn }

// UpsertPage inserts or replaces a page row.
func (d *DB) UpsertPage(ctx context.Context, p *parser.Page) error {
	headings := strings.Join(p.Headings, "\n")
	_, err := d.conn.ExecContext(ctx, `
		INSERT INTO pages (id, url, title, section, slug, content, headings)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			url=excluded.url,
			title=excluded.title,
			section=excluded.section,
			slug=excluded.slug,
			content=excluded.content,
			headings=excluded.headings`,
		p.ID, p.URL, p.Title, p.Section, p.Slug, p.Content, headings,
	)
	return err
}

// DeleteExamplesForURL removes all examples linked to the given page URL.
// Used before re-inserting fresh examples on resync (example IDs are
// derived from URL + index, so deleting first avoids stale rows when a page's
// example count shrinks).
func (d *DB) DeleteExamplesForURL(ctx context.Context, url string) error {
	_, err := d.conn.ExecContext(ctx, `DELETE FROM examples WHERE page_url = ?`, url)
	return err
}

// InsertExample stores a code example.
func (d *DB) InsertExample(ctx context.Context, id, pageURL, pageTitle, language, code string) error {
	_, err := d.conn.ExecContext(ctx, `
		INSERT INTO examples (id, page_url, page_title, language, code)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			page_url=excluded.page_url,
			page_title=excluded.page_title,
			language=excluded.language,
			code=excluded.code`,
		id, pageURL, pageTitle, language, code,
	)
	return err
}

// SetMeta stores a key/value pair (e.g. last_sync_at, source_size).
func (d *DB) SetMeta(ctx context.Context, key, value string) error {
	_, err := d.conn.ExecContext(ctx, `
		INSERT INTO meta (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value=excluded.value`,
		key, value,
	)
	return err
}

// GetMeta retrieves a meta value (empty string when missing).
func (d *DB) GetMeta(ctx context.Context, key string) string {
	var v string
	_ = d.conn.QueryRowContext(ctx, `SELECT value FROM meta WHERE key = ?`, key).Scan(&v)
	return v
}

// PageHit is one row returned by SearchPages.
type PageHit struct {
	URL     string  `json:"url"`
	Title   string  `json:"title"`
	Section string  `json:"section"`
	Slug    string  `json:"slug"`
	Snippet string  `json:"snippet"`
	Rank    float64 `json:"rank"`
}

// SearchPages runs an FTS5 MATCH query against pages and returns the top n
// hits ordered by bm25 rank (best first). Returns a 60-char snippet around
// the match.
func (d *DB) SearchPages(ctx context.Context, query string, limit int) ([]PageHit, error) {
	if limit <= 0 {
		limit = 10
	}
	q := sanitizeFTS(query)
	// bm25 weights: title=10, section=2, slug=8, content=1, headings=3.
	// Heavy title/slug weighting fixes "find the reference page" intent
	// (e.g. "PostgresDb" should hit reference/storage/postgres, not pages
	// that incidentally import it).
	rows, err := d.conn.QueryContext(ctx, `
		SELECT
			p.url, p.title, p.section, p.slug,
			snippet(pages_fts, 3, '[', ']', '…', 12) AS snippet,
			bm25(pages_fts, 10.0, 2.0, 8.0, 1.0, 3.0) AS rank
		FROM pages_fts
		JOIN pages p ON p.rowid = pages_fts.rowid
		WHERE pages_fts MATCH ?
		ORDER BY rank
		LIMIT ?`,
		q, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var hits []PageHit
	for rows.Next() {
		var h PageHit
		if err := rows.Scan(&h.URL, &h.Title, &h.Section, &h.Slug, &h.Snippet, &h.Rank); err != nil {
			return nil, err
		}
		hits = append(hits, h)
	}
	return hits, rows.Err()
}

// ExampleHit is one row returned by SearchExamples.
type ExampleHit struct {
	PageURL   string  `json:"page_url"`
	PageTitle string  `json:"page_title"`
	Language  string  `json:"language"`
	Code      string  `json:"code"`
	Rank      float64 `json:"rank"`
}

// SearchExamples runs FTS5 against the examples table.
func (d *DB) SearchExamples(ctx context.Context, query string, limit int) ([]ExampleHit, error) {
	if limit <= 0 {
		limit = 5
	}
	q := sanitizeFTS(query)
	rows, err := d.conn.QueryContext(ctx, `
		SELECT
			e.page_url, e.page_title, e.language, e.code,
			bm25(examples_fts) AS rank
		FROM examples_fts
		JOIN examples e ON e.rowid = examples_fts.rowid
		WHERE examples_fts MATCH ?
		ORDER BY rank
		LIMIT ?`,
		q, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var hits []ExampleHit
	for rows.Next() {
		var h ExampleHit
		if err := rows.Scan(&h.PageURL, &h.PageTitle, &h.Language, &h.Code, &h.Rank); err != nil {
			return nil, err
		}
		hits = append(hits, h)
	}
	return hits, rows.Err()
}

// GetPageBySlug returns the first page matching slug exactly, then by trailing
// URL segment, then by section+slug partial match. Returns nil when nothing
// matches.
func (d *DB) GetPageBySlug(ctx context.Context, slug string) (*parser.Page, error) {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return nil, nil
	}
	queries := []struct {
		sql  string
		args []any
	}{
		{`SELECT id, url, title, section, slug, content, headings FROM pages WHERE slug = ? LIMIT 1`, []any{slug}},
		{`SELECT id, url, title, section, slug, content, headings FROM pages WHERE url LIKE ? LIMIT 1`, []any{"%/" + slug}},
		{`SELECT id, url, title, section, slug, content, headings FROM pages WHERE url LIKE ? LIMIT 1`, []any{"%" + slug + "%"}},
	}
	for _, q := range queries {
		var (
			id, url, title, section, sl, content, headings string
		)
		err := d.conn.QueryRowContext(ctx, q.sql, q.args...).Scan(&id, &url, &title, &section, &sl, &content, &headings)
		if err == sql.ErrNoRows {
			continue
		}
		if err != nil {
			return nil, err
		}
		p := &parser.Page{
			ID: id, URL: url, Title: title, Section: section, Slug: sl, Content: content,
		}
		if headings != "" {
			p.Headings = strings.Split(headings, "\n")
		}
		return p, nil
	}
	return nil, nil
}

// ListSections returns distinct sections with page counts (largest first).
func (d *DB) ListSections(ctx context.Context) ([]struct {
	Section string
	Count   int
}, error) {
	rows, err := d.conn.QueryContext(ctx, `
		SELECT section, COUNT(*) AS n
		FROM pages
		GROUP BY section
		ORDER BY n DESC, section ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []struct {
		Section string
		Count   int
	}
	for rows.Next() {
		var s string
		var n int
		if err := rows.Scan(&s, &n); err != nil {
			return nil, err
		}
		out = append(out, struct {
			Section string
			Count   int
		}{s, n})
	}
	return out, rows.Err()
}

// Stats returns aggregate counts (pages, examples) for the doctor command.
func (d *DB) Stats(ctx context.Context) (pages, examples int, err error) {
	if err = d.conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM pages`).Scan(&pages); err != nil {
		return
	}
	if err = d.conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM examples`).Scan(&examples); err != nil {
		return
	}
	return
}

// sanitizeFTS escapes user input for FTS5 MATCH. Quotes each non-empty token
// to neutralize FTS5 syntax characters (- : * ^ etc.) while keeping multi-word
// queries as AND.
func sanitizeFTS(q string) string {
	q = strings.TrimSpace(q)
	if q == "" {
		return q
	}
	fields := strings.Fields(q)
	parts := make([]string, 0, len(fields))
	for _, f := range fields {
		f = strings.Trim(f, `"'.,;:()[]{}`)
		if f == "" {
			continue
		}
		// Wrap each token in double quotes so FTS5 treats it as a literal phrase.
		parts = append(parts, `"`+strings.ReplaceAll(f, `"`, ``)+`"`)
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, " ")
}
