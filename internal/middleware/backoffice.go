package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// BackofficeAuth middleware validates JWT tokens for backoffice requests and sets RLS context
// Expects: Authorization: Bearer <jwt_token>
// Sets context values: backoffice_user_id, backoffice_tenant_id, backoffice_role, tenant_id
// Also configures PostgreSQL RLS context for the request
//
// The middleware parses and validates JWT tokens without importing the backoffice package,
// avoiding import cycles while still providing JWT authentication.
func BackofficeAuth(pool *pgxpool.Pool, jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				writeErrorResponse(w, 401, "missing_auth_header", "Missing Authorization header")
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				writeErrorResponse(w, 401, "invalid_auth_header", "Invalid Authorization header format")
				return
			}

			tokenStr := parts[1]

			// Parse and validate JWT token
			claims := &jwt.MapClaims{}
			token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
				// Verify signing method
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				}
				return []byte(jwtSecret), nil
			})

			if err != nil || !token.Valid {
				writeErrorResponse(w, 401, "invalid_token", "Invalid or expired token")
				return
			}

			// Extract and validate claims
			userIDStr, ok := (*claims)["user_id"].(string)
			if !ok {
				writeErrorResponse(w, 401, "invalid_token", "Invalid token claims")
				return
			}
			userID, err := uuid.Parse(userIDStr)
			if err != nil {
				writeErrorResponse(w, 401, "invalid_token", "Invalid user ID in token")
				return
			}

			tenantIDStr, ok := (*claims)["tenant_id"].(string)
			if !ok {
				writeErrorResponse(w, 401, "invalid_token", "Invalid token claims")
				return
			}
			tenantID, err := uuid.Parse(tenantIDStr)
			if err != nil {
				writeErrorResponse(w, 401, "invalid_token", "Invalid tenant ID in token")
				return
			}

			roleStr, ok := (*claims)["role"].(string)
			if !ok {
				writeErrorResponse(w, 401, "invalid_token", "Invalid token claims")
				return
			}

			// Set backoffice context values
			ctx := r.Context()
			ctx = WithBackofficeUserID(ctx, userID)
			ctx = WithBackofficeTenantID(ctx, tenantID)
			ctx = WithBackofficeRole(ctx, roleStr) // Role is stored as string to avoid import cycle
			// Also set tenant_id for RLS consistency with API key auth
			ctx = WithTenantID(ctx, tenantID)

			// Set RLS context in the database (for the current request)
			if err := setRLSContext(ctx, pool, tenantID); err != nil {
				writeErrorResponse(w, 500, "internal_error", "Internal server error")
				return
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// Context key types for backoffice auth
type backofficeContextKey string

const (
	backofficeUserIDKey   backofficeContextKey = "backoffice_user_id"
	backofficeTenantIDKey backofficeContextKey = "backoffice_tenant_id"
	backofficeRoleKey     backofficeContextKey = "backoffice_role"
)

// WithBackofficeUserID adds backoffice user ID to context
func WithBackofficeUserID(ctx context.Context, userID uuid.UUID) context.Context {
	return context.WithValue(ctx, backofficeUserIDKey, userID)
}

// BackofficeUserIDFromContext retrieves backoffice user ID from context
func BackofficeUserIDFromContext(ctx context.Context) uuid.UUID {
	if id, ok := ctx.Value(backofficeUserIDKey).(uuid.UUID); ok {
		return id
	}
	return uuid.UUID{}
}

// WithBackofficeTenantID adds backoffice tenant ID to context
func WithBackofficeTenantID(ctx context.Context, tenantID uuid.UUID) context.Context {
	return context.WithValue(ctx, backofficeTenantIDKey, tenantID)
}

// BackofficeTenantIDFromContext retrieves backoffice tenant ID from context
func BackofficeTenantIDFromContext(ctx context.Context) uuid.UUID {
	if id, ok := ctx.Value(backofficeTenantIDKey).(uuid.UUID); ok {
		return id
	}
	return uuid.UUID{}
}

// WithBackofficeRole adds backoffice role to context
func WithBackofficeRole(ctx context.Context, role string) context.Context {
	return context.WithValue(ctx, backofficeRoleKey, role)
}

// BackofficeRoleFromContext retrieves backoffice role from context as a string
func BackofficeRoleFromContext(ctx context.Context) string {
	if role, ok := ctx.Value(backofficeRoleKey).(string); ok {
		return role
	}
	return ""
}
