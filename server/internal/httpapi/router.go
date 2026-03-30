package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/shikarii/spanpay/server/internal/ledger"
)

type healthResponse struct {
	OK             bool     `json:"ok"`
	Service        string   `json:"service"`
	CoreInvariants []string `json:"core_invariants"`
}

func NewRouter(deps *WebhookDeps) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", handleHealth)
	mux.HandleFunc("/v1/payments", handlePayments)
	if deps != nil {
		mux.HandleFunc("/webhooks/{provider}", HandleWebhook(*deps))
	}
	return mux
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, healthResponse{
		OK:      true,
		Service: "spanpay-api",
		CoreInvariants: []string{
			ledger.InvariantConservationOfValue,
			ledger.InvariantAppendOnly,
			ledger.InvariantDerivedBalances,
		},
	})
}

func handlePayments(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusNotImplemented, map[string]any{
		"ok":      false,
		"message": "payment orchestration endpoints are scaffolded but not implemented yet",
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
