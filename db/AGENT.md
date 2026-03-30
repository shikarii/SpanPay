# AGENT.md

Rules for `db/`, the PostgreSQL schema and migrations.

## 1. Purpose

`db/` owns:

- migration history
- authoritative table shape
- database-level invariants for ledger and orchestration persistence

## 2. Boundaries

- Do not weaken financial constraints casually.
- Prefer additive migrations over destructive rewrites.
- Keep provider raw payload storage explicit when needed for auditability.

## 3. Design rules

- Use precise names over vague generic tables.
- Encode invariants in constraints when practical.
- Keep migrations reviewable and human-readable.
