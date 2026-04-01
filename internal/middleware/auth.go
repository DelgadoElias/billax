package middleware

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/argon2"
)

const (
	argonTime    uint32 = 1
	argonMemory  uint32 = 64 * 1024
	argonThreads uint8  = 4
	argonKeyLen  uint32 = 32
)

// AuthMiddleware validates API key and sets tenant context
func AuthMiddleware(pool *pgxpool.Pool) func(http.Handler) http.Handler {
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

			token := parts[1]

			// Extract key prefix (first 12 chars)
			if len(token) < 12 {
				writeErrorResponse(w, 401, "invalid_api_key", "Invalid API key format")
				return
			}

			keyPrefix := token[:12]

			// Query database for API key
			ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
			defer cancel()

			var tenantID uuid.UUID
			var keyHash string
			var expiresAt sql.NullTime
			var isActive bool

			err := pool.QueryRow(ctx,
				`SELECT tenant_id, key_hash, expires_at, is_active
				 FROM tenant_api_keys
				 WHERE key_prefix = $1 AND is_active = true`,
				keyPrefix,
			).Scan(&tenantID, &keyHash, &expiresAt, &isActive)

			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					writeErrorResponse(w, 401, "invalid_api_key", "Invalid API key")
					return
				}
				writeErrorResponse(w, 500, "internal_error", "Internal server error")
				return
			}

			// Check if key is active
			if !isActive {
				writeErrorResponse(w, 401, "api_key_inactive", "API key is inactive")
				return
			}

			// Check if key has expired
			if expiresAt.Valid && expiresAt.Time.Before(time.Now()) {
				writeErrorResponse(w, 401, "api_key_expired", "API key has expired")
				return
			}

			// Verify hash
			hash := argon2.IDKey([]byte(token), []byte(keyPrefix), argonTime, argonMemory, argonThreads, argonKeyLen)
			hashStr := hashToString(hash)

			if hashStr != keyHash {
				writeErrorResponse(w, 401, "invalid_api_key", "Invalid API key")
				return
			}

			// Set tenant ID in context and database context
			ctx = WithTenantID(r.Context(), tenantID)

			// Set RLS context in the database
			// Note: This is a simplified version. In production, you'd want to use a transaction
			// and set the context for the database connection pool.
			err = setRLSContext(ctx, pool, tenantID)
			if err != nil {
				writeErrorResponse(w, 500, "internal_error", "Internal server error")
				return
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// setRLSContext sets the RLS context in the database
func setRLSContext(ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID) error {
	// For now, we'll just set it in a simple query
	// In a real implementation, you'd want to use a transaction
	_, err := pool.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", tenantID.String())
	return err
}

// hashToString converts a hash byte slice to a hex string
func hashToString(hash []byte) string {
	var result string
	for _, b := range hash {
		result += string(rune(b))
	}
	return result
}

// writeErrorResponse writes a standard error response
func writeErrorResponse(w http.ResponseWriter, statusCode int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write([]byte(`{"error":{"code":"` + code + `","message":"` + message + `"}}`))
}
