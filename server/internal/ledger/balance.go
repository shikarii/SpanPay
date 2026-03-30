package ledger

import (
	"context"
	"fmt"
	"time"
)

// Balance represents a derived account balance at a point in time.
type Balance struct {
	AccountID     string
	NormalBalance NormalBalance
	Amount        int64
}

// DeriveBalance computes the current balance for a single account.
// Credit-normal: SUM(credits) - SUM(debits).
// Debit-normal:  SUM(debits) - SUM(credits).
func DeriveBalance(ctx context.Context, tx DBTX, accountID string) (*Balance, error) {
	var normalBal string
	var sumDebit, sumCredit int64

	row := tx.QueryRowContext(ctx, `
		SELECT a.normal_balance,
		       COALESCE(SUM(e.debit), 0),
		       COALESCE(SUM(e.credit), 0)
		FROM ledger_accounts a
		LEFT JOIN ledger_entries e ON e.account_id = a.id
		WHERE a.id = $1
		GROUP BY a.normal_balance`,
		accountID,
	)
	if err := row.Scan(&normalBal, &sumDebit, &sumCredit); err != nil {
		return nil, fmt.Errorf("derive balance for %s: %w", accountID, err)
	}

	amount := balanceAmount(NormalBalance(normalBal), sumDebit, sumCredit)
	return &Balance{
		AccountID:     accountID,
		NormalBalance: NormalBalance(normalBal),
		Amount:        amount,
	}, nil
}

// DeriveBalances computes current balances for multiple accounts in one query.
func DeriveBalances(ctx context.Context, db QueryableDB, accountIDs []string) ([]Balance, error) {
	if len(accountIDs) == 0 {
		return nil, nil
	}

	rows, err := db.QueryContext(ctx, `
		SELECT a.id, a.normal_balance,
		       COALESCE(SUM(e.debit), 0),
		       COALESCE(SUM(e.credit), 0)
		FROM ledger_accounts a
		LEFT JOIN ledger_entries e ON e.account_id = a.id
		WHERE a.id = ANY($1)
		GROUP BY a.id, a.normal_balance`,
		accountIDs,
	)
	if err != nil {
		return nil, fmt.Errorf("derive balances: %w", err)
	}
	defer rows.Close()

	var balances []Balance
	for rows.Next() {
		var id, normalBal string
		var sumDebit, sumCredit int64
		if err := rows.Scan(&id, &normalBal, &sumDebit, &sumCredit); err != nil {
			return nil, fmt.Errorf("scan balance row: %w", err)
		}
		balances = append(balances, Balance{
			AccountID:     id,
			NormalBalance: NormalBalance(normalBal),
			Amount:        balanceAmount(NormalBalance(normalBal), sumDebit, sumCredit),
		})
	}
	return balances, rows.Err()
}

// DeriveBalanceAsOf computes the balance for an account at a specific point in time.
func DeriveBalanceAsOf(ctx context.Context, tx DBTX, accountID string, asOf time.Time) (*Balance, error) {
	var normalBal string
	var sumDebit, sumCredit int64

	row := tx.QueryRowContext(ctx, `
		SELECT a.normal_balance,
		       COALESCE(SUM(e.debit), 0),
		       COALESCE(SUM(e.credit), 0)
		FROM ledger_accounts a
		LEFT JOIN ledger_entries e ON e.account_id = a.id AND e.created_at <= $2
		WHERE a.id = $1
		GROUP BY a.normal_balance`,
		accountID, asOf,
	)
	if err := row.Scan(&normalBal, &sumDebit, &sumCredit); err != nil {
		return nil, fmt.Errorf("derive balance as-of for %s: %w", accountID, err)
	}

	amount := balanceAmount(NormalBalance(normalBal), sumDebit, sumCredit)
	return &Balance{
		AccountID:     accountID,
		NormalBalance: NormalBalance(normalBal),
		Amount:        amount,
	}, nil
}

func balanceAmount(normal NormalBalance, sumDebit, sumCredit int64) int64 {
	if normal == NormalCredit {
		return sumCredit - sumDebit
	}
	return sumDebit - sumCredit
}
