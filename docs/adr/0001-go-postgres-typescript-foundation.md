# ADR 0001: Go Server, Postgres, and TypeScript Console Foundation

- Status: Accepted
- Date: 2026-03-30

## Context

SpanPay needs a practical baseline stack that supports:

- deterministic server-side payment logic
- strong transactional persistence
- a browser-hosted operations console

The service also needs room for async workflows like webhook ingestion and reconciliation without pushing core financial logic into the browser.

## Decision

SpanPay uses:

- Go for the orchestration service in `server/`
- PostgreSQL for authoritative persistence and ledger data in `db/`
- React + TypeScript for the internal operations console in `client/`

## Consequences

Upsides:

- Go is a strong fit for explicit service boundaries, concurrency, and operational simplicity
- PostgreSQL gives us strong transactional semantics and constraint support
- the client remains useful for workflow and visibility without owning payment truth

Downsides:

- there is an explicit language boundary between server and client
- schema and API contracts must stay deliberate because they are not auto-shared by default

Follow-on constraints:

- the server remains authoritative for payment state and balances
- schema changes need migrations and docs updates

## Alternatives Considered

- TypeScript for the server
- a document database instead of PostgreSQL
