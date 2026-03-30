# Payment Flow Specifications

Deterministic step-by-step sequences for each supported payment method.
Each flow identifies risk points and required verification steps.

## Card Payment Flow: Authorization, Capture, Settlement

The dual-message card flow separates intent to charge from actual funds movement.

### Step 1: Authorization

1. Merchant system sends request to SpanPay orchestrator.
2. Orchestrator generates internal UUID, records CREATED state.
3. Ledger posts a Pending entry (Debit: Settlement_Pending, Credit: Merchant_Wallet_Pending).
4. Orchestrator calls PSP authorization endpoint via provider adapter.
5. PSP communicates with card network and issuer to verify funds and fraud risk.
6. On success: state transitions to AUTHORIZED, response returned to merchant.
7. On failure: state transitions to FAILED (terminal).

Risk: PSP timeout leaves outcome unknown. See failure-handling.md for UNKNOWN state handling.

### Step 2: Capture

1. Merchant triggers Capture request after order fulfillment.
2. Orchestrator validates state is AUTHORIZED, transitions to PROCESSING.
3. Calls PSP capture endpoint via provider adapter.
4. PSP emits capture confirmation webhook asynchronously.
5. Webhook lands in inbox, async worker transitions state to CAPTURED.
6. Ledger posts capture entries (moves pending to available).

Constraint: many card networks allow only one capture per authorization.
Partial capture of $50 on a $100 auth typically releases the remaining $50 immediately.

### Step 3: Settlement

1. Background process where issuer moves funds to acquirer.
2. Not visible via real-time APIs.
3. Verified by the reconciliation engine against PSP payout reports.

## ACH Debit Flow with Delayed Return

ACH transactions are pull payments. Funds are not guaranteed until the return window closes.

### Step 1: Initiation

1. Orchestrator initiates debit request via PSP.
2. PSP accepts and emits PENDING status.
3. Orchestrator must NOT mark as successful yet.

### Step 2: Batch Processing

1. ODFI groups transaction into NACHA file.
2. File sent to ACH Operator (Federal Reserve or TCH).

### Step 3: Provisional Settlement

1. Funds deposited into merchant account within 1-3 business days.
2. Ledger posts provisional credit.

### Step 4: Delayed Return

1. RDFI has up to 60 days (consumer accounts) to return the payment.
2. Common return codes: R01 (Insufficient Funds), R02 (Account Closed).
3. Orchestrator handles returns as asynchronous reversals in the ledger.

Risk: the return window means funds are never truly "final" for weeks.

## Dispute and Chargeback Flow

A dispute is an issuer-initiated reversal of a settled payment.

### Notification

PSP receives dispute notice from card network, sends webhook to orchestrator.

### Immediate Reversal

Upon receiving `dispute.created` webhook, the orchestrator must immediately:

1. Debit the merchant's Available balance in the ledger.
2. This happens before the merchant responds to the dispute.
3. Reflects reality: the PSP has already seized the funds.

If a dispute fee applies ($15-$25 typical), a separate entry debits Fee_Expense.

### Resolution

- Merchant wins: subsequent webhook triggers a credit entry back to merchant.
- Merchant loses: the debit stands, funds are permanently reversed.
