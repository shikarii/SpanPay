# ADR 0003: Webhook Inbox, Outbox Publication, and Reconciliation Are First-Class Workflows

- Status: Accepted
- Date: 2026-03-30

## Context

Payment systems combine synchronous request paths with asynchronous provider events, settlement reports, and delayed reversals.
Losing or partially processing those events is unacceptable in a system that is expected to survive audit.

## Decision

SpanPay adopts these workflow boundaries from day one:

- provider webhooks land in a durable inbox before downstream processing
- internal side effects should be recorded through an outbox-style durable publication seam
- reconciliation is a dedicated background concern, not an afterthought on request handlers

## Consequences

Upsides:

- webhook durability is explicit
- side effects can be replayed or retried more safely
- reconciliation gets a dedicated place in the architecture instead of living as ad hoc scripts

Downsides:

- there are more moving pieces than a thin request-response service
- developers need to think clearly about sync versus async effects

Follow-on constraints:

- request handlers should stay narrow
- inbox and outbox tables must stay auditable
- reconciliation mismatches should surface as actionable discrepancies, not silent drift

## Alternatives Considered

- processing webhooks inline without durable staging
- triggering side effects directly from request handlers without an outbox seam
