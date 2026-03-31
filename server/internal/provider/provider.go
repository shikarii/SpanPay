package provider

import (
	"context"
	"encoding/json"

	"github.com/shikarii/spanpay/server/internal/state"
)

// AttemptStatus classifies the outcome of a provider operation.
type AttemptStatus string

const (
	StatusSuccess AttemptStatus = "SUCCESS"
	StatusFailed  AttemptStatus = "FAILED"
	StatusPending AttemptStatus = "PENDING"
)

// PaymentMethod holds tokenized payment details passed to the provider.
type PaymentMethod struct {
	Type    string // "card", "bank_transfer", etc.
	TokenID string // provider-specific token or payment method ID
}

// NormalizedRequest is the provider-agnostic input for an authorization.
type NormalizedRequest struct {
	Amount        int64
	Currency      string
	MerchantID    string
	ExternalRef   string
	PaymentMethod PaymentMethod
	Metadata      map[string]string
}

// NormalizedResponse is the provider-agnostic result of any provider operation.
type NormalizedResponse struct {
	ProviderRef string
	Status      AttemptStatus
	ErrorCode   string
	RawResponse json.RawMessage
}

// InternalEvent is a normalized webhook event ready for the state machine.
type InternalEvent struct {
	PaymentID   string
	ProviderRef string
	Event       state.Event
	Amount      int64
	FeeAmount   int64
	Metadata    map[string]string
}

// Provider defines the contract every PSP adapter must implement.
type Provider interface {
	// Authorize initiates a charge or authorization hold.
	Authorize(ctx context.Context, req NormalizedRequest) (NormalizedResponse, error)

	// Capture collects funds from a previously authorized charge.
	Capture(ctx context.Context, providerRef string, amount int64) (NormalizedResponse, error)

	// Refund reverses a settled transaction (full or partial).
	Refund(ctx context.Context, providerRef string, amount int64) (NormalizedResponse, error)

	// QueryStatus queries the current status of a transaction at the provider.
	QueryStatus(ctx context.Context, providerRef string) (NormalizedResponse, error)

	// NormalizeEvent translates a raw webhook payload into an InternalEvent.
	NormalizeEvent(rawPayload []byte) (InternalEvent, error)

	// VerifyWebhookSignature validates the HMAC signature of an incoming webhook.
	VerifyWebhookSignature(rawBody []byte, signatureHeader string) error
}
