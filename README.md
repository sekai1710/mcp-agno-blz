# agno-docs-pp-cli

> **Offline, agent-native CLI for the [Agno](https://agno.com) developer documentation.**
> One HTTP GET → 3,200 pages indexed in SQLite + FTS5 → ~10-30 ms per query.

```bash
agno-docs-pp-cli which "how do teams work"
agno-docs-pp-cli context teams
agno-docs-pp-cli examples "PostgresDb" --language python
```

[![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Source: llms-full.txt](https://img.shields.io/badge/source-llms--full.txt-brightgreen)](https://docs.agno.com/llms-full.txt)

---

## Install — one command

```bash
curl -sSf https://raw.githubusercontent.com/sekai1710/agno-docs-pp-cli/main/install.sh | bash
```

That script:
1. checks for the Go toolchain (≥1.26)
2. runs `go install -tags sqlite_fts5 github.com/sekai1710/agno-docs-pp-cli/cmd/agno-docs-pp-cli@latest`
3. runs the first `sync` (~5 seconds, downloads `docs.agno.com/llms-full.txt` once)
4. tells you how to use it

If you prefer manual install:

```bash
go install -tags sqlite_fts5 github.com/sekai1710/agno-docs-pp-cli/cmd/agno-docs-pp-cli@latest
agno-docs-pp-cli sync
```

The `sqlite_fts5` tag is **required** — `go-sqlite3` ships FTS5 behind it.

## Quick tour

```bash
agno-docs-pp-cli doctor                              # health check
agno-docs-pp-cli sections                            # map of the docs
agno-docs-pp-cli which "knowledge embedder" -n 3     # top-3 pages
agno-docs-pp-cli context agents                      # full page body
agno-docs-pp-cli examples "team coordinate" --language python
```

Every leaf command supports `--json` for agent consumption and `--db <path>` to override the DB location (default `~/.local/share/agno-docs-pp-cli/data.db`).

## Commands

| Command    | What it does                                                                  |
|-----------|-------------------------------------------------------------------------------|
| `sync`    | Fetch `docs.agno.com/llms-full.txt` and rebuild the index. Re-run weekly.    |
| `which`   | Top-N pages for a topic, with snippet. **Start here in any agent workflow.** |
| `find`    | Same FTS as `which`, longer list, CLI-flavoured output.                       |
| `context` | Full markdown body of one page (by slug or full URL).                         |
| `examples`| Paste-ready code blocks. Filter with `--language python\|bash\|json\|yaml`.   |
| `sections`| List sections (e.g. `models/providers/native`) with page counts.              |
| `doctor`  | Pages, examples, last sync time, db path.                                     |
| `version` | Print the CLI version.                                                        |

## Agent integration

The CLI is framework-agnostic. `--json` output drops into anything that can shell out.

### Agno (Python) — ready-to-paste

A complete Agno `Toolkit` lives at [`examples/agent-toolkit.py`](examples/agent-toolkit.py). Copy it into your project and:

```python
from agent_toolkit import AgnoDocsTool

agent = Agent(
    model=OpenRouter(id="google/gemini-2.5-flash"),
    tools=[AgnoDocsTool()],
    instructions=["Always call agno_docs_which before answering Agno questions."],
)
```

Four tools are registered: `agno_docs_which`, `agno_docs_context`, `agno_docs_examples`, `agno_docs_sections`. All async, all `asyncio.to_thread`-wrapped, all subprocess-isolated (a CLI crash cannot take down your agent).

### Other frameworks

- **LangChain / LangGraph** — wrap as a `Tool` via `subprocess.run`.
- **Claude Code / Cursor** — call directly from `Bash`/`MultiEdit`.
- **CrewAI / AutoGen** — same subprocess pattern as Agno.
- **Shell** — `agno-docs-pp-cli which … --json | jq`.

## How it works

`docs.agno.com` publishes [`llms-full.txt`](https://docs.agno.com/llms-full.txt) — every page of the site concatenated into one plain-text markdown file, formatted per the [llms.txt](https://llmstxt.org/) convention:

```
# Approvals
Source: https://docs.agno.com/agent-os/approvals/overview

Manage approval workflows for agents and teams via the AgentOS Control Panel.
...
```

The CLI is a small pipeline around that single source:

```
fetch llms-full.txt (10 MB, 1 HTTP GET)
  ↓
parse on "# Title\nSource: <url>" boundaries (one regexp, no HTML)
  ↓
SQLite + FTS5 (porter unicode61, content-table + auto-sync triggers)
  ↓
which / find / context / examples / sections / doctor
```

Total Go source: ~1,100 lines. No HTML scraper, no headless browser, no auth, no rate limiting — because the source is a single file maintained by the docs team specifically for this purpose.

## Why this exists

If an LLM coding agent calls Agno APIs, it needs grounded reference. The three bad alternatives:

1. **Let it guess.** Hallucinated imports, made-up args, dead URLs.
2. **Web-fetch `docs.agno.com`.** Slow, HTML parsing, breaks when the site rebuilds.
3. **Scrape and re-scrape.** Brittle. Every page change breaks something.

`llms-full.txt` solves the source problem. This CLI solves the integration problem: turn the 9.9 MB blob into a 30 ms FTS5 query with JSON output suitable for any agent framework.

## Comparison with BLZ

[BLZ](https://github.com/outfitter-dev/blz) is an excellent general-purpose MCP server for any `llms.txt` source. If you already use BLZ inside an MCP-aware client (Claude Desktop, Cursor's MCP wiring, etc.), point BLZ at `https://docs.agno.com/llms-full.txt` and you're done.

This CLI is the **complementary** path:

| Property                  | BLZ (MCP server)            | `agno-docs-pp-cli`                   |
|---------------------------|-----------------------------|--------------------------------------|
| Transport                 | MCP stdio                   | Standalone CLI binary                |
| Setup                     | Install BLZ + add a source  | `go install` + `sync`                |
| Sources                   | Any `llms.txt`              | Agno (this repo) / sibling per-source repos |
| Integrates with           | MCP-aware clients only      | **Any** subprocess-capable runtime   |
| Where it runs             | Beside your MCP client      | Anywhere you can exec a binary       |
| Domain-specific commands  | Generic search              | `which` / `context` / `examples`     |
| Typical latency           | ~50-80 ms                   | ~10-30 ms                            |

Pick BLZ when you want one tool that handles many `llms.txt` sources in MCP clients.
Pick this CLI when you want a single self-contained binary on a server, in CI, or wired into a non-MCP agent.

You can run both side by side.

## Repository history (honest version)

This repo originally shipped a **benchmark** comparing an HTML scraper-MCP against BLZ for Agno docs. It did not ship an installable CLI — only the comparison artifacts. The benchmark methodology and numbers are preserved under [`benchmark/`](benchmark/) for reference.

The repo now ships the actual CLI:

- **Generator**: [Printing Press](https://github.com/mvanhorn/cli-printing-press) (CLI scaffolding).
- **Source path**: `llms-full.txt` (plain markdown) instead of HTML scraping — same insight the BLZ project popularized, applied to a CLI instead of an MCP server.
- **Result**: a single Go binary that any agent runtime (Agno, LangChain, Claude Code, plain bash) can call.

If you came here from the original benchmark post: this is the artifact that should have shipped from day one. Apologies for the delay.

## Development

```bash
git clone https://github.com/sekai1710/agno-docs-pp-cli
cd agno-docs-pp-cli
make build          # ./agno-docs-pp-cli
./agno-docs-pp-cli sync --file ./testdata/llms-full.txt
./agno-docs-pp-cli doctor --json | jq
make install        # → $GOBIN/agno-docs-pp-cli
```

Or skip `make` and use `go install -tags sqlite_fts5 ./cmd/agno-docs-pp-cli`.

## License

MIT. See [`LICENSE`](LICENSE).
