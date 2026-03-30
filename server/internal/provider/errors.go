package provider

import "fmt"

// ErrorCode classifies provider-side failures into a finite taxonomy.
type ErrorCode string

const (
	ErrInsufficientFunds ErrorCode = "INSUFFICIENT_FUNDS"
	ErrCardExpired       ErrorCode = "CARD_EXPIRED"
	ErrFraudBlocked      ErrorCode = "FRAUD_BLOCKED"
	ErrTechnicalError    ErrorCode = "TECHNICAL_ERROR"
	ErrRateLimited       ErrorCode = "RATE_LIMITED"
	ErrAccountClosed     ErrorCode = "ACCOUNT_CLOSED"
)

// retryable maps each error code to whether the operation can be retried.
var retryable = map[ErrorCode]bool{
	ErrInsufficientFunds: false,
	ErrCardExpired:       false,
	ErrFraudBlocked:      false,
	ErrTechnicalError:    true,
	ErrAccountClosed:     false,
	ErrRateLimited:       true,
}

// IsRetryable returns true if the given error code indicates a transient
// failure that may succeed on retry.
func IsRetryable(code ErrorCode) bool {
	return retryable[code]
}

// ProviderError wraps an ErrorCode with optional detail from the provider.
type ProviderError struct {
	Code    ErrorCode
	Message string
}

func (e *ProviderError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}
