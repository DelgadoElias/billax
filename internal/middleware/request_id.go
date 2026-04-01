package middleware

import (
	"net/http"

	"github.com/google/uuid"
)

// RequestID generates a unique ID for each request
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := "req_" + uuid.New().String()

		ctx := WithRequestID(r.Context(), requestID)
		w.Header().Set("X-Request-ID", requestID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
