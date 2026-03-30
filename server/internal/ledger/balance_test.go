package ledger

import "testing"

func TestBalanceAmount_CreditNormal(t *testing.T) {
	// Liability account: balance = credits - debits
	got := balanceAmount(NormalCredit, 200, 1000)
	if got != 800 {
		t.Fatalf("expected 800, got %d", got)
	}
}

func TestBalanceAmount_DebitNormal(t *testing.T) {
	// Asset account: balance = debits - credits
	got := balanceAmount(NormalDebit, 1000, 200)
	if got != 800 {
		t.Fatalf("expected 800, got %d", got)
	}
}

func TestBalanceAmount_Zero(t *testing.T) {
	got := balanceAmount(NormalDebit, 500, 500)
	if got != 0 {
		t.Fatalf("expected 0, got %d", got)
	}
}

func TestBalanceAmount_Negative(t *testing.T) {
	// A negative balance on a debit-normal account means credits exceed debits.
	got := balanceAmount(NormalDebit, 100, 300)
	if got != -200 {
		t.Fatalf("expected -200, got %d", got)
	}
}
