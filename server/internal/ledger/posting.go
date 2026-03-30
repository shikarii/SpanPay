package ledger

import "errors"

const (
	InvariantConservationOfValue = "conservation_of_value"
	InvariantAppendOnly          = "append_only_entries"
	InvariantDerivedBalances     = "derived_balances"
)

type Entry struct {
	Account string
	Debit   int64
	Credit  int64
}

func ValidateBalanced(entries []Entry) error {
	if len(entries) < 2 {
		return errors.New("ledger transaction requires at least two entries")
	}

	var debits int64
	var credits int64

	for _, entry := range entries {
		if entry.Account == "" {
			return errors.New("ledger entry account is required")
		}
		if entry.Debit < 0 || entry.Credit < 0 {
			return errors.New("ledger entry amounts must be non-negative")
		}
		if (entry.Debit == 0) == (entry.Credit == 0) {
			return errors.New("ledger entry must post to exactly one side")
		}
		debits += entry.Debit
		credits += entry.Credit
	}

	if debits != credits {
		return errors.New("ledger transaction is unbalanced")
	}

	return nil
}
