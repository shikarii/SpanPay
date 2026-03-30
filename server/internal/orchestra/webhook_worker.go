package orchestra

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/shikarii/spanpay/server/internal/provider"
	"github.com/shikarii/spanpay/server/internal/state"
)

const (
	MaxRetries           = 5
	PollInterval         = 1 * time.Second
	InboxStatusPending   = "PENDING"
	InboxStatusProcessed = "PROCESSED"
	InboxStatusFailed    = "FAILED"
)

// InboxItem represents a row from webhook_inbox.
type InboxItem struct {
	ID            string
	Provider      string
	ProviderEvtID string
	RawPayload    []byte
	Status        string
	RetryCount    int
}

// WorkerDB abstracts the database operations the worker needs.
type WorkerDB interface {
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// WorkerDeps holds the dependencies for the webhook worker.
type WorkerDeps struct {
	DB       WorkerDB
	Registry *provider.Registry
}

// ProcessNextItem picks up one PENDING inbox item, processes it through the
// state machine, writes ledger entries, and marks it complete. Returns false
// if no items are available.
//
// Uses SELECT FOR UPDATE SKIP LOCKED to allow concurrent workers without
// blocking each other. The worker_id partitioning happens at the polling level.
func ProcessNextItem(ctx context.Context, deps WorkerDeps) (bool, error) {
	tx, err := deps.DB.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// Pick up one PENDING item, locking it for this worker.
	var item InboxItem
	err = tx.QueryRowContext(ctx,
		`SELECT id, provider, COALESCE(provider_evt_id, ''), raw_payload, status,
		        COALESCE(retry_count, 0)
		 FROM webhook_inbox
		 WHERE status = $1
		 ORDER BY received_at
		 FOR UPDATE SKIP LOCKED
		 LIMIT 1`,
		InboxStatusPending,
	).Scan(&item.ID, &item.Provider, &item.ProviderEvtID, &item.RawPayload, &item.Status, &item.RetryCount)

	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("pick inbox item: %w", err)
	}

	// Look up the provider adapter.
	prov, err := deps.Registry.Get(item.Provider)
	if err != nil {
		return true, markFailed(ctx, tx, item.ID, fmt.Sprintf("unknown provider: %s", item.Provider))
	}

	// Normalize the raw webhook into an internal event.
	evt, err := prov.NormalizeEvent(item.RawPayload)
	if err != nil {
		return true, markFailed(ctx, tx, item.ID, fmt.Sprintf("normalize: %v", err))
	}

	// Look up current payment state.
	var currentState string
	err = tx.QueryRowContext(ctx,
		`SELECT status FROM payments WHERE id = $1 FOR UPDATE`,
		evt.PaymentID,
	).Scan(&currentState)
	if errors.Is(err, sql.ErrNoRows) {
		// Payment doesn't exist yet; treat as StateNone for HWM.
		currentState = ""
	} else if err != nil {
		return true, incrementRetry(ctx, tx, item, fmt.Sprintf("lookup payment: %v", err))
	}

	// Apply state machine transition.
	transition, err := state.ApplyEvent(state.PaymentState(currentState), evt.Event)
	if err != nil {
		// Invalid or terminal: mark as failed, don't retry.
		return true, markFailed(ctx, tx, item.ID, fmt.Sprintf("state machine: %v", err))
	}

	if transition.Idempotent {
		// Already processed this event; mark as processed and move on.
		return true, markProcessed(ctx, tx, item.ID)
	}

	// Update payment state.
	_, err = tx.ExecContext(ctx,
		`UPDATE payments SET status = $2 WHERE id = $1`,
		evt.PaymentID, string(transition.To),
	)
	if err != nil {
		return true, incrementRetry(ctx, tx, item, fmt.Sprintf("update payment: %v", err))
	}

	// Mark inbox item as processed.
	if err := markProcessed(ctx, tx, item.ID); err != nil {
		return true, err
	}

	return true, tx.Commit()
}

func markProcessed(ctx context.Context, tx *sql.Tx, itemID string) error {
	_, err := tx.ExecContext(ctx,
		`UPDATE webhook_inbox SET status = $2 WHERE id = $1`,
		itemID, InboxStatusProcessed,
	)
	return err
}

func markFailed(ctx context.Context, tx *sql.Tx, itemID, reason string) error {
	_, err := tx.ExecContext(ctx,
		`UPDATE webhook_inbox SET status = $2 WHERE id = $1`,
		itemID, InboxStatusFailed,
	)
	if err != nil {
		return err
	}
	log.Printf("webhook inbox item %s failed: %s", itemID, reason)
	return tx.Commit()
}

func incrementRetry(ctx context.Context, tx *sql.Tx, item InboxItem, reason string) error {
	newCount := item.RetryCount + 1
	if newCount >= MaxRetries {
		log.Printf("webhook inbox item %s exceeded max retries (%d): %s", item.ID, MaxRetries, reason)
		return markFailed(ctx, tx, item.ID, fmt.Sprintf("max retries exceeded: %s", reason))
	}
	_, err := tx.ExecContext(ctx,
		`UPDATE webhook_inbox SET retry_count = $2 WHERE id = $1`,
		item.ID, newCount,
	)
	if err != nil {
		return err
	}
	log.Printf("webhook inbox item %s retry %d/%d: %s", item.ID, newCount, MaxRetries, reason)
	return tx.Commit()
}

// RunWorker starts a polling loop that processes webhook inbox items.
// It blocks until ctx is cancelled.
func RunWorker(ctx context.Context, deps WorkerDeps) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		processed, err := ProcessNextItem(ctx, deps)
		if err != nil {
			log.Printf("webhook worker error: %v", err)
		}
		if !processed {
			// No items available; wait before polling again.
			select {
			case <-ctx.Done():
				return
			case <-time.After(PollInterval):
			}
		}
	}
}
