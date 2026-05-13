---
description: Refresh BLZ index for ALL local docs sources (agno, openrouter, etc.)
---

Esegui il refresh di tutte le sources BLZ indicizzate localmente e riporta il risultato.

Steps:
1. Run `blz refresh --all` via Bash tool.
2. Capture exit code + output.
3. Se exit 0: report compatto per ciascuna source aggiornata, formato `✓ <alias> — N headings, M lines`.
4. Se exit != 0: report errore esatto + suggerisci `blz doctor` per diagnostic.

Per refresh di UNA sola source: `blz refresh <alias>` (esempio `blz refresh openrouter`).

NON modificare config MCP/skill, NON aggiungere/rimuovere sources qui.

Output finale al user: tabella compatta delle sources aggiornate con metriche, oppure errore esatto.
