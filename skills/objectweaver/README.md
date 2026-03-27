# ObjectWeaver Skills Pack

Purpose: a compact, agent-consumable knowledge base for implementing ObjectWeaver flows quickly and safely.

## Start Here

1. Read `manifest.json` for machine-readable routing.
2. Read `core.md` for platform model and terminology.
3. Use the recipe files for concrete implementation tasks.
4. Copy from `templates/` and adjust.

## Canonical Defaults

- Preferred REST endpoint: `/api/objectGen`
- Schema generation endpoint: `/api/textToWeaver` (development mode)
- Health endpoint: `/health`
- Typical local base URL in docs: `http://localhost:2008`

## Notes on Docs Inconsistency

Some examples show `/objectGen` (without `/api`) or alternate response shapes. Treat `/api/objectGen` as the default and keep parsers tolerant.

## Skill Files

- `core.md` — architecture, data model, key fields.
- `schema-reference.md` — high-value definition fields and constraints.
- `recipes/*.md` — task-specific implementation playbooks.
- `ops/*.md` — streaming, batch, throughput manager.
- `guardrails/pitfalls.md` — anti-patterns and failure prevention.
- `templates/*.json` — ready-to-edit request payload patterns.
