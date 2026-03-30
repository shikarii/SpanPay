# Ledger Posting Rules

Deterministic rules mapping business events to accounting entries.
These rules are enforced by the ledger engine in `server/internal/ledger/`.

## Fundamental Invariants

1. **Conservation of Value**: Sum of debits must equal sum of credits for every transaction.
2. **Derived Balances**: Balances are computed, never stored as mutable fields.
   - Credit-normal accounts: Balance = SUM(credits) - SUM(debits)
   - Debit-normal accounts: Balance = SUM(debits) - SUM(credits)
3. **Immutability**: Entries are append-only. Corrections create new compensating entries.
4. **Reference Integrity**: Every entry references a valid ledger_transaction and ledger_account.

## Chart of Accounts

| Account | Type | Normal Balance | Purpose |
|---|---|---|---|
| Settlement_Pending | ASSET | DEBIT | Funds reserved at PSP but not yet captured |
| Settlement_Assets | ASSET | DEBIT | Funds held at PSP/Acquirer post-capture |
| Merchant_Wallet_Pending | LIABILITY | CREDIT | Funds owed to merchant, pending capture |
| Merchant_Wallet_Available | LIABILITY | CREDIT | Funds available for merchant payout |
| Merchant_Bank_Account | ASSET | DEBIT | Funds deposited in merchant's bank |
| Fee_Expense | EXPENSE | DEBIT | Costs paid to PSPs (processing fees, dispute fees) |
| Inbound_Clearing | ASSET | DEBIT | Temporary staging for unconfirmed funds |
| Currency_Exchange_Loss | EXPENSE | DEBIT | FX rounding gaps booked during reconciliation |
| Rounding_Expense | EXPENSE | DEBIT | Sub-threshold rounding differences |

## Posting Rules by Event

### Payment Authorized

Funds reserved but not yet received.

| Side | Account | Direction |
|---|---|---|
| Debit | Settlement_Pending | Asset increases |
| Credit | Merchant_Wallet_Pending | Liability increases |

Timing: immediately upon successful PSP authorization response.

### Payment Captured

Funds move from pending to available on internal books.

| Side | Account | Direction |
|---|---|---|
| Debit | Merchant_Wallet_Pending | Liability decreases |
| Credit | Merchant_Wallet_Available | Liability increases |
| Debit | Settlement_Assets | Asset increases |
| Credit | Settlement_Pending | Asset decreases |

Timing: upon capture success webhook.

### Refund Processed

| Side | Account | Direction |
|---|---|---|
| Debit | Merchant_Wallet_Available | Liability decreases |
| Credit | Settlement_Assets | Asset decreases |

Timing: when PSP confirms refund succeeded.

### Dispute/Chargeback Received

Two sub-transactions, both posted atomically:

Transaction amount reversal:

| Side | Account | Direction |
|---|---|---|
| Debit | Merchant_Wallet_Available | Liability decreases |
| Credit | Settlement_Assets | Asset decreases |

Dispute fee (if applicable):

| Side | Account | Direction |
|---|---|---|
| Debit | Fee_Expense | Expense increases |
| Credit | Settlement_Assets | Asset decreases |

Timing: immediately upon receiving `dispute.created` webhook.

### Dispute Won

Reverses the dispute reversal:

| Side | Account | Direction |
|---|---|---|
| Debit | Settlement_Assets | Asset increases |
| Credit | Merchant_Wallet_Available | Liability increases |

Fee is not reversed (dispute fees are typically non-refundable).

### Bank Payout (Settlement)

Physical funds movement from PSP to merchant's bank.

| Side | Account | Direction |
|---|---|---|
| Debit | Merchant_Bank_Account | Asset increases |
| Credit | Settlement_Assets | Asset decreases |

Timing: during reconciliation of bank statement.
