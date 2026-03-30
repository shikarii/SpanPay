package ledger

import "testing"

// Posting pattern fixtures: these mirror the rules in docs/spec/ledger-rules.md.
// Each fixture validates that the entries for a business event are balanced
// and structurally correct.

func TestPostingPattern_Authorize(t *testing.T) {
	entries := []Entry{
		{Account: AcctSettlementPending, Debit: 10000, Credit: 0},
		{Account: AcctMerchantWalletPending, Debit: 0, Credit: 10000},
	}
	if err := ValidateBalanced(entries); err != nil {
		t.Fatalf("authorize pattern should be balanced: %v", err)
	}
}

func TestPostingPattern_Capture(t *testing.T) {
	entries := []Entry{
		{Account: AcctMerchantWalletPending, Debit: 10000, Credit: 0},
		{Account: AcctMerchantWalletAvail, Debit: 0, Credit: 10000},
		{Account: AcctSettlementAssets, Debit: 10000, Credit: 0},
		{Account: AcctSettlementPending, Debit: 0, Credit: 10000},
	}
	if err := ValidateBalanced(entries); err != nil {
		t.Fatalf("capture pattern should be balanced: %v", err)
	}
}

func TestPostingPattern_Refund(t *testing.T) {
	entries := []Entry{
		{Account: AcctMerchantWalletAvail, Debit: 10000, Credit: 0},
		{Account: AcctSettlementAssets, Debit: 0, Credit: 10000},
	}
	if err := ValidateBalanced(entries); err != nil {
		t.Fatalf("refund pattern should be balanced: %v", err)
	}
}

func TestPostingPattern_DisputeWithFee(t *testing.T) {
	// Transaction amount reversal + dispute fee, all in one transaction.
	entries := []Entry{
		{Account: AcctMerchantWalletAvail, Debit: 10000, Credit: 0},
		{Account: AcctSettlementAssets, Debit: 0, Credit: 10000},
		{Account: AcctFeeExpense, Debit: 1500, Credit: 0},
		{Account: AcctSettlementAssets, Debit: 0, Credit: 1500},
	}
	if err := ValidateBalanced(entries); err != nil {
		t.Fatalf("dispute+fee pattern should be balanced: %v", err)
	}
}

func TestPostingPattern_DisputeWon(t *testing.T) {
	// Reverses the dispute amount but not the fee.
	entries := []Entry{
		{Account: AcctSettlementAssets, Debit: 10000, Credit: 0},
		{Account: AcctMerchantWalletAvail, Debit: 0, Credit: 10000},
	}
	if err := ValidateBalanced(entries); err != nil {
		t.Fatalf("dispute-won pattern should be balanced: %v", err)
	}
}

func TestPostingPattern_BankPayout(t *testing.T) {
	entries := []Entry{
		{Account: AcctMerchantBankAccount, Debit: 10000, Credit: 0},
		{Account: AcctSettlementAssets, Debit: 0, Credit: 10000},
	}
	if err := ValidateBalanced(entries); err != nil {
		t.Fatalf("bank payout pattern should be balanced: %v", err)
	}
}

// Invariant enforcement tests.

func TestInvariant_ConservationOfValue_RejectsOffByOne(t *testing.T) {
	entries := []Entry{
		{Account: AcctSettlementPending, Debit: 10000, Credit: 0},
		{Account: AcctMerchantWalletPending, Debit: 0, Credit: 9999},
	}
	if err := ValidateBalanced(entries); err == nil {
		t.Fatal("should reject entries off by one cent")
	}
}

func TestInvariant_SingleSide_RejectsBothPositive(t *testing.T) {
	entries := []Entry{
		{Account: AcctSettlementPending, Debit: 500, Credit: 500},
		{Account: AcctMerchantWalletPending, Debit: 500, Credit: 500},
	}
	if err := ValidateBalanced(entries); err == nil {
		t.Fatal("should reject entry with both debit and credit positive")
	}
}

func TestInvariant_SingleSide_RejectsBothZero(t *testing.T) {
	entries := []Entry{
		{Account: AcctSettlementPending, Debit: 0, Credit: 0},
		{Account: AcctMerchantWalletPending, Debit: 0, Credit: 0},
	}
	if err := ValidateBalanced(entries); err == nil {
		t.Fatal("should reject entry with both debit and credit zero")
	}
}

func TestInvariant_NegativeAmounts_Rejected(t *testing.T) {
	entries := []Entry{
		{Account: AcctSettlementPending, Debit: -1000, Credit: 0},
		{Account: AcctMerchantWalletPending, Debit: 0, Credit: -1000},
	}
	if err := ValidateBalanced(entries); err == nil {
		t.Fatal("should reject negative amounts")
	}
}

func TestInvariant_EmptyAccountName_Rejected(t *testing.T) {
	entries := []Entry{
		{Account: "", Debit: 1000, Credit: 0},
		{Account: AcctMerchantWalletPending, Debit: 0, Credit: 1000},
	}
	if err := ValidateBalanced(entries); err == nil {
		t.Fatal("should reject empty account name")
	}
}

func TestInvariant_MinimumTwoEntries(t *testing.T) {
	entries := []Entry{
		{Account: AcctSettlementPending, Debit: 1000, Credit: 0},
	}
	if err := ValidateBalanced(entries); err == nil {
		t.Fatal("should reject single entry")
	}
}

func TestInvariant_EmptyEntries_Rejected(t *testing.T) {
	if err := ValidateBalanced(nil); err == nil {
		t.Fatal("should reject nil entries")
	}
	if err := ValidateBalanced([]Entry{}); err == nil {
		t.Fatal("should reject empty entries")
	}
}

// Full lifecycle: authorize -> capture -> refund verifies that all posting
// patterns in sequence maintain balance consistency.
func TestFullLifecycle_AuthCaptureRefund(t *testing.T) {
	patterns := []struct {
		name    string
		entries []Entry
	}{
		{"authorize", []Entry{
			{Account: AcctSettlementPending, Debit: 5000, Credit: 0},
			{Account: AcctMerchantWalletPending, Debit: 0, Credit: 5000},
		}},
		{"capture", []Entry{
			{Account: AcctMerchantWalletPending, Debit: 5000, Credit: 0},
			{Account: AcctMerchantWalletAvail, Debit: 0, Credit: 5000},
			{Account: AcctSettlementAssets, Debit: 5000, Credit: 0},
			{Account: AcctSettlementPending, Debit: 0, Credit: 5000},
		}},
		{"refund", []Entry{
			{Account: AcctMerchantWalletAvail, Debit: 5000, Credit: 0},
			{Account: AcctSettlementAssets, Debit: 0, Credit: 5000},
		}},
	}

	for _, p := range patterns {
		t.Run(p.name, func(t *testing.T) {
			if err := ValidateBalanced(p.entries); err != nil {
				t.Fatalf("%s entries should be balanced: %v", p.name, err)
			}
		})
	}
}
