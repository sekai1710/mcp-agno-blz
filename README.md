# Agno MCP docs — benchmark: HTML scraper vs BLZ + llms-full.txt

Confronto tecnico fra due approcci per esporre la documentazione [Agno](https://docs.agno.com) come MCP server a Claude (Code/Desktop) e altri client MCP.

- **Approccio A**: `mcp-agno` (wrapper community) — scraping HTML live di `docs.agno.com` + parsing `BeautifulSoup` + `html2text`.
- **Approccio B**: [BLZ](https://github.com/outfitter-dev/blz) (MCP generico OSS) che indicizza `llms-full.txt` ufficiale di Agno con Tantivy FTS.

## TL;DR

| Metrica | mcp-agno (scraper) | BLZ |
|---|---|---|
| Top-result corretto su 3 query | 0/3 | 3/3 |
| Snippet inclusi nei search results | mai | sempre |
| Cache | RAM, persa al restart | SQLite persistente |
| Latency search | 500–2000 ms (HTTP live) | ~40 ms (locale) |
| Token medi per Q&A completa | 3000–5000 | 500–1500 |
| Efficienza | baseline | **5–10x** |

## Il problema

Il MCP `mcp-agno` ha tre difetti strutturali che bruciano token e degradano la qualità delle risposte:

1. **Scraping HTML grezzo**: ogni pagina di `docs.agno.com` viene scaricata e convertita in Markdown via `html2text`. Il problema è che la conversione include sempre tutto il template Mintlify — sidebar nav, header, footer, link "⌘K Ask AI", menù duplicati. Circa l'80% dell'output è rumore, non contenuto utile.

2. **Search keyword-only, niente FTS**: `search_agno_docs` ritorna solo gli slug, niente snippet, niente scoring semantico. Sei costretto a un secondo round-trip con `get_agno_page` per leggere il contenuto, e spesso la risposta torna troncata.

3. **Cache in-memory**: ogni restart del server perde tutto e rifa lo scraping da capo.

## La soluzione

Agno pubblica già due URL pensati apposta per gli LLM:

- `https://docs.agno.com/llms.txt` — indice di 3290 pagine, una riga per ognuna
- `https://docs.agno.com/llms-full.txt` — 9.6 MB di tutto il contenuto docs in Markdown puro, zero HTML, zero nav

[BLZ](https://github.com/outfitter-dev/blz) è un MCP server (Rust, OSS, outfitter-dev) che indicizza file `llms.txt` con Tantivy (FTS vero), restituisce snippet con citation della riga esatta e tiene tutto in SQLite locale.

## Benchmark — 3 casi d'uso

Setup: entrambi i MCP attivi nella stessa sessione Claude Code, stessa query, output catturato grezzo. Token stimati a 1 token ≈ 4 char.

Raw output dei tre casi: vedi [`benchmark/raw-metrics.md`](benchmark/raw-metrics.md).

### Caso 1 — "workflow step parallel execution"

**mcp-agno**: la prima chiamata restituisce 5 slug senza contenuto, ~150 token, top-result non pertinente (propone "sequence of steps" invece di "parallel"). Devi fare una seconda chiamata per leggere il contenuto, e quella torna ~2500 token troncata, con l'80% di sidebar nav prima del codice utile. Totale: ~2650 token, risposta incompleta, ranking sbagliato.

**BLZ**: una chiamata, 5 risultati pertinenti con snippet inline. Top-result è "Parallel Workflow" esatto. ~310 token, spesso il follow-up non serve. Se vuoi il contenuto completo, una seconda chiamata mirata aggiunge ~260 token di Markdown pulito. Totale: ~575 token, risposta giusta al primo colpo.

**4.6x meno token, ranking corretto.**

### Caso 2 — "agent memory persistent storage"

**mcp-agno**: top 3 risultati sono pagine di "in-memory storage", concetto opposto a "persistent". La pagina corretta è al 4° posto. Niente snippet, devi aprire più pagine prima di capire qual è quella giusta.

**BLZ**: top-result è "Agent With Persistent Memory" esatto. Bonus: il 4° risultato mostra direttamente la tabella comparativa "Memory vs Storage" dentro lo snippet — il dubbio risolto senza nemmeno fare il fetch.

**~9x meno token per arrivare alla risposta giusta.**

### Caso 3 — "team coordinator delegation"

**mcp-agno**: zero risultati pertinenti su cinque. Tutti match keyword "team" su pagine random (deploy, observability, reasoning). La pagina ufficiale `/teams/delegation` non compare nemmeno nei primi cinque.

**BLZ**: top-result è `/teams/delegation` esatto. Gli altri quattro sono sotto-sezioni dello stesso concetto: Tasks Mode, Route Mode, Structured Input, Developer Resources. Tutti pertinenti.

**Su questa query mcp-agno è semplicemente inutilizzabile.**

## Setup BLZ per Agno docs

```bash
# macOS
brew tap outfitter-dev/tap
brew install blz

# Linux/Windows: vedi GitHub releases di BLZ
# https://github.com/outfitter-dev/blz/releases

# Aggiungi Agno docs come source
blz add agno https://docs.agno.com/llms-full.txt
# Output atteso: ✓ Added agno (10034 headings, 291990 lines)

# Test CLI
blz "workflow parallel" -s agno
```

Config MCP (Claude Code `.claude.json` o Claude Desktop `claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "agno-docs": {
      "type": "stdio",
      "command": "/opt/homebrew/bin/blz",
      "args": ["mcp-server", "--quiet"]
    }
  }
}
```

Restart client, verifica `/mcp` per la connessione. Tool esposti: `find` (search/get/toc) e `blz` (gestione sources).

Aggiornare l'indice quando le docs cambiano: `blz refresh agno` (automatizzabile via cron).

## Replay del benchmark

Per replicare i test:

1. Installa entrambi i MCP nello stesso ambiente
2. Lancia le tre query da `benchmark/queries.txt`
3. Confronta gli output con quelli salvati in `benchmark/raw-metrics.md`

## Onestà tecnica

Lo scraping HTML resta utile quando il sito target **non pubblica** un `llms.txt`. Ma `docs.agno.com` lo pubblica già, quindi ricavare il contenuto via HTML è solo lavoro extra che produce output peggiore.

Lo stesso approccio BLZ funziona per qualsiasi sito con `llms.txt`: Anthropic, Cloudflare, Stripe, e in generale tutti i siti Mintlify-based.

## Credits

- [Agno team](https://agno.com) per pubblicare `llms-full.txt` curato ufficiale
- [outfitter-dev/blz](https://github.com/outfitter-dev/blz) per il tool di indicizzazione
- Community Agno per i tentativi iniziali di MCP wrapper, che hanno motivato questa analisi

## License

MIT — vedi [LICENSE](LICENSE).
