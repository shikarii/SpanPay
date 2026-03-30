# Idempotency Specification

Idempotency is the primary safeguard against the double-spend problem.
Every API endpoint must accept an Idempotency-Key header.

## Key Strategy

### Format

- Client provides an `Idempotency-Key` header on every mutating request.
- Keys are scoped per `merchant_id` to prevent cross-business collisions.
- No key, no request processing (return 400).

### Request Hashing

To prevent key reuse for different payloads:

1. Hash the request body with SHA-256.
2. Store the hash alongside the key.
3. If key matches but body hash differs, return `IdempotencyConflictError` (422).

### Storage

- Fast path: distributed cache (Redis with SET NX) for locking during processing.
- Durable path: `idempotency_keys` SQL table for long-term deduplication.
- Retention: 30 days recommended, governed by `expires_at` column.

## Request Lifecycle

### First Request

1. Check if key exists in cache or database.
2. If not found: set key to IN_PROGRESS in cache.
3. Process the request normally.
4. On completion: store final HTTP status and response body.
5. Update key status to COMPLETED.

### Duplicate During Processing

If a second request arrives while the first is still IN_PROGRESS:

- Return 409 Conflict.
- Client should retry after a short delay.

### Duplicate After Completion

If the key exists and status is COMPLETED:

1. Verify request body hash matches.
2. If match: return the cached response (same HTTP status and body).
3. If mismatch: return IdempotencyConflictError (422).

No side effects are triggered. No provider calls, no ledger entries, no state changes.

## Scope

Idempotency applies to all mutating API endpoints:

- POST /payments (create payment intent)
- POST /payments/:id/capture
- POST /payments/:id/refund
- POST /payments/:id/void
