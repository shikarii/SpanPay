package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthRouteReturnsCoreInvariants(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	recorder := httptest.NewRecorder()

	NewRouter().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	var payload struct {
		OK             bool     `json:"ok"`
		Service        string   `json:"service"`
		CoreInvariants []string `json:"core_invariants"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if !payload.OK {
		t.Fatalf("expected ok=true")
	}
	if payload.Service != "spanpay-api" {
		t.Fatalf("unexpected service name: %s", payload.Service)
	}
	if len(payload.CoreInvariants) != 3 {
		t.Fatalf("expected three invariants, got %d", len(payload.CoreInvariants))
	}
}

func TestPaymentsRouteIsExplicitlyUnimplemented(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/v1/payments", nil)
	recorder := httptest.NewRecorder()

	NewRouter().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotImplemented {
		t.Fatalf("expected 501, got %d", recorder.Code)
	}
}
