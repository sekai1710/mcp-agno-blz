---
name: agno-docs
description: Routing hint per la documentazione Agno (docs.agno.com). Attiva quando l'utente menziona Agno, Agent, Team, Workflow, AgentOS, Knowledge, Tools, Models providers, embedders, vector db, memory, storage, db backends. Dice solo COSA fa il CLI e QUANDO usarlo.
---

# Agno — docs CLI routing

CLI offline `agno-docs-pp-cli` (~/go/bin). Indicizza `docs.agno.com/llms-full.txt` in SQLite+FTS5. Zero HTML scraping, zero auth.

## Cosa fa

Lookup grounded sulla doc Agno completa (~3200 pagine). 4 comandi agent-native con `--json`:

- `which "<topic>"` — top-N pagine con snippet (start HERE)
- `context <slug-or-url>` — body completo di una pagina
- `examples "<topic>" --language python` — code block paste-ready
- `sections` — mappa sezioni con page count

## Quando usarlo

Prima di rispondere a qualsiasi domanda Agno:
- "come si crea un Team?" → `which "team create"`
- "PostgresDb config" → `examples "PostgresDb" --language python`
- "OpenRouter model" → `which "OpenRouter model"`
- "AgentOS approval workflow" → `context approvals`
- "embedder Gemini" → `examples "GeminiEmbedder"`

## Quando NON usarlo

- Stato runtime di un deployment Agno (CLI conosce solo la doc)
- Domande non-Agno (usa il CLI giusto per quell'API)
- Generazione codice non documentato (la doc copre solo API pubbliche)

## Setup

```bash
agno-docs-pp-cli sync       # ~5s
agno-docs-pp-cli doctor     # verifica 3000+ pagine
```

## Flow agent ideale

```
which → pick best URL → context <slug> → (optional) examples <query>
```

Non saltare `which`. La slug per `context` deve venire dal risultato di `which`, non inventata.

## File reference

- `~/printing-press/library/agno-docs/README.md` — installer + Python toolkit ready-to-paste
- `~/printing-press/library/agno-docs/AGENTS.md` — pattern grounding per agenti
- `agency-platform: modules/agno_docs/docs_tool.py` — Agno Toolkit wrapping CLI (asyncio.to_thread + subprocess)
