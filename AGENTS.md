# AGENTS.md

Repository operating rules for coding agents.

## 0. Prime Directive

- Move quickly, but do not make payment-system architecture changes silently.
- Prefer correctness, auditability, and provider portability over short-term convenience.
- If a requested change conflicts with the repository contracts below, stop and realign before proceeding.
- Never introduce or modify licensing terms without explicit maintainer approval.
- Never use emojis in repository artifacts.

## 1. Required Repository Contracts

These artifacts must remain present:

- `AGENTS.md`
- `.github/pull_request_template.md`
- `docs/architecture.md`
- `docs/adr/0000-template.md`

## 1.1 Nested Agent Instructions Precedence

- Always discover and read any closer `AGENT.md` or `AGENTS.md` in the subtree you are changing.
- Treat the nearest inner agent file as authoritative for that subtree.
- Apply root rules as defaults; let nested rules tighten them.
- If instructions conflict and cannot be safely reconciled, stop and ask for clarification.

## 2. Architecture Decision Records

Write an ADR under `docs/adr/` whenever a change involves any of the following:

- A new public API route or change to an existing request or response shape
- A new ledger posting rule or change to a financial invariant
- A new provider adapter contract or reconciliation workflow
- A new storage engine, queue, transport, or schema technology
- A change to idempotency semantics, webhook processing, or outbox behavior
- A significant change to import rules or cross-package dependencies

Use `docs/adr/0000-template.md` as the starting point. Number sequentially
(`0001-short-slug.md`, `0002-short-slug.md`, ...).

Keep `docs/architecture.md` in sync whenever package layout, payment flow boundaries,
or authoritative data ownership changes.

## 3. Module Ownership and Boundaries

### Package layout

```txt
client/      React + TypeScript operations console.
             Owns dashboards, workflow views, docs surfaces, and browser UX.

server/      Go orchestration service.
             Owns HTTP APIs, payment state transitions, provider adapters, webhook ingress,
             idempotency, outbox publication, and reconciliation jobs.

db/          PostgreSQL schema and migrations.
             Owns immutable ledger tables, payment persistence, inbox/outbox tables,
             and database-level invariants.

docs/        Architecture notes, ADRs, and product/technical planning.
```

### Import and dependency rules

- `client/` must not import from `server/`.
- `server/` must not depend on client runtime code.
- `db/` is authoritative for persisted schema, not generated from client code.
- The client is never the source of truth for payment state, balances, or reconciliation outcomes.
- Provider-specific response shapes must remain behind explicit adapter boundaries.

## 4. Architecture Rules

### Sovereign orchestration direction

This repository is intentionally treating the orchestration service as a sovereign financial system:

- no provider is treated as the final source of truth
- internal state transitions must remain deterministic
- money movement is modeled through immutable double-entry postings
- provider responses are provisional until reconciliation confirms them

### Reliability direction

- synchronous request handling and asynchronous workflows must remain explicitly separated
- webhook ingress should persist first and process second
- side effects should leave durable traces through an outbox-style pattern
- idempotency must prevent duplicate financial effects, not just duplicate HTTP responses

### Token efficiency is a repository constraint

Token usage in LLM-assisted development is a first-class engineering concern in this repository.

The working rule is:

- load the minimum context needed to act correctly
- expand context only when evidence says it is needed
- summarize and externalize stable knowledge instead of replaying it

Required practices:

- prefer targeted reads over broad repository dumps
- keep stable architecture and invariants in docs under `docs/`
- break large work into explicit discover, plan, implement, and verify stages
- avoid blind retry loops that do not incorporate concrete failures

## 5. Quality and Enforcement

### Testing expectations

Every non-trivial feature should add tests appropriate to its layer:

- `server/`: state transitions, ledger invariants, idempotency, webhook processing, reconciliation helpers
- `client/`: pure state logic and isolated UI behavior where practical
- `db/`: migration-level invariants and schema constraints where practical

Do not write shallow tests that only restate implementation details.
When a concrete failure is discovered, add tests for that exact failure and nearby cases of the same family when practical.

### Adversarial self-review

For every substantial change:

- try malformed provider payloads and idempotency collisions
- check negative amounts, invalid currencies, and impossible state transitions
- check whether the change makes future reconciliation harder
- check whether provider portability got worse

### Entropy prevention

- No `utils` module names for new code; name by responsibility.
- No `TODO` without an issue reference.
- No new public API shape without corresponding docs and validation updates.

## 6. File Size Guidelines

These are repository-wide defaults unless a nested agent file is stricter:

- Target: 100-250 LOC
- Soft limit: 350 LOC
- Hard limit: 500 LOC

If a file exceeds 350 LOC, split it by responsibility unless there is a strong reason not to.

## 7. Human-Facing Wording for Issues and PRs

- Use specific, human language over boilerplate.
- State intent, impact, and tradeoffs clearly.
- Prefer real line breaks; never write literal `\n` sequences in GitHub issue/PR text.
- Call out validation and residual risk explicitly.
