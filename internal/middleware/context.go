package middleware

import (
	"context"

	"github.com/google/uuid"
)

type contextKey string

const (
	requestIDKey contextKey = "request_id"
	tenantIDKey  contextKey = "tenant_id"
)

// WithRequestID adds request ID to context
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// RequestIDFromContext retrieves request ID from context
func RequestIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}

// WithTenantID adds tenant ID to context
func WithTenantID(ctx context.Context, tenantID uuid.UUID) context.Context {
	return context.WithValue(ctx, tenantIDKey, tenantID)
}

// TenantIDFromContext retrieves tenant ID from context
func TenantIDFromContext(ctx context.Context) uuid.UUID {
	if id, ok := ctx.Value(tenantIDKey).(uuid.UUID); ok {
		return id
	}
	return uuid.UUID{}
}
