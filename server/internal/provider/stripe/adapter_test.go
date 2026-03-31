package stripe

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	gostripe "github.com/stripe/stripe-go/v82"

	"github.com/shikarii/spanpay/server/internal/provider"
	"github.com/shikarii/spanpay/server/internal/state"
)

// mockAPI implements StripeAPI for testing.
type mockAPI struct {
	createPI  func(ctx context.Context, params *gostripe.PaymentIntentCreateParams) (*gostripe.PaymentIntent, error)
	capturePI func(ctx context.Context, id string, params *gostripe.PaymentIntentCaptureParams) (*gostripe.PaymentIntent, error)
	createRef func(ctx context.Context, params *gostripe.RefundCreateParams) (*gostripe.Refund, error)
	getPI     func(ctx context.Context, id string, params *gostripe.PaymentIntentRetrieveParams) (*gostripe.PaymentIntent, error)
}

func (m *mockAPI) CreatePaymentIntent(ctx context.Context, params *gostripe.PaymentIntentCreateParams) (*gostripe.PaymentIntent, error) {
	return m.createPI(ctx, params)
}

func (m *mockAPI) CapturePaymentIntent(ctx context.Context, id string, params *gostripe.PaymentIntentCaptureParams) (*gostripe.PaymentIntent, error) {
	return m.capturePI(ctx, id, params)
}

func (m *mockAPI) CreateRefund(ctx context.Context, params *gostripe.RefundCreateParams) (*gostripe.Refund, error) {
	return m.createRef(ctx, params)
}

func (m *mockAPI) RetrievePaymentIntent(ctx context.Context, id string, params *gostripe.PaymentIntentRetrieveParams) (*gostripe.PaymentIntent, error) {
	return m.getPI(ctx, id, params)
}

func TestAuthorizeSuccess(t *testing.T) {
	mock := &mockAPI{
		createPI: func(_ context.Context, params *gostripe.PaymentIntentCreateParams) (*gostripe.PaymentIntent, error) {
			if *params.CaptureMethod != "manual" {
				t.Fatal("expected capture_method=manual")
			}
			if *params.Amount != 5000 {
				t.Fatalf("expected amount 5000, got %d", *params.Amount)
			}
			return &gostripe.PaymentIntent{
				ID:     "pi_test_123",
				Status: gostripe.PaymentIntentStatusRequiresCapture,
			}, nil
		},
	}

	adapter := NewWithAPI(mock, "whsec_test")
	resp, err := adapter.Authorize(context.Background(), provider.NormalizedRequest{
		Amount:   5000,
		Currency: "usd",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ProviderRef != "pi_test_123" {
		t.Fatalf("expected pi_test_123, got %s", resp.ProviderRef)
	}
	if resp.Status != provider.StatusSuccess {
		t.Fatalf("expected SUCCESS, got %s", resp.Status)
	}
}

func TestAuthorizeWithPaymentMethod(t *testing.T) {
	mock := &mockAPI{
		createPI: func(_ context.Context, params *gostripe.PaymentIntentCreateParams) (*gostripe.PaymentIntent, error) {
			if params.PaymentMethod == nil || *params.PaymentMethod != "pm_card_visa" {
				t.Fatal("expected payment method pm_card_visa")
			}
			if params.Confirm == nil || !*params.Confirm {
				t.Fatal("expected confirm=true when payment method provided")
			}
			return &gostripe.PaymentIntent{
				ID:     "pi_test_456",
				Status: gostripe.PaymentIntentStatusRequiresCapture,
			}, nil
		},
	}

	adapter := NewWithAPI(mock, "whsec_test")
	_, err := adapter.Authorize(context.Background(), provider.NormalizedRequest{
		Amount:   3000,
		Currency: "usd",
		PaymentMethod: provider.PaymentMethod{
			Type:    "card",
			TokenID: "pm_card_visa",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAuthorizeStripeError(t *testing.T) {
	mock := &mockAPI{
		createPI: func(_ context.Context, _ *gostripe.PaymentIntentCreateParams) (*gostripe.PaymentIntent, error) {
			return nil, &gostripe.Error{
				Code: gostripe.ErrorCodeInsufficientFunds,
				Msg:  "insufficient funds",
			}
		},
	}

	adapter := NewWithAPI(mock, "whsec_test")
	_, err := adapter.Authorize(context.Background(), provider.NormalizedRequest{
		Amount:   5000,
		Currency: "usd",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	var provErr *provider.ProviderError
	if !errors.As(err, &provErr) {
		t.Fatalf("expected ProviderError, got %T", err)
	}
	if provErr.Code != provider.ErrInsufficientFunds {
		t.Fatalf("expected INSUFFICIENT_FUNDS, got %s", provErr.Code)
	}
}

func TestCaptureSuccess(t *testing.T) {
	mock := &mockAPI{
		capturePI: func(_ context.Context, id string, params *gostripe.PaymentIntentCaptureParams) (*gostripe.PaymentIntent, error) {
			if id != "pi_test_123" {
				t.Fatalf("unexpected id: %s", id)
			}
			if params.AmountToCapture == nil || *params.AmountToCapture != 3000 {
				t.Fatal("expected partial capture of 3000")
			}
			return &gostripe.PaymentIntent{
				ID:     "pi_test_123",
				Status: gostripe.PaymentIntentStatusSucceeded,
			}, nil
		},
	}

	adapter := NewWithAPI(mock, "whsec_test")
	resp, err := adapter.Capture(context.Background(), "pi_test_123", 3000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != provider.StatusSuccess {
		t.Fatalf("expected SUCCESS, got %s", resp.Status)
	}
}

func TestRefundSuccess(t *testing.T) {
	mock := &mockAPI{
		createRef: func(_ context.Context, params *gostripe.RefundCreateParams) (*gostripe.Refund, error) {
			if *params.PaymentIntent != "pi_test_123" {
				t.Fatal("unexpected payment intent")
			}
			if *params.Amount != 2000 {
				t.Fatal("expected partial refund of 2000")
			}
			return &gostripe.Refund{
				ID:     "re_test_789",
				Status: gostripe.RefundStatusSucceeded,
			}, nil
		},
	}

	adapter := NewWithAPI(mock, "whsec_test")
	resp, err := adapter.Refund(context.Background(), "pi_test_123", 2000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ProviderRef != "re_test_789" {
		t.Fatalf("expected re_test_789, got %s", resp.ProviderRef)
	}
	if resp.Status != provider.StatusSuccess {
		t.Fatalf("expected SUCCESS, got %s", resp.Status)
	}
}

func TestQueryStatusSuccess(t *testing.T) {
	mock := &mockAPI{
		getPI: func(_ context.Context, id string, _ *gostripe.PaymentIntentRetrieveParams) (*gostripe.PaymentIntent, error) {
			return &gostripe.PaymentIntent{
				ID:     id,
				Status: gostripe.PaymentIntentStatusProcessing,
			}, nil
		},
	}

	adapter := NewWithAPI(mock, "whsec_test")
	resp, err := adapter.QueryStatus(context.Background(), "pi_test_123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != provider.StatusPending {
		t.Fatalf("expected PENDING for processing status, got %s", resp.Status)
	}
}

func TestNormalizeEventPaymentIntentSucceeded(t *testing.T) {
	payload := makeEventPayload("payment_intent.succeeded", `{"id":"pi_abc","amount":10000}`)

	adapter := NewWithAPI(nil, "whsec_test")
	evt, err := adapter.NormalizeEvent(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if evt.Event != state.EventAuthSuccess {
		t.Fatalf("expected AUTH_SUCCESS, got %s", evt.Event)
	}
	if evt.PaymentID != "pi_abc" {
		t.Fatalf("expected pi_abc, got %s", evt.PaymentID)
	}
	if evt.Amount != 10000 {
		t.Fatalf("expected 10000, got %d", evt.Amount)
	}
}

func TestNormalizeEventChargeRefunded(t *testing.T) {
	payload := makeEventPayload("charge.refunded", `{"id":"ch_xyz","amount":5000}`)

	adapter := NewWithAPI(nil, "whsec_test")
	evt, err := adapter.NormalizeEvent(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if evt.Event != state.EventRefundSuccess {
		t.Fatalf("expected REFUND_SUCCESS, got %s", evt.Event)
	}
}

func TestNormalizeEventDisputeCreated(t *testing.T) {
	payload := makeEventPayload("charge.dispute.created", `{"id":"dp_123","amount":7500}`)

	adapter := NewWithAPI(nil, "whsec_test")
	evt, err := adapter.NormalizeEvent(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if evt.Event != state.EventDisputeCreated {
		t.Fatalf("expected DISPUTE_CREATED, got %s", evt.Event)
	}
}

func TestNormalizeEventUnknownType(t *testing.T) {
	payload := makeEventPayload("customer.created", `{"id":"cus_123"}`)

	adapter := NewWithAPI(nil, "whsec_test")
	_, err := adapter.NormalizeEvent(payload)
	if err == nil {
		t.Fatal("expected error for unknown event type")
	}
}

func TestNormalizeEventInvalidJSON(t *testing.T) {
	adapter := NewWithAPI(nil, "whsec_test")
	_, err := adapter.NormalizeEvent([]byte(`not json`))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestMapStripeErrorCodes(t *testing.T) {
	cases := []struct {
		stripeCode gostripe.ErrorCode
		wantCode   provider.ErrorCode
	}{
		{gostripe.ErrorCodeInsufficientFunds, provider.ErrInsufficientFunds},
		{gostripe.ErrorCodeExpiredCard, provider.ErrCardExpired},
		{gostripe.ErrorCodeCardDeclined, provider.ErrFraudBlocked},
		{"processing_error", provider.ErrTechnicalError},
		{"rate_limit", provider.ErrRateLimited},
	}

	for _, tc := range cases {
		t.Run(string(tc.stripeCode), func(t *testing.T) {
			err := mapStripeError(&gostripe.Error{Code: tc.stripeCode, Msg: "test"})
			var provErr *provider.ProviderError
			if !errors.As(err, &provErr) {
				t.Fatalf("expected ProviderError, got %T", err)
			}
			if provErr.Code != tc.wantCode {
				t.Fatalf("expected %s, got %s", tc.wantCode, provErr.Code)
			}
		})
	}
}

func TestMapUnknownStripeError(t *testing.T) {
	err := mapStripeError(&gostripe.Error{Code: "some_unknown_code", Msg: "unknown"})
	var provErr *provider.ProviderError
	if !errors.As(err, &provErr) {
		t.Fatal("expected ProviderError")
	}
	if provErr.Code != provider.ErrTechnicalError {
		t.Fatalf("expected TECHNICAL_ERROR for unknown code, got %s", provErr.Code)
	}
}

func TestMapNonStripeError(t *testing.T) {
	err := mapStripeError(errors.New("network timeout"))
	var provErr *provider.ProviderError
	if !errors.As(err, &provErr) {
		t.Fatal("expected ProviderError")
	}
	if provErr.Code != provider.ErrTechnicalError {
		t.Fatalf("expected TECHNICAL_ERROR, got %s", provErr.Code)
	}
}

func TestIntentStatusMapping(t *testing.T) {
	cases := []struct {
		status gostripe.PaymentIntentStatus
		want   provider.AttemptStatus
	}{
		{gostripe.PaymentIntentStatusSucceeded, provider.StatusSuccess},
		{gostripe.PaymentIntentStatusRequiresCapture, provider.StatusSuccess},
		{gostripe.PaymentIntentStatusCanceled, provider.StatusFailed},
		{gostripe.PaymentIntentStatusProcessing, provider.StatusPending},
		{gostripe.PaymentIntentStatusRequiresPaymentMethod, provider.StatusPending},
	}

	for _, tc := range cases {
		t.Run(string(tc.status), func(t *testing.T) {
			got := mapIntentStatus(tc.status)
			if got != tc.want {
				t.Fatalf("expected %s, got %s", tc.want, got)
			}
		})
	}
}

func TestRefundStatusMapping(t *testing.T) {
	cases := []struct {
		status gostripe.RefundStatus
		want   provider.AttemptStatus
	}{
		{gostripe.RefundStatusSucceeded, provider.StatusSuccess},
		{gostripe.RefundStatusFailed, provider.StatusFailed},
		{gostripe.RefundStatusCanceled, provider.StatusFailed},
		{gostripe.RefundStatusPending, provider.StatusPending},
	}

	for _, tc := range cases {
		t.Run(string(tc.status), func(t *testing.T) {
			got := mapRefundStatus(tc.status)
			if got != tc.want {
				t.Fatalf("expected %s, got %s", tc.want, got)
			}
		})
	}
}

func TestAdapterImplementsProvider(t *testing.T) {
	var _ provider.Provider = (*Adapter)(nil)
}

func TestStripeEventMap(t *testing.T) {
	expected := map[string]state.Event{
		"payment_intent.succeeded":      state.EventAuthSuccess,
		"payment_intent.payment_failed": state.EventAuthFailure,
		"charge.captured":               state.EventCaptureSuccess,
		"charge.refunded":               state.EventRefundSuccess,
		"charge.dispute.created":        state.EventDisputeCreated,
		"charge.dispute.closed":         state.EventDisputeWon,
	}
	for stripeType, want := range expected {
		got, ok := stripeEventMap[stripeType]
		if !ok {
			t.Fatalf("missing mapping for %s", stripeType)
		}
		if got != want {
			t.Fatalf("for %s: expected %s, got %s", stripeType, want, got)
		}
	}
}

func makeEventPayload(eventType, objectJSON string) []byte {
	evt := map[string]any{
		"type": eventType,
		"data": map[string]any{
			"object": json.RawMessage(objectJSON),
		},
	}
	b, _ := json.Marshal(evt)
	return b
}
