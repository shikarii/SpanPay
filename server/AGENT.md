# AGENT.md

Rules for `server/`, the Go orchestration service.

## 1. Purpose

`server/` owns:

- HTTP APIs
- deterministic payment state transitions
- provider adapter seams
- idempotency handling
- webhook inbox processing
- outbox publication
- reconciliation orchestration

## 2. Boundaries

- Do not move payment truth into the client.
- Do not treat provider responses as final balances or final settlement.
- Keep provider-specific logic behind adapter interfaces.
- Keep ledger invariants testable without HTTP or database I/O when possible.

## 3. Design rules

- Prefer small domain packages over broad service god-objects.
- Keep request handlers thin.
- Express impossible payment transitions explicitly and test them.
- Avoid hidden side effects in ledger code.

## 4. Size guidance

- target under 250 LOC
- soft limit 350 LOC
- hard limit 500 LOC
