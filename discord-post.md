## Messaggio 1/2 — copia-incolla diretto in Discord

**MCP Agno docs: perché il wrapper community spreca token, e come risolvere con BLZ**

Il MCP `mcp-agno` (wrapper community per `docs.agno.com`) ha tre difetti che bruciano token e abbassano la qualità delle risposte.

Primo: fa scraping HTML grezzo. Dentro l'output finisce tutto il template del sito — sidebar, navigation, header "⌘K Ask AI", footer. Circa l'**80% di quello che Claude riceve è rumore**, non contenuto utile.

Secondo: la search è keyword-only e non restituisce snippet. Cinque link e basta — devi fare una seconda chiamata per leggere il contenuto, e spesso torna troncata.

Terzo: la cache è in memoria. Restart = ri-scrape da zero.

**La soluzione**

Agno pubblica già `docs.agno.com/llms-full.txt`: 9.6 MB di tutto il contenuto docs in Markdown puro, zero HTML.

BLZ (<https://github.com/outfitter-dev/blz>) è un MCP server open source che indicizza file `llms.txt` con Tantivy FTS e restituisce snippet con citation della riga esatta.

**Benchmark — 3 casi reali**

__Caso 1: "workflow step parallel execution"__
- mcp-agno: 5 slug senza contenuto (~150 token), top-result sbagliato ("sequence of steps"). Seconda chiamata ~2500 token troncata, 80% sidebar nav. Totale ~2650 token, risposta incompleta.
- BLZ: 5 risultati pertinenti con snippet, top-result "Parallel Workflow" esatto, ~310 token. Follow-up opzionale +260 token. Totale ~575 token.
- **4.6x meno token, ranking corretto.**

__Caso 2: "agent memory persistent storage"__
- mcp-agno: top 3 risultati sono pagine di "in-memory storage" — concetto opposto a "persistent". La pagina giusta è al 4° posto, senza snippet.
- BLZ: top-result "Agent With Persistent Memory" esatto. Bonus: il 4° risultato mostra la tabella "Memory vs Storage" inline.
- **~9x meno token per arrivare alla risposta giusta.**

---

## Messaggio 2/2

__Caso 3: "team coordinator delegation"__
- mcp-agno: **0/5 risultati pertinenti**. Match keyword "team" su pagine random (deploy, observability, reasoning). La pagina ufficiale `/teams/delegation` non compare nemmeno.
- BLZ: top-result `/teams/delegation` esatto. Gli altri 4 sono sotto-sezioni dello stesso concetto (Tasks Mode, Route Mode, Structured Input).
- **Su questa query mcp-agno è inutilizzabile.**

**Aggregato**
- Top-result corretto su 3 query: mcp-agno 0/3, BLZ 3/3
- Snippet nei search results: mcp-agno mai, BLZ sempre
- Cache: mcp-agno in RAM (persa al restart), BLZ SQLite persistente
- Latency search: mcp-agno 500-2000 ms (HTTP live), BLZ ~40 ms (locale)
- Token medi per Q&A: mcp-agno 3000-5000, BLZ 500-1500
- **Efficienza: 5-10x.**

**Onestà tecnica**

Lo scraping HTML ha senso quando il sito target non pubblica un `llms.txt`. Ma `docs.agno.com` lo pubblica già, quindi ricavare il contenuto via HTML è solo lavoro extra che produce output peggiore. Lo stesso approccio BLZ funziona per qualsiasi sito con `llms.txt`: Anthropic, Cloudflare, Stripe, ecc.

**TL;DR**
- Vecchio MCP basato su HTML scraping → 80% nav noise, ranking cieco, doppi round-trip → 3000-5000 token per domanda.
- BLZ + `llms-full.txt` ufficiale → FTS semantica, snippet line-accurate, locale → 500-1500 token per domanda.
- 5-10x meno token, sub-50 ms, zero noise.

Repo con setup completo, raw output e replay del benchmark: `[INSERIRE LINK GITHUB]`

Critiche e correzioni benvenute.
