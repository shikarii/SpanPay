package provider

import (
	"testing"

	"github.com/shikarii/spanpay/server/internal/state"
)

func TestErrorCodeConstants(t *testing.T) {
	codes := []ErrorCode{
		ErrInsufficientFunds, ErrCardExpired, ErrFraudBlocked,
		ErrTechnicalError, ErrRateLimited, ErrAccountClosed,
	}
	seen := make(map[ErrorCode]bool)
	for _, c := range codes {
		if c == "" {
			t.Fatal("empty error code")
		}
		if seen[c] {
			t.Fatalf("duplicate error code: %s", c)
		}
		seen[c] = true
	}
}

func TestRetryClassification(t *testing.T) {
	cases := []struct {
		code      ErrorCode
		retryable bool
	}{
		{ErrInsufficientFunds, false},
		{ErrCardExpired, false},
		{ErrFraudBlocked, false},
		{ErrTechnicalError, true},
		{ErrRateLimited, true},
		{ErrAccountClosed, false},
	}
	for _, tc := range cases {
		t.Run(string(tc.code), func(t *testing.T) {
			if got := IsRetryable(tc.code); got != tc.retryable {
				t.Fatalf("IsRetryable(%s) = %v, want %v", tc.code, got, tc.retryable)
			}
		})
	}
}

func TestUnknownErrorCodeNotRetryable(t *testing.T) {
	if IsRetryable("UNKNOWN_CODE") {
		t.Fatal("unknown error codes should not be retryable")
	}
}

func TestProviderErrorMessage(t *testing.T) {
	err := &ProviderError{Code: ErrCardExpired, Message: "card ending 4242 expired"}
	got := err.Error()
	want := "CARD_EXPIRED: card ending 4242 expired"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestAttemptStatusConstants(t *testing.T) {
	if StatusSuccess != "SUCCESS" {
		t.Fatal("unexpected SUCCESS constant")
	}
	if StatusFailed != "FAILED" {
		t.Fatal("unexpected FAILED constant")
	}
	if StatusPending != "PENDING" {
		t.Fatal("unexpected PENDING constant")
	}
}

func TestNormalizedRequestFields(t *testing.T) {
	req := NormalizedRequest{
		Amount:      5000,
		Currency:    "usd",
		MerchantID:  "merch_1",
		ExternalRef: "order_99",
		PaymentMethod: PaymentMethod{
			Type:    "card",
			TokenID: "pm_test_123",
		},
		Metadata: map[string]string{"key": "val"},
	}
	if req.PaymentMethod.Type != "card" {
		t.Fatal("unexpected payment method type")
	}
	if req.PaymentMethod.TokenID != "pm_test_123" {
		t.Fatal("unexpected token ID")
	}
}

func TestInternalEventFields(t *testing.T) {
	evt := InternalEvent{
		PaymentID:   "pay_1",
		ProviderRef: "pi_abc",
		Event:       state.EventAuthSuccess,
		Amount:      10000,
		FeeAmount:   0,
		Metadata:    map[string]string{"stripe_id": "evt_123"},
	}
	if evt.ProviderRef != "pi_abc" {
		t.Fatal("unexpected provider ref")
	}
	if evt.Event != state.EventAuthSuccess {
		t.Fatal("unexpected event type")
	}
}

func TestRegistryRegisterAndGet(t *testing.T) {
	reg := NewRegistry()
	_, err := reg.Get("nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
}
