package httpapi

import (
	"context"
	"database/sql"
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/shikarii/spanpay/server/internal/provider"
)

// Execer abstracts the ExecContext method for testability.
type Execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// WebhookDeps holds the dependencies for the webhook handler.
type WebhookDeps struct {
	Registry *provider.Registry
	DB       Execer
}

// HandleWebhook returns an http.HandlerFunc for POST /webhooks/{provider}.
// It verifies the signature, persists the raw payload to webhook_inbox,
// and returns 200 OK without doing any downstream processing.
func HandleWebhook(deps WebhookDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		providerName := r.PathValue("provider")
		if providerName == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing provider"})
			return
		}

		prov, err := deps.Registry.Get(providerName)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		// Read raw body before any parsing — signature verification needs exact bytes.
		rawBody, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1 MB limit
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "failed to read body"})
			return
		}

		// Verify signature using provider adapter.
		// Signature header name varies by provider; adapters read from the raw header string.
		sigHeader := r.Header.Get("Stripe-Signature")
		if sig := r.Header.Get("X-Webhook-Signature"); sig != "" {
			sigHeader = sig
		}

		if err := prov.VerifyWebhookSignature(rawBody, sigHeader); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid signature"})
			return
		}

		// Extract provider event ID for deduplication.
		evt, err := prov.NormalizeEvent(rawBody)
		if err != nil {
			// If we can't normalize but signature is valid, still persist for manual review.
			_ = persistInbox(r.Context(), deps.DB, providerName, "", rawBody)
			writeJSON(w, http.StatusOK, map[string]string{"status": "accepted"})
			return
		}

		// Persist to inbox. Duplicate provider_evt_id is silently ignored (ON CONFLICT DO NOTHING).
		if err := persistInbox(r.Context(), deps.DB, providerName, evt.PaymentID, rawBody); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "persist failed"})
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{"status": "accepted"})
	}
}

func persistInbox(ctx context.Context, db Execer, providerName, providerEvtID string, rawPayload []byte) error {
	id := uuid.New().String()
	var evtID any
	if providerEvtID != "" {
		evtID = providerEvtID
	}
	_, err := db.ExecContext(ctx,
		`INSERT INTO webhook_inbox (id, provider, provider_evt_id, raw_payload, status)
		 VALUES ($1, $2, $3, $4, 'PENDING')
		 ON CONFLICT (provider_evt_id) DO NOTHING`,
		id, providerName, evtID, rawPayload,
	)
	return err
}
