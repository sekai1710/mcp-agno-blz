# Raw Benchmark Data — 2026-05-12

## Query 1: "workflow step parallel execution"

### agno-docs (Attilio81)
- search output chars: 610
- estimated tokens: 153
- top result relevant: NO (sequence-of-steps, not parallel)
- snippet included: NO
- follow-up call needed: YES (get_agno_page)
- get_agno_page output chars: ~10000 (TRUNCATED)
- est tokens content: 2500
- noise ratio: ~80% sidebar nav + 20% real content
- total Q&A est tokens: 3000-5000+

### agno-docs-blz
- search output chars: 1247
- est tokens: 312
- top result relevant: YES (Parallel Workflow)
- snippet included: YES (5/5)
- follow-up call: optional
- get snippet output chars: 1050 (clean, no nav)
- est tokens: 263
- total Q&A est tokens: 575

## Query 2: "agent memory persistent storage"

### agno-docs
- search output chars: 644 → est 161 tokens
- top result relevant: NO (in-memory storage example, user asked persistent)
- 5/5 relevant: NO (3/5 wrong concept)

### BLZ
- search output chars: 1296 → est 324 tokens
- top result relevant: YES (Agent With Persistent Memory)
- 5/5 relevant: YES
- bonus: result #4 shows "Memory vs Storage" comparison table snippet

## Query 3: "team coordinator delegation"

### agno-docs
- search output chars: 620 → est 155 tokens
- top 5: ALL keyword "team" matches (deploy/observability/reasoning), 0/5 about delegation concept
- semantic understanding: FAIL

### BLZ
- search output chars: 1198 → est 300 tokens
- top result: /teams/delegation exact concept
- 5/5 relevant (delegation, tasks mode, route mode, structured input)

## Aggregated

| Metric | agno-docs (Attilio81) | BLZ |
|---|---|---|
| Top-result relevant (3 queries) | 0/3 | 3/3 |
| Snippets in search results | 0 | 5/5 every query |
| Search output (avg chars) | 625 | 1247 |
| Get page output (Q1) | 10000+ truncated | 1050 clean |
| Noise ratio in content | ~80% nav | 0% |
| Estimated Q&A tokens (per question) | 3000-5000 | 500-1500 |
| Effective improvement | baseline | **5-10x cheaper, more accurate** |
