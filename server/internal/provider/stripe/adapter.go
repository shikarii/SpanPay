package stripe

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	gostripe "github.com/stripe/stripe-go/v82"

	"github.com/shikarii/spanpay/server/internal/provider"
	"github.com/shikarii/spanpay/server/internal/state"
)

// StripeAPI abstracts Stripe SDK calls for testability.
type StripeAPI interface {
	CreatePaymentIntent(ctx context.Context, params *gostripe.PaymentIntentCreateParams) (*gostripe.PaymentIntent, error)
	CapturePaymentIntent(ctx context.Context, id string, params *gostripe.PaymentIntentCaptureParams) (*gostripe.PaymentIntent, error)
	CreateRefund(ctx context.Context, params *gostripe.RefundCreateParams) (*gostripe.Refund, error)
	RetrievePaymentIntent(ctx context.Context, id string, params *gostripe.PaymentIntentRetrieveParams) (*gostripe.PaymentIntent, error)
}

// sdkClient wraps the real Stripe client.
type sdkClient struct {
	client *gostripe.Client
}

func (s *sdkClient) CreatePaymentIntent(ctx context.Context, params *gostripe.PaymentIntentCreateParams) (*gostripe.PaymentIntent, error) {
	return s.client.V1PaymentIntents.Create(ctx, params)
}

func (s *sdkClient) CapturePaymentIntent(ctx context.Context, id string, params *gostripe.PaymentIntentCaptureParams) (*gostripe.PaymentIntent, error) {
	return s.client.V1PaymentIntents.Capture(ctx, id, params)
}

func (s *sdkClient) CreateRefund(ctx context.Context, params *gostripe.RefundCreateParams) (*gostripe.Refund, error) {
	return s.client.V1Refunds.Create(ctx, params)
}

func (s *sdkClient) RetrievePaymentIntent(ctx context.Context, id string, params *gostripe.PaymentIntentRetrieveParams) (*gostripe.PaymentIntent, error) {
	return s.client.V1PaymentIntents.Retrieve(ctx, id, params)
}

// Adapter implements provider.Provider for Stripe.
type Adapter struct {
	api           StripeAPI
	webhookSecret string
}

// New creates a production Stripe adapter.
func New(apiKey, webhookSecret string) *Adapter {
	return &Adapter{
		api:           &sdkClient{client: gostripe.NewClient(apiKey)},
		webhookSecret: webhookSecret,
	}
}

// NewWithAPI creates an Adapter with a custom StripeAPI (for testing).
func NewWithAPI(api StripeAPI, webhookSecret string) *Adapter {
	return &Adapter{api: api, webhookSecret: webhookSecret}
}

// Authorize creates a PaymentIntent with capture_method=manual.
func (a *Adapter) Authorize(ctx context.Context, req provider.NormalizedRequest) (provider.NormalizedResponse, error) {
	params := &gostripe.PaymentIntentCreateParams{
		Amount:        gostripe.Int64(req.Amount),
		Currency:      gostripe.String(req.Currency),
		CaptureMethod: gostripe.String("manual"),
		Metadata:      req.Metadata,
	}
	if req.PaymentMethod.TokenID != "" {
		params.PaymentMethod = gostripe.String(req.PaymentMethod.TokenID)
		params.Confirm = gostripe.Bool(true)
	}

	pi, err := a.api.CreatePaymentIntent(ctx, params)
	if err != nil {
		return provider.NormalizedResponse{}, mapStripeError(err)
	}

	return provider.NormalizedResponse{
		ProviderRef: pi.ID,
		Status:      mapIntentStatus(pi.Status),
		RawResponse: mustMarshal(pi),
	}, nil
}

// Capture collects funds from a previously authorized PaymentIntent.
func (a *Adapter) Capture(ctx context.Context, providerRef string, amount int64) (provider.NormalizedResponse, error) {
	params := &gostripe.PaymentIntentCaptureParams{}
	if amount > 0 {
		params.AmountToCapture = gostripe.Int64(amount)
	}

	pi, err := a.api.CapturePaymentIntent(ctx, providerRef, params)
	if err != nil {
		return provider.NormalizedResponse{}, mapStripeError(err)
	}

	return provider.NormalizedResponse{
		ProviderRef: pi.ID,
		Status:      mapIntentStatus(pi.Status),
		RawResponse: mustMarshal(pi),
	}, nil
}

// Refund creates a refund against a PaymentIntent.
func (a *Adapter) Refund(ctx context.Context, providerRef string, amount int64) (provider.NormalizedResponse, error) {
	params := &gostripe.RefundCreateParams{
		PaymentIntent: gostripe.String(providerRef),
	}
	if amount > 0 {
		params.Amount = gostripe.Int64(amount)
	}

	ref, err := a.api.CreateRefund(ctx, params)
	if err != nil {
		return provider.NormalizedResponse{}, mapStripeError(err)
	}

	return provider.NormalizedResponse{
		ProviderRef: ref.ID,
		Status:      mapRefundStatus(ref.Status),
		RawResponse: mustMarshal(ref),
	}, nil
}

// QueryStatus retrieves the current state of a PaymentIntent.
func (a *Adapter) QueryStatus(ctx context.Context, providerRef string) (provider.NormalizedResponse, error) {
	pi, err := a.api.RetrievePaymentIntent(ctx, providerRef, &gostripe.PaymentIntentRetrieveParams{})
	if err != nil {
		return provider.NormalizedResponse{}, mapStripeError(err)
	}

	return provider.NormalizedResponse{
		ProviderRef: pi.ID,
		Status:      mapIntentStatus(pi.Status),
		RawResponse: mustMarshal(pi),
	}, nil
}

// NormalizeEvent translates a raw Stripe webhook payload into an InternalEvent.
func (a *Adapter) NormalizeEvent(rawPayload []byte) (provider.InternalEvent, error) {
	var raw struct {
		Type string `json:"type"`
		Data struct {
			Object json.RawMessage `json:"object"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rawPayload, &raw); err != nil {
		return provider.InternalEvent{}, fmt.Errorf("unmarshal event: %w", err)
	}

	evt, ok := stripeEventMap[raw.Type]
	if !ok {
		return provider.InternalEvent{}, fmt.Errorf("unhandled stripe event type: %s", raw.Type)
	}

	var obj struct {
		ID       string            `json:"id"`
		Amount   int64             `json:"amount"`
		Metadata map[string]string `json:"metadata"`
	}
	if err := json.Unmarshal(raw.Data.Object, &obj); err != nil {
		return provider.InternalEvent{}, fmt.Errorf("unmarshal event object: %w", err)
	}

	return provider.InternalEvent{
		PaymentID:   obj.ID,
		ProviderRef: obj.ID,
		Event:       evt,
		Amount:      obj.Amount,
		Metadata:    obj.Metadata,
	}, nil
}

// VerifyWebhookSignature validates a Stripe webhook signature.
func (a *Adapter) VerifyWebhookSignature(rawBody []byte, signatureHeader string) error {
	_, err := gostripe.ConstructEvent(rawBody, signatureHeader, a.webhookSecret)
	if err != nil {
		return fmt.Errorf("invalid stripe signature: %w", err)
	}
	return nil
}

// stripeEventMap maps Stripe webhook event types to internal state machine events.
var stripeEventMap = map[string]state.Event{
	"payment_intent.succeeded":      state.EventAuthSuccess,
	"payment_intent.payment_failed": state.EventAuthFailure,
	"charge.captured":               state.EventCaptureSuccess,
	"charge.refunded":               state.EventRefundSuccess,
	"charge.dispute.created":        state.EventDisputeCreated,
	"charge.dispute.closed":         state.EventDisputeWon,
}

// stripeErrorMap maps Stripe error codes to internal error codes.
var stripeErrorMap = map[gostripe.ErrorCode]provider.ErrorCode{
	gostripe.ErrorCodeInsufficientFunds: provider.ErrInsufficientFunds,
	gostripe.ErrorCodeExpiredCard:       provider.ErrCardExpired,
	gostripe.ErrorCodeCardDeclined:      provider.ErrFraudBlocked,
	"processing_error":                  provider.ErrTechnicalError,
	"rate_limit":                        provider.ErrRateLimited,
}

func mapStripeError(err error) error {
	var stripeErr *gostripe.Error
	if errors.As(err, &stripeErr) {
		code, ok := stripeErrorMap[stripeErr.Code]
		if !ok {
			code = provider.ErrTechnicalError
		}
		return &provider.ProviderError{
			Code:    code,
			Message: stripeErr.Msg,
		}
	}
	return &provider.ProviderError{
		Code:    provider.ErrTechnicalError,
		Message: err.Error(),
	}
}

func mapIntentStatus(s gostripe.PaymentIntentStatus) provider.AttemptStatus {
	switch s {
	case gostripe.PaymentIntentStatusSucceeded, gostripe.PaymentIntentStatusRequiresCapture:
		return provider.StatusSuccess
	case gostripe.PaymentIntentStatusCanceled:
		return provider.StatusFailed
	default:
		return provider.StatusPending
	}
}

func mapRefundStatus(s gostripe.RefundStatus) provider.AttemptStatus {
	switch s {
	case gostripe.RefundStatusSucceeded:
		return provider.StatusSuccess
	case gostripe.RefundStatusFailed, gostripe.RefundStatusCanceled:
		return provider.StatusFailed
	default:
		return provider.StatusPending
	}
}

func mustMarshal(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}
