# SpanPay

SpanPay is a provider-agnostic payment orchestration service scaffold.

The current repo baseline assumes:

- `server/` in Go for the orchestration API, ledger core, provider adapters, and async workflows
- `client/` in React + TypeScript for an operations-facing console
- `db/` in PostgreSQL migrations for the immutable ledger, idempotency, inbox/outbox, and payment state tables

## Architectural intent

SpanPay is being shaped around a few hard rules:

- no PSP is treated as the final source of truth
- internal balances derive from immutable ledger entries
- payment state transitions stay deterministic
- idempotency, webhook durability, and reconciliation are first-class concerns

See [docs/architecture.md](docs/architecture.md) for the current system shape.

## Getting started

Prerequisites:

- Go 1.24+
- Node.js 22+
- Docker or Podman for local PostgreSQL

Local commands:

```bash
make db-up
make client-install
make typecheck
make test
make build
```

Useful endpoints after the first server run:

- API health: `http://127.0.0.1:6940/healthz`
- Client dev UI: `http://localhost:6941`

## Project structure

```txt
client/   Operations console and browser UX
server/   Go orchestration service
db/       PostgreSQL migrations
docs/     Architecture notes and ADRs
```

## Workflow

- branch from `develop`
- keep scope tight
- run validation locally
- open a PR to `develop` with a linked issue URL
- merge after required checks pass

See [CONTRIBUTING.md](CONTRIBUTING.md), [SECURITY.md](SECURITY.md), and [AGENTS.md](AGENTS.md).
