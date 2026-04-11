package errors

import (
	"errors"
	"net/http"
)

// Sentinel errors — use errors.Is() to match
var (
	ErrNotFound              = errors.New("not found")
	ErrConflict              = errors.New("conflict")
	ErrInvalidInput          = errors.New("invalid input")
	ErrDuplicatePlan         = errors.New("plan with this slug already exists")
	ErrPlanNotActive         = errors.New("plan is not active")
	ErrSubscriptionExpired   = errors.New("subscription has expired")
	ErrDuplicateIdempotency  = errors.New("duplicate idempotency key")
	ErrMissingIdempotencyKey = errors.New("Idempotency-Key header is required")
	ErrMissingTenantID       = errors.New("tenant ID missing from context")
	ErrProviderNotFound      = errors.New("provider not registered")
	ErrProviderError         = errors.New("provider error")
	ErrPayPerUseNotSupported = errors.New("provider does not support pay-per-use billing")
	ErrPlansNotSupported     = errors.New("provider does not support plan-based billing")
	ErrProviderRequired      = errors.New("a provider must be set to update amount")
	ErrNotSupported          = errors.New("operation not supported by this provider")
	ErrDuplicateTenantSlug   = errors.New("a tenant with this slug already exists")
	ErrDuplicateEmail        = errors.New("this email is already registered")
	ErrKeyNotFound           = errors.New("api key not found")
)

// DomainError carries HTTP-level context for the handler layer
type DomainError struct {
	Code       string
	Message    string
	HTTPStatus int
	Cause      error
}

func (e *DomainError) Error() string { return e.Message }
func (e *DomainError) Unwrap() error { return e.Cause }

// HTTPStatusFor maps a sentinel error to an HTTP status code
func HTTPStatusFor(err error) int {
	// Check DomainError first for custom HTTP status
	if de, ok := err.(*DomainError); ok {
		return de.HTTPStatus
	}

	switch {
	case errors.Is(err, ErrNotFound), errors.Is(err, ErrKeyNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrConflict), errors.Is(err, ErrDuplicatePlan), errors.Is(err, ErrDuplicateIdempotency), errors.Is(err, ErrDuplicateTenantSlug), errors.Is(err, ErrDuplicateEmail):
		return http.StatusConflict
	case errors.Is(err, ErrInvalidInput), errors.Is(err, ErrMissingIdempotencyKey):
		return http.StatusBadRequest
	case errors.Is(err, ErrPlanNotActive), errors.Is(err, ErrSubscriptionExpired):
		return http.StatusUnprocessableEntity
	case errors.Is(err, ErrPayPerUseNotSupported), errors.Is(err, ErrPlansNotSupported), errors.Is(err, ErrProviderRequired):
		return http.StatusUnprocessableEntity
	case errors.Is(err, ErrProviderNotFound):
		return http.StatusBadRequest
	case errors.Is(err, ErrProviderError):
		return http.StatusBadGateway
	case errors.Is(err, ErrNotSupported):
		return http.StatusNotImplemented
	default:
		return http.StatusInternalServerError
	}
}

// CodeFor maps a sentinel error to a machine-readable string code
func CodeFor(err error) string {
	// Check DomainError first for custom code
	if de, ok := err.(*DomainError); ok {
		return de.Code
	}

	switch {
	case errors.Is(err, ErrNotFound), errors.Is(err, ErrKeyNotFound):
		return "not_found"
	case errors.Is(err, ErrDuplicatePlan), errors.Is(err, ErrDuplicateTenantSlug), errors.Is(err, ErrDuplicateEmail):
		return "duplicate"
	case errors.Is(err, ErrPlanNotActive):
		return "plan_not_active"
	case errors.Is(err, ErrSubscriptionExpired):
		return "subscription_expired"
	case errors.Is(err, ErrDuplicateIdempotency):
		return "duplicate_idempotency_key"
	case errors.Is(err, ErrMissingIdempotencyKey):
		return "missing_idempotency_key"
	case errors.Is(err, ErrInvalidInput):
		return "invalid_input"
	case errors.Is(err, ErrConflict):
		return "conflict"
	case errors.Is(err, ErrPayPerUseNotSupported):
		return "pay_per_use_not_supported"
	case errors.Is(err, ErrPlansNotSupported):
		return "plans_not_supported"
	case errors.Is(err, ErrProviderRequired):
		return "provider_required"
	case errors.Is(err, ErrProviderNotFound):
		return "provider_not_found"
	case errors.Is(err, ErrProviderError):
		return "provider_error"
	case errors.Is(err, ErrNotSupported):
		return "not_supported"
	default:
		return "internal_error"
	}
}
