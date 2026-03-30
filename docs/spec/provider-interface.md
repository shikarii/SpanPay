# Provider Abstraction Layer

A strict interface that normalizes vendor-specific logic into a unified internal format.
Allows replacing one PSP with another without changing the core ledger or business logic.

## Unified Provider Interface

```go
// IPaymentProvider defines the contract every PSP adapter must implement.
type IPaymentProvider interface {
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
```

## Normalized Response

```go
type NormalizedResponse struct {
    ProviderRef string
    Status      AttemptStatus // SUCCESS, FAILED, PENDING
    ErrorCode   string        // Normalized internal error code, empty on success
    RawResponse json.RawMessage
}
```

## Error Taxonomy

Providers use different codes for the same problem.
The adapter maps these to internal enums.

| Internal Code | Stripe | PayPal | ACH |
|---|---|---|---|
| INSUFFICIENT_FUNDS | insufficient_funds | INSTRUMENT_DECLINED | R01 |
| CARD_EXPIRED | expired_card | EXPIRED_CREDIT_CARD | N/A |
| ACCOUNT_CLOSED | N/A | N/A | R02 |
| FRAUD_BLOCKED | fraudulent | RISK_REJECTION | R29 |
| TECHNICAL_ERROR | processing_error | INTERNAL_SERVER_ERROR | N/A |
| RATE_LIMITED | rate_limit | RATE_LIMIT_REACHED | N/A |

## Retry Decisions Based on Error Code

| Error Code | Retryable | Action |
|---|---|---|
| INSUFFICIENT_FUNDS | No | Fail permanently |
| CARD_EXPIRED | No | Fail permanently |
| FRAUD_BLOCKED | No | Fail permanently |
| TECHNICAL_ERROR | Yes | Retry on same or different provider |
| RATE_LIMITED | Yes | Retry with backoff |
| ACCOUNT_CLOSED | No | Fail permanently |

## MVP Provider: Stripe

The first adapter implements `IPaymentProvider` for Stripe.
Testing uses the Stripe CLI test mode and webhook forwarding.

Stripe-specific mapping details:
- Authorization: `POST /v1/payment_intents` with `capture_method=manual`
- Capture: `POST /v1/payment_intents/:id/capture`
- Refund: `POST /v1/refunds`
- Webhooks: `payment_intent.succeeded`, `charge.refunded`, `charge.dispute.created`
