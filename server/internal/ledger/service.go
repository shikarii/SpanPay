package ledger

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// DBTX abstracts *sql.DB and *sql.Tx so callers can provide either.
type DBTX interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// Transaction is the result of a successful PostTransaction call.
type Transaction struct {
	ID        string
	Reference string
	Entries   []Entry
}

// PostTransaction validates and atomically persists a set of ledger entries.
// It enforces conservation of value: the entries must balance before any write.
// The caller must provide a *sql.Tx to ensure atomicity with surrounding work.
func PostTransaction(ctx context.Context, tx DBTX, paymentID, reference string, entries []Entry) (*Transaction, error) {
	if err := ValidateBalanced(entries); err != nil {
		return nil, fmt.Errorf("ledger validation: %w", err)
	}

	txnID := uuid.New().String()

	var paymentArg any
	if paymentID != "" {
		paymentArg = paymentID
	}

	_, err := tx.ExecContext(ctx,
		`INSERT INTO ledger_transactions (id, payment_id, reference) VALUES ($1, $2, $3)`,
		txnID, paymentArg, reference,
	)
	if err != nil {
		return nil, fmt.Errorf("insert ledger_transactions: %w", err)
	}

	for _, e := range entries {
		entryID := uuid.New().String()
		_, err := tx.ExecContext(ctx,
			`INSERT INTO ledger_entries (id, transaction_id, account_id, debit, credit) VALUES ($1, $2, $3, $4, $5)`,
			entryID, txnID, e.Account, e.Debit, e.Credit,
		)
		if err != nil {
			return nil, fmt.Errorf("insert ledger_entries: %w", err)
		}
	}

	// Verify conservation of value at the database level as a safety net.
	if err := verifyTransactionBalance(ctx, tx, txnID); err != nil {
		return nil, err
	}

	return &Transaction{
		ID:        txnID,
		Reference: reference,
		Entries:   entries,
	}, nil
}

func verifyTransactionBalance(ctx context.Context, tx DBTX, txnID string) error {
	var sumDebit, sumCredit int64
	row := tx.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(debit),0), COALESCE(SUM(credit),0) FROM ledger_entries WHERE transaction_id = $1`,
		txnID,
	)
	if err := row.Scan(&sumDebit, &sumCredit); err != nil {
		return fmt.Errorf("verify balance: %w", err)
	}
	if sumDebit != sumCredit {
		return errors.New("database-level balance check failed: debits != credits")
	}
	return nil
}
