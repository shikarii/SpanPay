# Reconciliation Specification

Reconciliation compares internal records with external reality to ensure no money is missing.
It is the definitive check on system correctness.

## Three-Way Match

The reconciliation engine matches across three data sources:

1. **Internal Ledger**: what SpanPay thinks happened (entries in `ledger_entries`).
2. **Provider Settlement Reports**: what the PSP says they processed (e.g. Stripe Payout CSV).
3. **Bank Statements**: what actually landed in the corporate bank account.

## Algorithm

### Step 1: Data Collection

Fetch settlement file from the PSP via API or SFTP.
Store raw file for audit trail.

### Step 2: Normalization

Map PSP CSV headers to internal format:
- `txn_id` -> `provider_ref`
- `net_amt` -> `amount`
- `fee` -> `provider_fee`

### Step 3: Matching

Match each row in the settlement report to a `payment_attempt` using `provider_ref`.
Build three sets: matched, orphan-internal, orphan-external.

### Step 4: Exception Identification

| Exception | Description | Action |
|---|---|---|
| Orphan Ledger Entry | Payment SUCCESS internally but not in PSP report | Investigate, may need correction journal |
| Orphan PSP Entry | Payment in PSP report but not in internal database | Auto-heal: create internal state and ledger entries |
| Amount Mismatch | Net amount differs from expected (after fees) | Flag for review, book difference if within tolerance |

## Tolerance Rules

- Differences below threshold (e.g. $0.05) due to rounding or minor FX fluctuations
  are automatically booked to `Rounding_Expense` account.
- Multi-currency transactions may show FX rate differences;
  book to `Currency_Exchange_Loss` account.

## Drift Detection

If total balance in `Settlement_Assets` differs from the PSP-reported balance,
raise an "Unreconciled Drift" alert.

## Repair Strategy

- Finance team posts a "Correction Journal" to the ledger.
- Original entries are never modified.
- The correction journal references the reconciliation_run that identified the issue.
- Complete audit trail of how the books were aligned.

## Auto-Heal: Provider Success, Local DB Failure

If the PSP captured a payment but SpanPay crashed before updating the DB:

1. Reconciliation detects the orphan PSP entry.
2. Creates the missing internal state (payment in CAPTURED).
3. Posts the missing ledger entries.
4. Logs the auto-heal action for audit.

## Schedule

- Daily reconciliation against PSP settlement reports.
- Weekly reconciliation against bank statements.
- On-demand reconciliation for specific payment ranges.
