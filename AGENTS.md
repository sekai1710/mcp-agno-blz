# AGENTS.md — agno-docs-pp-cli

Guidance for AI agents (Claude Code, Cursor, custom Agno agents) that have this CLI on PATH.

## What this gives you

A grounded, offline lookup over `docs.agno.com`. Use it instead of guessing Agno APIs.

## Decision tree

The user asks something about Agno (Agent, Team, Workflow, AgentOS, Knowledge, Tools, Models, DB, Memory, Embedders, Storage, etc.):

1. **Always start with `which`.** It returns 5 top-ranked pages with snippets — enough to pick the right one without reading the docs site.

   ```bash
   agno-docs-pp-cli which "<topic in plain English>" --json
   ```

2. **Read the page that matched best.** Use `context`. Accepts a slug OR the full URL from the `which` result.

   ```bash
   agno-docs-pp-cli context teams --json
   # or
   agno-docs-pp-cli context "https://docs.agno.com/teams/overview" --json
   ```

3. **Need a code snippet?** Use `examples`, filter by language.

   ```bash
   agno-docs-pp-cli examples "PostgresDb agent" --language python --json
   ```

4. **Lost? Need to orient yourself?** List sections.

   ```bash
   agno-docs-pp-cli sections --json
   ```

## When NOT to use this

- Questions about something other than Agno (use the right tool for the API in question).
- Live runtime state of an Agno deployment (this CLI knows the docs, not your running app).
- Generating an Agno install — read the `quickstart` page with `context quickstart`.

## Output contract

- Every command supports `--json`. Always pass it when calling from an agent.
- Errors are returned as `{"error": "<message>", ...}` on stdout (not stderr) with exit code 1.
- Empty results: `{"matches": []}` or `{"examples": []}` — never raise, always return.
- Slugs in `context`: a full URL gets normalised to its last path segment automatically.

## Performance

- All queries: <50ms warm, <200ms cold (FTS5 with porter tokenizer).
- One process per call — safe to fan out in parallel.
- Subprocess crashes are contained; the index itself is read-only after `sync`.

## Refresh policy

Run `agno-docs-pp-cli sync` weekly or when the user mentions a feature that doesn't appear in `which` results. The full re-sync takes ~5 seconds.

`doctor --json` returns `last_sync_at` for stale-detection logic.

## Pattern: agent grounding flow

```text
user: "How do I run a Team in coordinate mode?"
  agent → agno_docs_which("team coordinate mode")
  agent ← {"matches": [{"url": ".../teams/modes/coordinate", ...}, ...]}
  agent → agno_docs_context("coordinate")
  agent ← {"content": "# Coordinate Mode\n\n...", ...}
  agent → agno_docs_examples("team coordinate", language="python")
  agent ← {"examples": [{"code": "from agno.team import Team, TeamMode\n...", ...}]}
  agent → synthesised answer with citation
```

Never skip step 1. The slug for `context` and the topic for `examples` should both come from the `which` result, not invented from scratch.
