package orchestra

import (
	"testing"
)

func TestMaxRetriesConstant(t *testing.T) {
	if MaxRetries != 5 {
		t.Fatalf("expected MaxRetries=5, got %d", MaxRetries)
	}
}

func TestInboxStatusConstants(t *testing.T) {
	if InboxStatusPending != "PENDING" {
		t.Fatal("unexpected PENDING constant")
	}
	if InboxStatusProcessed != "PROCESSED" {
		t.Fatal("unexpected PROCESSED constant")
	}
	if InboxStatusFailed != "FAILED" {
		t.Fatal("unexpected FAILED constant")
	}
}

func TestInboxItemFields(t *testing.T) {
	item := InboxItem{
		ID:            "test-id",
		Provider:      "stripe",
		ProviderEvtID: "evt_123",
		RawPayload:    []byte(`{"type":"payment_intent.succeeded"}`),
		Status:        InboxStatusPending,
		RetryCount:    0,
	}
	if item.ID != "test-id" {
		t.Fatal("unexpected ID")
	}
	if item.RetryCount != 0 {
		t.Fatal("unexpected retry count")
	}
}

// ProcessNextItem and RunWorker require a real *sql.Tx (BeginTx returns *sql.Tx,
// which is concrete and can't be easily mocked). These are validated in integration
// tests with a real Postgres instance.
//
// The core processing logic delegates to:
// - provider.NormalizeEvent (tested in provider package)
// - state.ApplyEvent (tested in state package)
// - markProcessed/markFailed/incrementRetry (SQL operations)
//
// Unit coverage focuses on ensuring constants and types are correct.
// Full flow coverage is in integration tests.
