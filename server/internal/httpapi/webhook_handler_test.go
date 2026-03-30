package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/shikarii/spanpay/server/internal/provider"
	"github.com/shikarii/spanpay/server/internal/state"
)

// mockProvider satisfies provider.Provider for testing.
type mockProvider struct {
	verifySigErr   error
	normalizeErr   error
	normalizeEvent provider.InternalEvent
}

func (m *mockProvider) VerifyWebhookSignature(_ []byte, _ string) error {
	return m.verifySigErr
}

func (m *mockProvider) NormalizeEvent(_ []byte) (provider.InternalEvent, error) {
	return m.normalizeEvent, m.normalizeErr
}

func (m *mockProvider) Authorize(_ context.Context, _ provider.NormalizedRequest) (provider.NormalizedResponse, error) {
	return provider.NormalizedResponse{}, nil
}

func (m *mockProvider) Capture(_ context.Context, _ string, _ int64) (provider.NormalizedResponse, error) {
	return provider.NormalizedResponse{}, nil
}

func (m *mockProvider) Refund(_ context.Context, _ string, _ int64) (provider.NormalizedResponse, error) {
	return provider.NormalizedResponse{}, nil
}

func (m *mockProvider) QueryStatus(_ context.Context, _ string) (provider.NormalizedResponse, error) {
	return provider.NormalizedResponse{}, nil
}

// mockExecer satisfies the Execer interface for testing.
type mockExecer struct {
	err       error
	callCount int
}

func (m *mockExecer) ExecContext(_ context.Context, _ string, _ ...any) (sql.Result, error) {
	m.callCount++
	if m.err != nil {
		return nil, m.err
	}
	return mockDBResult{}, nil
}

type mockDBResult struct{}

func (mockDBResult) LastInsertId() (int64, error) { return 0, nil }
func (mockDBResult) RowsAffected() (int64, error) { return 1, nil }

func newTestRouter(mock *mockProvider, db Execer) http.Handler {
	reg := provider.NewRegistry()
	reg.Register("stripe", mock)
	deps := &WebhookDeps{Registry: reg, DB: db}
	return NewRouter(deps)
}

func TestWebhookValidSignatureAndPersist(t *testing.T) {
	db := &mockExecer{}
	mock := &mockProvider{
		normalizeEvent: provider.InternalEvent{
			PaymentID: "pay_123",
			Event:     state.EventCaptureSuccess,
			Amount:    10000,
		},
	}
	router := newTestRouter(mock, db)

	req := httptest.NewRequest(http.MethodPost, "/webhooks/stripe", strings.NewReader(`{"type":"payment_intent.succeeded"}`))
	req.Header.Set("Stripe-Signature", "valid-sig")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if db.callCount != 1 {
		t.Fatalf("expected 1 DB call, got %d", db.callCount)
	}
	var body map[string]string
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body["status"] != "accepted" {
		t.Fatalf("expected status=accepted, got %s", body["status"])
	}
}

func TestWebhookInvalidSignature(t *testing.T) {
	db := &mockExecer{}
	mock := &mockProvider{
		verifySigErr: errors.New("bad signature"),
	}
	router := newTestRouter(mock, db)

	req := httptest.NewRequest(http.MethodPost, "/webhooks/stripe", strings.NewReader(`{}`))
	req.Header.Set("Stripe-Signature", "bad-sig")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	if db.callCount != 0 {
		t.Fatal("DB should not be called on invalid signature")
	}
}

func TestWebhookUnknownProvider(t *testing.T) {
	reg := provider.NewRegistry()
	deps := &WebhookDeps{Registry: reg, DB: &mockExecer{}}
	router := NewRouter(deps)

	req := httptest.NewRequest(http.MethodPost, "/webhooks/unknown", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestWebhookMethodNotAllowed(t *testing.T) {
	router := newTestRouter(&mockProvider{}, &mockExecer{})

	req := httptest.NewRequest(http.MethodGet, "/webhooks/stripe", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestWebhookNormalizeFailStillAccepts(t *testing.T) {
	db := &mockExecer{}
	mock := &mockProvider{
		normalizeErr: errors.New("unrecognized event type"),
	}
	router := newTestRouter(mock, db)

	req := httptest.NewRequest(http.MethodPost, "/webhooks/stripe", strings.NewReader(`{"type":"unknown.event"}`))
	req.Header.Set("Stripe-Signature", "valid-sig")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 (accept despite normalize failure), got %d", rec.Code)
	}
	if db.callCount != 1 {
		t.Fatalf("should persist even when normalize fails; got %d calls", db.callCount)
	}
}

func TestWebhookDBFailure(t *testing.T) {
	db := &mockExecer{err: errors.New("connection refused")}
	mock := &mockProvider{
		normalizeEvent: provider.InternalEvent{PaymentID: "pay_1"},
	}
	router := newTestRouter(mock, db)

	req := httptest.NewRequest(http.MethodPost, "/webhooks/stripe", strings.NewReader(`{}`))
	req.Header.Set("Stripe-Signature", "valid")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 on DB failure, got %d", rec.Code)
	}
}
