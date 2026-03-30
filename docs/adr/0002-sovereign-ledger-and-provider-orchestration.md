# ADR 0002: SpanPay Owns the Canonical Ledger and Payment State Machine

- Status: Accepted
- Date: 2026-03-30

## Context

The reference design for SpanPay explicitly treats PSPs as provisional sources rather than authoritative ones.
If provider responses are treated as the sole truth, duplicate effects, drift, and reconciliation failures become much harder to reason about.

## Decision

SpanPay owns:

- the canonical payment intent lifecycle
- the immutable double-entry ledger
- the normalized provider-attempt record
- the final internal interpretation of payment success, failure, refund, dispute, and return events

Provider adapters normalize external payloads, but they do not replace the internal state machine or ledger.

## Consequences

Upsides:

- balance and state invariants live under one authority
- provider portability stays possible
- reconciliation has a stable internal baseline

Downsides:

- the service must carry more domain complexity up front
- provider integrations need careful mapping instead of thin pass-through wrappers

Follow-on constraints:

- balances are derived from entries, not stored as mutable truth
- corrections happen through compensating entries, not mutation of historical facts

## Alternatives Considered

- treating each provider as the authoritative source for its own transactions
- storing mutable balances without a journaled ledger
