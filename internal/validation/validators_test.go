package validation

import (
	"errors"
	"strings"
	"testing"

	apperrors "github.com/DelgadoElias/billax/internal/errors"
)

func TestValidationError_Unwrap(t *testing.T) {
	ve := &ValidationError{
		Fields: []FieldError{
			{Field: "name", Message: "required"},
		},
	}

	// Verify that errors.Is(ve, ErrInvalidInput) returns true
	if !errors.Is(ve, apperrors.ErrInvalidInput) {
		t.Error("ValidationError.Unwrap() should make errors.Is(ve, ErrInvalidInput) return true")
	}
}

func TestValidationError_Add(t *testing.T) {
	ve := &ValidationError{}
	ve.Add(NonEmpty("name", ""))
	ve.Add(PositiveInt("amount", 0))
	ve.Add(nil) // should not panic

	if len(ve.Fields) != 2 {
		t.Errorf("expected 2 errors, got %d", len(ve.Fields))
	}
}

func TestValidationError_Err(t *testing.T) {
	ve := &ValidationError{}
	if err := ve.Err(); err != nil {
		t.Error("Err() should return nil when no errors present")
	}

	ve.Add(NonEmpty("name", ""))
	if err := ve.Err(); err == nil {
		t.Error("Err() should return non-nil when errors present")
	}
}

func TestNonEmpty(t *testing.T) {
	tests := []struct {
		name    string
		field   string
		value   string
		wantErr bool
	}{
		{"valid", "name", "John", false},
		{"empty", "name", "", true},
		{"whitespace", "name", "   ", true},
		{"single char", "name", "a", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NonEmpty(tt.field, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("NonEmpty() = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMaxLength(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		max     int
		wantErr bool
	}{
		{"valid", "hello", 10, false},
		{"exact", "hello", 5, false},
		{"too long", "hello", 4, true},
		{"empty", "", 10, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := MaxLength("test", tt.value, tt.max)
			if (err != nil) != tt.wantErr {
				t.Errorf("MaxLength() = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPositiveInt(t *testing.T) {
	tests := []struct {
		name    string
		value   int64
		wantErr bool
	}{
		{"positive", 100, false},
		{"one", 1, false},
		{"zero", 0, true},
		{"negative", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := PositiveInt("amount", tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("PositiveInt() = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMinInt(t *testing.T) {
	tests := []struct {
		name    string
		value   int64
		min     int64
		wantErr bool
	}{
		{"valid", 100, 50, false},
		{"exact", 50, 50, false},
		{"below min", 49, 50, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := MinInt("value", tt.value, tt.min)
			if (err != nil) != tt.wantErr {
				t.Errorf("MinInt() = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMaxInt(t *testing.T) {
	tests := []struct {
		name    string
		value   int64
		max     int64
		wantErr bool
	}{
		{"valid", 100, 150, false},
		{"exact", 150, 150, false},
		{"above max", 151, 150, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := MaxInt("value", tt.value, tt.max)
			if (err != nil) != tt.wantErr {
				t.Errorf("MaxInt() = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestISOCurrency(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"ARS", "ARS", false},
		{"ars lowercase", "ars", false},
		{"USD", "USD", false},
		{"BRL", "BRL", false},
		{"invalid", "XYZ", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ISOCurrency("currency", tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("ISOCurrency(%q) = %v, wantErr %v", tt.value, err, tt.wantErr)
			}
		})
	}
}

func TestValidEnum(t *testing.T) {
	allowed := []string{"active", "pending", "completed"}

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid", "active", false},
		{"valid second", "completed", false},
		{"invalid", "unknown", true},
		{"case sensitive", "Active", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidEnum("status", tt.value, allowed)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidEnum(%q) = %v, wantErr %v", tt.value, err, tt.wantErr)
			}
		})
	}
}

func TestValidUUID(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid", "550e8400-e29b-41d4-a716-446655440000", false},
		{"uppercase", "550E8400-E29B-41D4-A716-446655440000", false},
		{"no hyphens", "550e8400e29b41d4a716446655440000", true},
		{"short", "550e8400", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidUUID("id", tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidUUID(%q) = %v, wantErr %v", tt.value, err, tt.wantErr)
			}
		})
	}
}

func TestValidationError_Error(t *testing.T) {
	ve := &ValidationError{
		Fields: []FieldError{
			{Field: "name", Message: "required"},
			{Field: "amount", Message: "must be positive"},
		},
	}

	errMsg := ve.Error()
	if errMsg == "" {
		t.Error("Error() should return non-empty string")
	}
	if !strings.Contains(errMsg, "name") || !strings.Contains(errMsg, "required") {
		t.Errorf("Error() should contain field and message: %s", errMsg)
	}
}

func TestValidationError_HasErrors(t *testing.T) {
	ve := &ValidationError{}
	if ve.HasErrors() {
		t.Error("empty ValidationError should not have errors")
	}

	ve.Add(NonEmpty("name", ""))
	if !ve.HasErrors() {
		t.Error("ValidationError with fields should have errors")
	}
}
