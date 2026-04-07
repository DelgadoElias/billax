package validation

import (
	"fmt"
	"regexp"
	"strings"

	apperrors "github.com/DelgadoElias/billax/internal/errors"
)

// FieldError represents a single field-level validation failure
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationError accumulates multiple FieldErrors and satisfies the error interface
type ValidationError struct {
	Fields []FieldError `json:"fields"`
}

// Error returns a human-readable concatenation of all field errors
func (ve *ValidationError) Error() string {
	if len(ve.Fields) == 0 {
		return "validation error"
	}
	msgs := make([]string, len(ve.Fields))
	for i, f := range ve.Fields {
		msgs[i] = fmt.Sprintf("%s: %s", f.Field, f.Message)
	}
	return strings.Join(msgs, "; ")
}

// Unwrap returns ErrInvalidInput so errors.Is(ve, ErrInvalidInput) == true
// This makes HTTPStatusFor and CodeFor return 400 / "invalid_input" automatically
func (ve *ValidationError) Unwrap() error {
	return apperrors.ErrInvalidInput
}

// HasErrors reports whether any field errors were collected
func (ve *ValidationError) HasErrors() bool {
	return len(ve.Fields) > 0
}

// Add appends a FieldError if it is non-nil; supports chainable pattern
func (ve *ValidationError) Add(fe *FieldError) {
	if fe != nil {
		ve.Fields = append(ve.Fields, *fe)
	}
}

// Err returns the ValidationError as an error if it has fields, otherwise nil
// This allows callers to write: if err := v.Err(); err != nil { ... }
func (ve *ValidationError) Err() error {
	if ve.HasErrors() {
		return ve
	}
	return nil
}

// --- Individual validators: each returns *FieldError if invalid, nil if valid ---

// NonEmpty validates that a string is not empty after trimming whitespace
func NonEmpty(field, value string) *FieldError {
	if strings.TrimSpace(value) == "" {
		return &FieldError{Field: field, Message: "must not be empty"}
	}
	return nil
}

// MaxLength validates that a string does not exceed a maximum length
func MaxLength(field, value string, max int) *FieldError {
	if len(value) > max {
		return &FieldError{Field: field, Message: fmt.Sprintf("must be at most %d characters", max)}
	}
	return nil
}

// MinInt validates that an integer is at least a minimum value
func MinInt(field string, value, min int64) *FieldError {
	if value < min {
		return &FieldError{Field: field, Message: fmt.Sprintf("must be at least %d", min)}
	}
	return nil
}

// MaxInt validates that an integer does not exceed a maximum value
func MaxInt(field string, value, max int64) *FieldError {
	if value > max {
		return &FieldError{Field: field, Message: fmt.Sprintf("must be at most %d", max)}
	}
	return nil
}

// PositiveInt validates that an integer is strictly greater than zero
func PositiveInt(field string, value int64) *FieldError {
	if value <= 0 {
		return &FieldError{Field: field, Message: "must be a positive integer"}
	}
	return nil
}

// ISOCurrency validates that a currency code is in the supported Latin America-focused allowlist
// Supported currencies: ARS, USD, BRL, CLP, COP, MXN, UYU, PEN, EUR, GBP
func ISOCurrency(field, value string) *FieldError {
	allowed := map[string]bool{
		"ARS": true, "USD": true, "BRL": true, "CLP": true,
		"COP": true, "MXN": true, "UYU": true, "PEN": true,
		"EUR": true, "GBP": true,
	}
	upper := strings.ToUpper(value)
	if !allowed[upper] {
		return &FieldError{Field: field, Message: fmt.Sprintf("must be a supported ISO 4217 currency code (got %q)", value)}
	}
	return nil
}

// ValidEnum validates that a string is one of the allowed values
func ValidEnum(field, value string, allowed []string) *FieldError {
	for _, a := range allowed {
		if value == a {
			return nil
		}
	}
	return &FieldError{Field: field, Message: fmt.Sprintf("must be one of: %s", strings.Join(allowed, ", "))}
}

// ValidUUID validates that a string is a valid UUID (case-insensitive)
var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func ValidUUID(field, value string) *FieldError {
	if !uuidPattern.MatchString(strings.ToLower(value)) {
		return &FieldError{Field: field, Message: "must be a valid UUID"}
	}
	return nil
}

// ErrInvalidInput wraps a ValidationError for use with errors.Is
func ErrValidation(ve *ValidationError) error {
	if ve.HasErrors() {
		return ve
	}
	return nil
}
