package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newFindCmd(f *Flags) *cobra.Command {
	var limit int
	cmd := &cobra.Command{
		Use:   "find <query>",
		Short: "Full-text search across all Agno docs pages",
		Long: `find runs an FTS5 MATCH over page titles, sections, slugs, and body
content. Returns ranked hits with a short context snippet around the match.

For natural-language questions ("how do teams work?") prefer 'which' — it is
a thin wrapper around find with slightly different ranking and presentation
geared at agent consumption.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			db, err := openDB(ctx, f)
			if err != nil {
				return err
			}
			defer db.Close()
			query := strings.Join(args, " ")
			hits, err := db.SearchPages(ctx, query, limit)
			if err != nil {
				return err
			}
			if f.JSON {
				return writeJSON(cmd.OutOrStdout(), map[string]any{
					"query":   query,
					"results": hits,
				})
			}
			if len(hits) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No matches.")
				return nil
			}
			for i, h := range hits {
				fmt.Fprintf(cmd.OutOrStdout(), "%d. %s\n   %s\n   %s\n   %s\n\n",
					i+1, h.Title, h.URL, h.Section, h.Snippet,
				)
			}
			return nil
		},
	}
	cmd.Flags().IntVarP(&limit, "limit", "n", 10, "Maximum number of hits")
	return cmd
}

func newWhichCmd(f *Flags) *cobra.Command {
	var limit int
	cmd := &cobra.Command{
		Use:   "which <query>",
		Short: "Find the Agno docs page(s) covering a topic (agent-friendly)",
		Long: `which is the recommended first call for any agent grounding workflow.
It returns the top-ranked pages for the topic with title, URL, and snippet —
just enough to pick which page to 'context' next.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			db, err := openDB(ctx, f)
			if err != nil {
				return err
			}
			defer db.Close()
			query := strings.Join(args, " ")
			hits, err := db.SearchPages(ctx, query, limit)
			if err != nil {
				return err
			}
			if f.JSON {
				return writeJSON(cmd.OutOrStdout(), map[string]any{
					"query":   query,
					"matches": hits,
				})
			}
			if len(hits) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No matches.")
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Top %d matches for %q:\n\n", len(hits), query)
			for i, h := range hits {
				fmt.Fprintf(cmd.OutOrStdout(), "%d. [%s] %s\n   %s\n   %s\n\n",
					i+1, h.Section, h.Title, h.URL, h.Snippet,
				)
			}
			return nil
		},
	}
	cmd.Flags().IntVarP(&limit, "limit", "n", 5, "Maximum number of matches")
	return cmd
}

func newContextCmd(f *Flags) *cobra.Command {
	return &cobra.Command{
		Use:   "context <slug-or-url>",
		Short: "Print the full markdown body of one docs page",
		Long: `context retrieves the complete body of a single docs page so an agent
can read it before answering. Accepts either:

  - the page slug (last URL segment, e.g. "agents")
  - the full https://docs.agno.com/... URL
  - a partial path (e.g. "agent-os/approvals")

If multiple pages match, the first by slug wins.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			db, err := openDB(ctx, f)
			if err != nil {
				return err
			}
			defer db.Close()

			target := strings.TrimSpace(args[0])
			if strings.HasPrefix(target, "http") {
				target = strings.TrimRight(target, "/")
				if i := strings.LastIndex(target, "/"); i >= 0 {
					target = target[i+1:]
				}
			}

			page, err := db.GetPageBySlug(ctx, target)
			if err != nil {
				return err
			}
			if page == nil {
				if f.JSON {
					return writeJSON(cmd.OutOrStdout(), map[string]any{
						"error": "page not found",
						"slug":  target,
					})
				}
				return fmt.Errorf("page not found: %s", target)
			}
			if f.JSON {
				return writeJSON(cmd.OutOrStdout(), map[string]any{
					"url":      page.URL,
					"title":    page.Title,
					"section":  page.Section,
					"slug":     page.Slug,
					"headings": page.Headings,
					"content":  page.Content,
				})
			}
			fmt.Fprintf(cmd.OutOrStdout(), "# %s\nSource: %s\n\n%s\n", page.Title, page.URL, page.Content)
			return nil
		},
	}
}

func newExamplesCmd(f *Flags) *cobra.Command {
	var limit int
	var language string
	cmd := &cobra.Command{
		Use:   "examples <query>",
		Short: "Extract paste-ready code examples matching a query",
		Long: `examples returns code blocks (python/bash/yaml/...) from Agno docs
pages matching the query. Use --language to filter by language tag.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			db, err := openDB(ctx, f)
			if err != nil {
				return err
			}
			defer db.Close()
			query := strings.Join(args, " ")
			hits, err := db.SearchExamples(ctx, query, limit*3) // overfetch when filtering
			if err != nil {
				return err
			}
			if language != "" {
				filtered := hits[:0]
				for _, h := range hits {
					if strings.EqualFold(h.Language, language) {
						filtered = append(filtered, h)
					}
				}
				hits = filtered
			}
			if len(hits) > limit {
				hits = hits[:limit]
			}
			if f.JSON {
				return writeJSON(cmd.OutOrStdout(), map[string]any{
					"query":    query,
					"language": language,
					"examples": hits,
				})
			}
			if len(hits) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No examples found.")
				return nil
			}
			for i, h := range hits {
				fmt.Fprintf(cmd.OutOrStdout(), "--- %d. [%s] %s (%s) ---\n%s\n\n",
					i+1, h.Language, h.PageTitle, h.PageURL, h.Code,
				)
			}
			return nil
		},
	}
	cmd.Flags().IntVarP(&limit, "limit", "n", 5, "Maximum examples to return")
	cmd.Flags().StringVar(&language, "language", "", "Filter by language tag (python|bash|json|yaml|...)")
	return cmd
}

func newSectionsCmd(f *Flags) *cobra.Command {
	return &cobra.Command{
		Use:   "sections",
		Short: "List documentation sections and page counts",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			db, err := openDB(ctx, f)
			if err != nil {
				return err
			}
			defer db.Close()
			rows, err := db.ListSections(ctx)
			if err != nil {
				return err
			}
			if f.JSON {
				out := make([]map[string]any, 0, len(rows))
				for _, r := range rows {
					out = append(out, map[string]any{"section": r.Section, "pages": r.Count})
				}
				return writeJSON(cmd.OutOrStdout(), map[string]any{"sections": out})
			}
			for _, r := range rows {
				fmt.Fprintf(cmd.OutOrStdout(), "%-40s %d\n", r.Section, r.Count)
			}
			return nil
		},
	}
}

func newDoctorCmd(f *Flags) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Report database health and sync state",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			db, err := openDB(ctx, f)
			if err != nil {
				if f.JSON {
					return writeJSON(cmd.OutOrStdout(), map[string]any{"ok": false, "error": err.Error()})
				}
				return err
			}
			defer db.Close()
			pages, examples, err := db.Stats(ctx)
			if err != nil {
				return err
			}
			out := map[string]any{
				"ok":           pages > 0,
				"version":      Version,
				"db_path":      db.Path(),
				"pages":        pages,
				"examples":     examples,
				"last_sync_at": db.GetMeta(ctx, "last_sync_at"),
				"source":       db.GetMeta(ctx, "source"),
				"source_bytes": db.GetMeta(ctx, "source_bytes"),
			}
			if pages == 0 {
				out["hint"] = "Run: agno-docs-pp-cli sync"
			}
			if f.JSON {
				return writeJSON(cmd.OutOrStdout(), out)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "agno-docs-pp-cli %s\n", Version)
			fmt.Fprintf(cmd.OutOrStdout(), "Database: %s\n", db.Path())
			fmt.Fprintf(cmd.OutOrStdout(), "Pages:    %d\n", pages)
			fmt.Fprintf(cmd.OutOrStdout(), "Examples: %d\n", examples)
			fmt.Fprintf(cmd.OutOrStdout(), "Last sync: %s\n", db.GetMeta(ctx, "last_sync_at"))
			if pages == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "\nHint: run `agno-docs-pp-cli sync` to index the docs.")
			}
			return nil
		},
	}
}
