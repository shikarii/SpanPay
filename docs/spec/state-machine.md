# State Machine Specification

The state machine prevents illegal transitions and handles high concurrency
of payment events without corruption.

## State Transition Table

| Current State | Triggering Event | Source | Target State |
|---|---|---|---|
| NONE | PAYMENT_CREATE | API | CREATED |
| CREATED | AUTH_SUCCESS | PSP API | AUTHORIZED |
| CREATED | AUTH_FAILURE | PSP API | FAILED |
| AUTHORIZED | CAPTURE_REQUEST | API | PROCESSING |
| PROCESSING | CAPTURE_SUCCESS | Webhook | CAPTURED |
| AUTHORIZED | VOID_REQUEST | API | VOIDED |
| CAPTURED | REFUND_REQUEST | API | REFUNDED_PENDING |
| REFUNDED_PENDING | REFUND_SUCCESS | Webhook | REFUNDED |
| CAPTURED | DISPUTE_CREATED | Webhook | DISPUTED |
| DISPUTED | DISPUTE_WON | Webhook | CAPTURED |

## Terminal States

VOIDED, FAILED, and REFUNDED (full) are terminal.
No further CAPTURE or DISPUTE triggers should be processed once these are reached.

## Idempotent Transitions

If an event would move the machine to its current state
(e.g. a duplicate CAPTURE_SUCCESS webhook), the machine must:

1. Acknowledge it as success.
2. Perform no state or ledger changes.
3. Return the same response as the original transition.

## Out-of-Order Event Handling

If a CAPTURE_SUCCESS webhook arrives before the synchronous AUTH_SUCCESS response
has committed (due to network lag), the state machine uses a High-Water Mark approach:

1. Transition to the most advanced valid state (CAPTURED).
2. Skip intermediate states (AUTHORIZED).
3. Post all required ledger entries for skipped states atomically.

Example: if `dispute.created` arrives before `payment.succeeded` is processed,
the state machine must create the payment in DISPUTED state and correctly
sequence the ledger entries as if the payment had succeeded and then been disputed.

## Concurrency

- State transitions must use optimistic locking (version column or SELECT FOR UPDATE).
- The machine rejects concurrent transitions that would violate the transition table.
- Webhook workers should use payment_id partitioning to ensure serial processing per payment.
