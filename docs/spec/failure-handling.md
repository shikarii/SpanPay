# Failure Handling Rules

The orchestrator must handle ambiguous failure states where the outcome at the provider is unknown.

## Failure Scenarios

### Provider Success, Local DB Failure

The PSP captures the payment but SpanPay crashes before updating the database.

- The reconciliation engine detects the orphan PSP entry.
- Auto-heals by updating internal state to SUCCESS and posting missing ledger entries.
- Logged as a reconciliation auto-heal event.

### Local Success, Provider Timeout

The request to the PSP times out with no response.

1. Transition internal state to UNKNOWN.
2. Poll the PSP's status API via `QueryStatus`.
3. If PSP has no record: safe to retry on same or different provider.
4. If PSP says SUCCESS: update local state to match.
5. If PSP says FAILED: update local state to FAILED.

### Duplicate Webhook

- The `webhook_inbox` table unique constraint on `provider_evt_id` prevents re-processing.
- Worker ignores the duplicate event.
- Returns 200 OK to the provider to stop retries.

### Out-of-Order Dispute

If `dispute.created` webhook arrives before `payment.succeeded` is processed:

1. State machine creates payment in DISPUTED state.
2. Correctly sequences ledger entries as if payment had succeeded and then been disputed.
3. All intermediate entries are posted atomically.

## Hard Constraints

These are non-negotiable and must be enforced by code review and automated testing.

1. **Append-Only Ledger**: No SQL UPDATE or DELETE statements on `ledger_entries`.
2. **Balanced Transactions**: The ledger engine wraps all entries for a transaction in a DB transaction and verifies they sum to zero before committing.
3. **Internal Ledger Is Source of Truth**: If the PSP says balance is $100 and ledger says $90, the $10 is an error to investigate, not a reason to adjust the ledger.
4. **No Side Effects in DB Transactions**: Provider API calls and message queue publishes happen after the DB transaction commits, or via the Outbox pattern.
5. **Strict Idempotency**: Every mutating API endpoint requires an idempotency key. No key means no processing.

## Production Pitfalls

These are subtle issues that surface in real payment systems:

1. **Rounding Gaps**: Multi-currency transactions may use different FX rates between PSP and internal system. Book explicitly to `Currency_Exchange_Loss` account.
2. **Settlement Delays**: Distinguish between "available at processor" and "liquid in bank."
3. **Partial Capture Limits**: Many networks allow only one capture per auth. Remaining amount is released.
4. **Chargeback Fees**: Disputes include non-refundable fees ($15-$25). Must be tracked separately or the ledger will never match bank statements.
5. **Clock Skew**: Webhook signature verification includes timestamp checks. Server clock drift > 5 minutes breaks all webhook verification.
