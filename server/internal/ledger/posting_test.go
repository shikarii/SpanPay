package ledger

import "testing"

func TestValidateBalancedAcceptsBalancedEntries(t *testing.T) {
	err := ValidateBalanced([]Entry{
		{Account: "settlement_pending", Debit: 1000},
		{Account: "merchant_wallet_pending", Credit: 1000},
	})

	if err != nil {
		t.Fatalf("expected balanced entries, got %v", err)
	}
}

func TestValidateBalancedRejectsUnbalancedEntries(t *testing.T) {
	err := ValidateBalanced([]Entry{
		{Account: "settlement_pending", Debit: 1000},
		{Account: "merchant_wallet_pending", Credit: 900},
	})

	if err == nil {
		t.Fatalf("expected unbalanced entries to fail")
	}
}

func TestValidateBalancedRejectsInvalidEntryShapes(t *testing.T) {
	cases := [][]Entry{
		{{Account: "", Debit: 100}},
		{{Account: "a", Debit: 100, Credit: 100}, {Account: "b", Credit: 200}},
		{{Account: "a", Debit: -1}, {Account: "b", Credit: 1}},
		{{Account: "a", Debit: 100}},
	}

	for index, entries := range cases {
		if err := ValidateBalanced(entries); err == nil {
			t.Fatalf("case %d should fail", index)
		}
	}
}
