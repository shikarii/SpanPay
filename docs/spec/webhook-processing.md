# Webhook Processing Specification

Webhooks are unreliable, asynchronous, and out-of-order.
The orchestrator treats them as raw data ingress, not instructions.

## Ingestion Pipeline (Inbox Pattern)

### Step 1: Fast Ack

The webhook endpoint must:

1. Verify signature.
2. Persist raw payload to `webhook_inbox` table.
3. Return 200 OK within 1-2 seconds.

Any longer risks the provider timing out and retrying, creating duplicate events.

### Step 2: Signature Verification

- Use the provider's shared secret and HMAC-SHA256.
- Verification must happen against the raw body bytes.
- Do not parse JSON before verification (whitespace changes break the signature).
- Include timestamp check with a 5-minute tolerance to guard against replay attacks.

Clock skew warning: if the server clock drifts more than 5 minutes from the PSP's clock,
all webhooks will fail verification.

### Step 3: Persistence

- Store the raw payload in `webhook_inbox` with status PENDING.
- The unique constraint on `provider_evt_id` is the final defense against duplicate webhooks.
- If the insert conflicts (duplicate), acknowledge 200 OK and skip.

### Step 4: Deduplication

- Database-level unique constraint on `provider_evt_id` handles exact duplicates.
- The state machine handles semantic duplicates (same event type for same payment).

## Async Processing

### Worker Strategy

- A pool of background workers pulls PENDING items from `webhook_inbox`.
- Workers use payment_id partitioning (hash of payment_id modulo worker count).
- This ensures one payment is handled by one worker at a time, preventing race conditions.

### Processing Steps

1. Worker picks up inbox item.
2. Calls provider adapter's `NormalizeEvent` to produce an `InternalEvent`.
3. Applies the event to the state machine.
4. If the state machine accepts the transition, writes ledger entries and outbox events atomically.
5. Marks inbox item as PROCESSED.

### Ordering Logic

Do not rely on timestamps for ordering.
The state machine itself handles ordering:

- If an "old" event (e.g. `invoice.created`) arrives after a "new" event (`invoice.paid`),
  the state machine rejects the transition because PAID is a more advanced state.
- The High-Water Mark approach automatically resolves out-of-order delivery.

### Poison Events

If an event fails processing 5 times:

1. Mark inbox item as FAILED.
2. Move to Dead Letter Queue (separate table or queue).
3. Alert operations team for manual investigation.
4. Do not block processing of other events.
