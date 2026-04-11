package middleware

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NewRouter creates a new chi router with all middlewares
// registerPublicRoutes is a callback to mount public /v1 routes (no auth)
// registerDomainRoutes is a callback to mount protected /v1 routes (with auth)
func NewRouter(logger *slog.Logger, pool *pgxpool.Pool, rateLimitDefault int, metricsEnabled bool, version string, registerDomainRoutes func(r chi.Router)) chi.Router {
	return newRouter(logger, pool, rateLimitDefault, metricsEnabled, version, nil, registerDomainRoutes, nil, "")
}

// NewRouterWithPublicRoutes creates a router with both public and protected routes
func NewRouterWithPublicRoutes(logger *slog.Logger, pool *pgxpool.Pool, rateLimitDefault int, metricsEnabled bool, version string, registerPublicRoutes, registerDomainRoutes func(r chi.Router)) chi.Router {
	return NewRouterWithPublicRoutesAndBackoffice(logger, pool, rateLimitDefault, metricsEnabled, version, registerPublicRoutes, registerDomainRoutes, nil, "")
}

// NewRouterWithPublicRoutesAndBackoffice creates a router with public, protected, and backoffice routes
// backofficeJWTSecret is the secret for backoffice JWT authentication
// registerBackofficeAuthRoutes is a callback to mount protected backoffice routes (with JWT auth)
func NewRouterWithPublicRoutesAndBackoffice(logger *slog.Logger, pool *pgxpool.Pool, rateLimitDefault int, metricsEnabled bool, version string, registerPublicRoutes, registerDomainRoutes, registerBackofficeAuthRoutes func(r chi.Router), backofficeJWTSecret string) chi.Router {
	return newRouter(logger, pool, rateLimitDefault, metricsEnabled, version, registerPublicRoutes, registerDomainRoutes, registerBackofficeAuthRoutes, backofficeJWTSecret)
}

func newRouter(logger *slog.Logger, pool *pgxpool.Pool, rateLimitDefault int, metricsEnabled bool, version string, registerPublicRoutes, registerDomainRoutes, registerBackofficeAuthRoutes func(r chi.Router), backofficeJWTSecret string) chi.Router {
	r := chi.NewRouter()

	// Global middlewares (apply to all routes)
	// Metrics middleware must wrap all handlers to accurately measure latency
	if metricsEnabled {
		r.Use(MetricsMiddleware)
	}
	r.Use(RequestID)
	r.Use(Logger(logger))
	r.Use(Recovery(logger))

	// Health check endpoint (no auth required)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","version":"` + version + `"}`))
	})

	// Setup rate limiter (shared across all /v1 routes)
	rateLimiter := NewRateLimiter(rateLimitDefault)

	// All /v1 routes under one group
	r.Route("/v1", func(r chi.Router) {
		// Rate limiting for all /v1 routes
		r.Use(RateLimitMiddleware(rateLimiter))

		// Public routes first (no auth)
		if registerPublicRoutes != nil {
			registerPublicRoutes(r)
		}

		// Protected routes under /v1 with API key auth
		// Create a new subrouter for authenticated endpoints
		r.Group(func(r chi.Router) {
			r.Use(AuthMiddleware(pool))

			// Test endpoint
			r.Get("/me", func(w http.ResponseWriter, r *http.Request) {
				tenantID := TenantIDFromContext(r.Context())
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"tenant_id":"` + tenantID.String() + `"}`))
			})

			// Mount domain routes via callback (for authenticated endpoints)
			if registerDomainRoutes != nil {
				registerDomainRoutes(r)
			}
		})

		// Backoffice protected routes with JWT auth (separate from API key auth)
		if registerBackofficeAuthRoutes != nil && backofficeJWTSecret != "" {
			r.Group(func(r chi.Router) {
				r.Use(BackofficeAuth(pool, backofficeJWTSecret))

				// Mount backoffice auth routes via callback
				registerBackofficeAuthRoutes(r)
			})
		}
	})

	return r
}
