package cli

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/sekai1710/agno-docs-pp-cli/internal/parser"
	"github.com/sekai1710/agno-docs-pp-cli/internal/store"
)

const llmsFullURL = "https://docs.agno.com/llms-full.txt"

func newSyncCmd(f *Flags) *cobra.Command {
	var sourceURL string
	var sourceFile string
	cmd := &cobra.Command{
		Use:     "sync",
		Aliases: []string{"index", "fetch"},
		Short:   "Download docs.agno.com/llms-full.txt and rebuild the local index",
		Long: `sync fetches the Agno llms-full.txt bundle (single HTTP GET, no HTML
parsing) and rebuilds the local SQLite/FTS5 index.

Run once before using find/which/context/examples. Re-run to refresh after
docs.agno.com updates (typically <10s on a normal connection).`,
		Example: `  # Default: fetch from docs.agno.com
  agno-docs-pp-cli sync

  # Index from a local copy
  agno-docs-pp-cli sync --file ./llms-full.txt

  # Custom URL (e.g. a mirror)
  agno-docs-pp-cli sync --source https://mirror.example.com/llms-full.txt`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			db, err := openDB(ctx, f)
			if err != nil {
				return err
			}
			defer db.Close()

			var body io.ReadCloser
			var size int64
			var origin string

			if sourceFile != "" {
				origin = sourceFile
				fi, err := os.Open(sourceFile)
				if err != nil {
					return fmt.Errorf("opening %s: %w", sourceFile, err)
				}
				if st, _ := fi.Stat(); st != nil {
					size = st.Size()
				}
				body = fi
			} else {
				if sourceURL == "" {
					sourceURL = llmsFullURL
				}
				origin = sourceURL
				fmt.Fprintf(cmd.ErrOrStderr(), "Fetching %s...\n", sourceURL)
				rc, sz, err := httpGet(ctx, sourceURL)
				if err != nil {
					return err
				}
				body = rc
				size = sz
			}
			defer body.Close()

			fmt.Fprintln(cmd.ErrOrStderr(), "Parsing sections...")
			counter := &countingReader{r: body}
			pages, err := parser.Parse(counter)
			if err != nil {
				return err
			}
			if size <= 0 {
				size = counter.n
			}
			total := len(pages)
			fmt.Fprintf(cmd.ErrOrStderr(), "Found %d pages. Indexing...\n", total)

			indexed, exampleCount := 0, 0
			for _, p := range pages {
				if err := db.UpsertPage(ctx, p); err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "  ERR %s: %v\n", p.URL, err)
					continue
				}
				indexed++

				// Refresh examples for this page (delete-then-insert).
				if err := db.DeleteExamplesForURL(ctx, p.URL); err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "  WARN %s clearing examples: %v\n", p.URL, err)
				}
				for i, code := range p.CodeExamples {
					id := fmt.Sprintf("%x-%d", md5.Sum([]byte(p.URL)), i)
					if err := db.InsertExample(ctx, id, p.URL, p.Title, parser.DetectLanguage(code), code); err == nil {
						exampleCount++
					}
				}
				if indexed%200 == 0 {
					fmt.Fprintf(cmd.ErrOrStderr(), "  %d/%d pages indexed...\n", indexed, total)
				}
			}

			now := time.Now().UTC().Format(time.RFC3339)
			_ = db.SetMeta(ctx, "last_sync_at", now)
			_ = db.SetMeta(ctx, "source", origin)
			_ = db.SetMeta(ctx, "source_bytes", strconv.FormatInt(size, 10))
			_ = db.SetMeta(ctx, "pages_count", strconv.Itoa(indexed))
			_ = db.SetMeta(ctx, "examples_count", strconv.Itoa(exampleCount))

			result := map[string]any{
				"pages_indexed":   indexed,
				"pages_total":     total,
				"examples_stored": exampleCount,
				"source":          origin,
				"source_bytes":    size,
				"db_path":         db.Path(),
				"synced_at":       now,
			}
			if f.JSON {
				return writeJSON(cmd.OutOrStdout(), result)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "\nSync complete:\n")
			fmt.Fprintf(cmd.OutOrStdout(), "  Pages indexed:   %d / %d\n", indexed, total)
			fmt.Fprintf(cmd.OutOrStdout(), "  Examples stored: %d\n", exampleCount)
			fmt.Fprintf(cmd.OutOrStdout(), "  Source bytes:    %d\n", size)
			fmt.Fprintf(cmd.OutOrStdout(), "  Database:        %s\n", db.Path())
			fmt.Fprintln(cmd.OutOrStdout(), "\nNext: agno-docs-pp-cli which \"create agent\"")
			return nil
		},
	}
	cmd.Flags().StringVar(&sourceURL, "source", "", "Override the llms-full.txt URL (default: docs.agno.com)")
	cmd.Flags().StringVar(&sourceFile, "file", "", "Read llms-full.txt from a local path instead of fetching")
	return cmd
}

func httpGet(ctx context.Context, url string) (io.ReadCloser, int64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", "agno-docs-pp-cli/"+Version)
	req.Header.Set("Accept", "text/plain")
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("fetching %s: %w", url, err)
	}
	if resp.StatusCode != 200 {
		resp.Body.Close()
		return nil, 0, fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}
	return resp.Body, resp.ContentLength, nil
}

func openDB(ctx context.Context, f *Flags) (*store.DB, error) {
	path := f.DB
	if path == "" {
		path = store.DefaultPath()
	}
	return store.Open(ctx, path)
}

func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

type countingReader struct {
	r io.Reader
	n int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += int64(n)
	return n, err
}
