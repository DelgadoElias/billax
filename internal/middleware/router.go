package middleware

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NewRouter creates a new chi router with all middlewares
func NewRouter(logger *slog.Logger, pool *pgxpool.Pool, rateLimitDefault int) chi.Router {
	r := chi.NewRouter()

	// Global middlewares (apply to all routes)
	r.Use(RequestID)
	r.Use(Logger(logger))
	r.Use(Recovery(logger))

	// Health check endpoint (no auth required)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","version":"0.1.0"}`))
	})

	// Protected routes under /v1
	r.Route("/v1", func(r chi.Router) {
		// Auth and rate limiting middlewares
		r.Use(AuthMiddleware(pool))
		rateLimiter := NewRateLimiter(rateLimitDefault)
		r.Use(RateLimitMiddleware(rateLimiter))

		// Test endpoint
		r.Get("/me", func(w http.ResponseWriter, r *http.Request) {
			tenantID := TenantIDFromContext(r.Context())
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"tenant_id":"` + tenantID.String() + `"}`))
		})
	})

	return r
}
