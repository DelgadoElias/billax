package provider

import "fmt"

// ErrorCategory classifies provider errors in a provider-agnostic way
// All providers (MercadoPago, Stripe, Helipagos) map their HTTP status codes
// to these categories, enabling shared error handling in the adapter and webhook handler
type ErrorCategory string

const (
	// ErrorCategoryAuthFailure indicates authentication/authorization failure (HTTP 401 from provider)
	// Examples: invalid access token, expired credentials, revoked key
	ErrorCategoryAuthFailure ErrorCategory = "auth_failure"

	// ErrorCategoryRejected indicates the provider rejected the operation (HTTP 422 from provider)
	// Examples: insufficient funds, limit exceeded, card declined, invalid customer
	ErrorCategoryRejected ErrorCategory = "rejected"

	// ErrorCategoryRateLimited indicates rate limiting (HTTP 429 from provider, retries exhausted)
	// Examples: too many requests, quota exceeded
	ErrorCategoryRateLimited ErrorCategory = "rate_limited"

	// ErrorCategoryValidation indicates the request payload was invalid (HTTP 400 from provider)
	// Examples: malformed JSON, missing required field, invalid format
	ErrorCategoryValidation ErrorCategory = "validation_error"

	// ErrorCategoryUnknown is used for unclassifiable errors or unexpected status codes
	ErrorCategoryUnknown ErrorCategory = "unknown"
)

// ProviderError represents a classified error response from any payment provider
// The error is categorized in a provider-agnostic way, enabling the adapter and webhook
// handler to handle errors identically regardless of which provider is used
type ProviderError struct {
	// Category is the provider-agnostic classification (auth_failure, rate_limited, etc.)
	Category ErrorCategory

	// HTTPStatus is the HTTP status code returned by the provider (401, 422, 429, 400, etc.)
	HTTPStatus int

	// ProviderCode is the error code from the provider (e.g. "unauthorized", "insufficient_funds")
	// Used for debugging and logging
	ProviderCode string

	// ProviderMessage is the human-readable error message from the provider
	// Can be exposed to clients for context
	ProviderMessage string
}

// Error implements the error interface
func (e *ProviderError) Error() string {
	return fmt.Sprintf("provider error [%s] %d: %s (%s)",
		e.Category, e.HTTPStatus, e.ProviderMessage, e.ProviderCode)
}

// IsAuthFailure returns true if this is an authentication/authorization error
func (e *ProviderError) IsAuthFailure() bool {
	return e.Category == ErrorCategoryAuthFailure
}

// IsRejected returns true if this is a rejected operation
func (e *ProviderError) IsRejected() bool {
	return e.Category == ErrorCategoryRejected
}

// IsRateLimited returns true if this is a rate limit error
func (e *ProviderError) IsRateLimited() bool {
	return e.Category == ErrorCategoryRateLimited
}

// IsValidationError returns true if this is a validation error
func (e *ProviderError) IsValidationError() bool {
	return e.Category == ErrorCategoryValidation
}
