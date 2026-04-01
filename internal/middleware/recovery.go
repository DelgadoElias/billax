package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
)

// Recovery recovers from panics and logs the stack trace
func Recovery(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					requestID := RequestIDFromContext(r.Context())
					logger.Error("panic recovered",
						"error", fmt.Sprint(err),
						"request_id", requestID,
						"path", r.URL.Path,
						"method", r.Method,
						"stack", string(debug.Stack()),
					)

					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(`{"error":{"code":"internal_error","message":"internal server error"}}`))
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
