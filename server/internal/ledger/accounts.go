package ledger

// System-level ledger account IDs.
// These match the seed data in db/migrations/000002_chart_of_accounts.sql.
const (
	AcctSettlementPending      = "a0000000-0000-0000-0000-000000000001"
	AcctSettlementAssets       = "a0000000-0000-0000-0000-000000000002"
	AcctMerchantWalletPending  = "a0000000-0000-0000-0000-000000000003"
	AcctMerchantWalletAvail    = "a0000000-0000-0000-0000-000000000004"
	AcctMerchantBankAccount    = "a0000000-0000-0000-0000-000000000005"
	AcctFeeExpense             = "a0000000-0000-0000-0000-000000000006"
	AcctInboundClearing        = "a0000000-0000-0000-0000-000000000007"
	AcctCurrencyExchangeLoss   = "a0000000-0000-0000-0000-000000000008"
	AcctRoundingExpense        = "a0000000-0000-0000-0000-000000000009"
)

// AccountType enumerates the valid ledger account classifications.
type AccountType string

const (
	AccountTypeAsset     AccountType = "ASSET"
	AccountTypeLiability AccountType = "LIABILITY"
	AccountTypeEquity    AccountType = "EQUITY"
	AccountTypeExpense   AccountType = "EXPENSE"
	AccountTypeRevenue   AccountType = "REVENUE"
)

// NormalBalance indicates which side increases the account.
type NormalBalance string

const (
	NormalDebit  NormalBalance = "DEBIT"
	NormalCredit NormalBalance = "CREDIT"
)
