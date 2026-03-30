# AGENT.md

Rules for `client/`, the operations console.

## 1. Purpose

`client/` owns:

- dashboards and workflow surfaces
- browser UX
- internal visibility into payment, ledger, and reconciliation state

## 2. Boundaries

- Do not implement payment truth or ledger math in React state.
- Keep browser state separate from durable server state.
- Prefer pure helper modules for any non-trivial view logic.

## 3. Design rules

- Keep React components thin when state can live in dedicated modules.
- Test pure formatting and state logic directly.
- Avoid leaking provider-specific transport details throughout the component tree.
