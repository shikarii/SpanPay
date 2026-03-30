package ledger

import (
	"context"
	"database/sql"
	"errors"
	"testing"
)

// mockTx implements DBTX for unit tests.
// It captures exec calls and can simulate errors.
// QueryRowContext returns nil since *sql.Row cannot be mocked;
// full DB-path tests belong in integration tests with a real Postgres.
type mockTx struct {
	execCalls []mockExecCall
	execErr   error
}

type mockExecCall struct {
	query string
	args  []any
}

func (m *mockTx) ExecContext(_ context.Context, query string, args ...any) (sql.Result, error) {
	m.execCalls = append(m.execCalls, mockExecCall{query: query, args: args})
	return mockResult{}, m.execErr
}

func (m *mockTx) QueryRowContext(_ context.Context, _ string, _ ...any) *sql.Row {
	return nil
}

type mockResult struct{}

func (mockResult) LastInsertId() (int64, error) { return 0, nil }
func (mockResult) RowsAffected() (int64, error) { return 1, nil }

func TestPostTransaction_RejectsUnbalanced(t *testing.T) {
	mtx := &mockTx{}
	_, err := PostTransaction(context.Background(), mtx, "", "test", []Entry{
		{Account: AcctSettlementPending, Debit: 1000, Credit: 0},
		{Account: AcctMerchantWalletPending, Debit: 0, Credit: 999},
	})
	if err == nil {
		t.Fatal("expected error for unbalanced entries")
	}
	if len(mtx.execCalls) != 0 {
		t.Fatal("expected no DB writes for unbalanced entries")
	}
}

func TestPostTransaction_RejectsSingleEntry(t *testing.T) {
	mtx := &mockTx{}
	_, err := PostTransaction(context.Background(), mtx, "", "test", []Entry{
		{Account: AcctSettlementPending, Debit: 1000, Credit: 0},
	})
	if err == nil {
		t.Fatal("expected error for single entry")
	}
}

func TestPostTransaction_RejectsEmptyAccount(t *testing.T) {
	mtx := &mockTx{}
	_, err := PostTransaction(context.Background(), mtx, "", "test", []Entry{
		{Account: "", Debit: 1000, Credit: 0},
		{Account: AcctMerchantWalletPending, Debit: 0, Credit: 1000},
	})
	if err == nil {
		t.Fatal("expected error for empty account")
	}
}

func TestPostTransaction_RejectsBothSides(t *testing.T) {
	mtx := &mockTx{}
	_, err := PostTransaction(context.Background(), mtx, "", "test", []Entry{
		{Account: AcctSettlementPending, Debit: 500, Credit: 500},
		{Account: AcctMerchantWalletPending, Debit: 0, Credit: 0},
	})
	if err == nil {
		t.Fatal("expected error for entry with both debit and credit")
	}
}

func TestPostTransaction_PropagatesExecError(t *testing.T) {
	mtx := &mockTx{execErr: errors.New("db connection lost")}
	_, err := PostTransaction(context.Background(), mtx, "", "test", []Entry{
		{Account: AcctSettlementPending, Debit: 1000, Credit: 0},
		{Account: AcctMerchantWalletPending, Debit: 0, Credit: 1000},
	})
	if err == nil {
		t.Fatal("expected error from DB failure")
	}
}

func TestPostTransaction_CallsInsertBeforeVerify(t *testing.T) {
	mtx := &mockTx{}
	// PostTransaction will succeed through inserts but panic on nil Row from
	// QueryRowContext. We use recover to verify inserts happened.
	func() {
		defer func() { recover() }()
		PostTransaction(context.Background(), mtx, "pay_123", "authorize", []Entry{
			{Account: AcctSettlementPending, Debit: 1000, Credit: 0},
			{Account: AcctMerchantWalletPending, Debit: 0, Credit: 1000},
		})
	}()

	// Should have: 1 ledger_transactions insert + 2 ledger_entries inserts.
	if len(mtx.execCalls) != 3 {
		t.Fatalf("expected 3 exec calls, got %d", len(mtx.execCalls))
	}
}
