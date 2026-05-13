---
description: Refresh BLZ index for Agno docs from llms-full.txt
---

Esegui il refresh dell'indice BLZ per Agno docs e riporta il risultato.

Steps:
1. Run `blz refresh agno` via Bash tool.
2. Capture exit code + output.
3. Se exit 0: report compatto con "Indice aggiornato" + numero di righe/heading se nel output.
4. Se exit != 0: report errore esatto + suggerisci `blz doctor` per diagnostic.
5. NON aggiornare altri sources, NON toccare `.claude.json` config.

Output finale al user: una riga sintetica del tipo "✓ Agno docs refresh — N headings, M lines, latest commit YYYY-MM-DD" oppure "✗ Refresh failed: <error>".
