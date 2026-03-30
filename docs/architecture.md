# Architecture

## Product goal

SpanPay is a provider-agnostic payment orchestration service with an immutable internal ledger.

The intended direction is:

- a Go orchestration core that owns payment state transitions and async workflows
- a PostgreSQL-backed immutable double-entry ledger and operational data model
- a React + TypeScript operations console for internal users
- provider adapters that normalize PSP-specific APIs, webhooks, and error taxonomies

## Working principles

- no external payment provider is the final source of truth
- provider responses are provisional until internal state and reconciliation agree
- balances are derived from journal entries, not stored as mutable scalar fields
- synchronous request handling and asynchronous workflows remain explicitly separated
- webhook ingress persists before processing
- outbox publication and reconciliation remain first-class seams
- repo docs and ADRs are the low-token memory layer for future work

## Package layout

- `client/`: React + TypeScript operations console
- `server/`: Go orchestration API, ledger logic, provider adapters, webhook ingress, and async workflows
- `db/`: PostgreSQL migrations and schema contracts
- `docs/`: architecture notes and ADRs

## Core payment model

### Internal truth

The orchestration service should own:

- payment intents and attempts
- deterministic payment state transitions
- idempotency handling
- immutable ledger postings
- webhook inbox persistence
- outbox publication of internal side effects
- reconciliation jobs and discrepancy handling

### External systems

Providers are integration partners, not system authorities.

They can report:

- synchronous authorization or capture responses
- asynchronous webhooks
- settlement and payout reports
- disputes and returns

But SpanPay is expected to decide:

- what the canonical payment state is
- what the canonical merchant balance is
- whether external and internal records are reconciled

## Data ownership

PostgreSQL is the authoritative system of record for:

- `payments`
- `payment_attempts`
- `ledger_accounts`
- `ledger_transactions`
- `ledger_entries`
- `idempotency_keys`
- `webhook_inbox`
- `outbox_events`
- reconciliation tracking tables

The first migration establishes the baseline schema, but later migrations should deepen constraints and auditability rather than relaxing them.

## Synchronous vs asynchronous paths

### Synchronous path

The request path should stay narrow:

1. validate request
2. check or reserve idempotency
3. create or advance internal state
4. call a provider adapter when needed
5. persist the resulting internal state
6. return a deterministic response

### Asynchronous path

The background path should stay explicit:

1. ingest and persist provider webhooks
2. process inbox items through the state machine
3. write ledger and outbox effects transactionally
4. run reconciliation against provider or bank artifacts
5. surface discrepancies for repair and audit

## Near-term plan

- keep the Go server small and explicit
- deepen ledger primitives before broad provider coverage
- keep the client focused on operations workflows, not user-facing checkout
- encode major architectural decisions as ADRs instead of rediscovering them in chat

## Specification documents

Detailed engineering specifications derived from the design research:

- [spec/payment-flows.md](spec/payment-flows.md) - card, ACH, and dispute flow step-by-step sequences
- [spec/data-model.md](spec/data-model.md) - complete table definitions, constraints, and indexing strategy
- [spec/state-machine.md](spec/state-machine.md) - state transition table, terminal states, out-of-order handling
- [spec/ledger-rules.md](spec/ledger-rules.md) - chart of accounts and deterministic posting rules per event
- [spec/provider-interface.md](spec/provider-interface.md) - unified provider interface, error taxonomy, retry logic
- [spec/webhook-processing.md](spec/webhook-processing.md) - inbox pattern, signature verification, async workers
- [spec/idempotency.md](spec/idempotency.md) - key strategy, request hashing, replay and conflict handling
- [spec/reconciliation.md](spec/reconciliation.md) - three-way match algorithm, tolerance rules, drift detection
- [spec/failure-handling.md](spec/failure-handling.md) - ambiguous failure scenarios, hard constraints, production pitfalls

## Architecture decision records

- [adr/0000-template.md](adr/0000-template.md)
- [adr/0001-go-postgres-typescript-foundation.md](adr/0001-go-postgres-typescript-foundation.md)
- [adr/0002-sovereign-ledger-and-provider-orchestration.md](adr/0002-sovereign-ledger-and-provider-orchestration.md)
- [adr/0003-inbox-outbox-and-reconciliation-workflows.md](adr/0003-inbox-outbox-and-reconciliation-workflows.md)
