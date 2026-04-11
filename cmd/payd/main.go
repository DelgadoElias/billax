package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/DelgadoElias/billax/internal/config"
	"github.com/DelgadoElias/billax/internal/db"
	"github.com/DelgadoElias/billax/internal/metrics"
	"github.com/DelgadoElias/billax/internal/middleware"
	"github.com/DelgadoElias/billax/internal/payment"
	"github.com/DelgadoElias/billax/internal/plan"
	"github.com/DelgadoElias/billax/internal/provider"
	"github.com/DelgadoElias/billax/internal/provider/mercadopago"
	"github.com/DelgadoElias/billax/internal/providercredentials"
	"github.com/DelgadoElias/billax/internal/subscription"
	"github.com/DelgadoElias/billax/internal/tenant"
)

// version is set via ldflags during build: -X 'main.version=x.y.z'
var version = "dev"

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Set version from ldflags
	cfg.AppVersion = version

	// Initialize logger
	logger := initLogger(cfg.LogLevel, cfg.AppEnv)

	// Run database migrations
	if err := db.RunMigrations(cfg.DatabaseURL, cfg.MigrationsPath); err != nil {
		logger.Error("migration failed", "error", err)
		os.Exit(1)
	}
	logger.Info("migrations applied successfully")

	// Create database connection pool
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	cancel()
	if err != nil {
		logger.Error("failed to initialize database", "error", err)
		os.Exit(1)
	}
	defer db.Close(pool)

	logger.Info("database connected successfully")

	// Load provider capabilities from YAML
	yamlCaps, err := provider.LoadCapabilitiesFile(cfg.ProvidersConfigPath)
	if err != nil {
		logger.Error("failed to load provider capabilities", "error", err)
		os.Exit(1)
	}

	// Initialize provider layer
	registry := provider.NewRegistry()
	registry.Register(mercadopago.New())
	adapter := provider.NewAdapter(registry, yamlCaps)

	// Initialize repositories
	planRepo := plan.NewRepository(pool)
	subRepo := subscription.NewRepository(pool)
	paymentRepo := payment.NewRepository(pool)
	credRepo := providercredentials.NewRepository(pool)
	tenantRepo := tenant.NewRepository(pool)

	// Initialize services
	planSvc := plan.NewService(planRepo)
	subSvc := subscription.NewService(subRepo, planRepo, paymentRepo, adapter)
	paySvc := payment.NewService(paymentRepo, adapter)
	credSvc := providercredentials.NewService(credRepo, adapter)
	tenantSvc := tenant.NewService(tenantRepo, cfg.AppEnv)

	// Initialize handlers
	planHandler := plan.NewHandler(planSvc)
	subHandler := subscription.NewHandler(subSvc)
	paymentHandler := payment.NewHandler(paySvc, credSvc)
	credHandler := providercredentials.NewHandler(credSvc)
	tenantHandler := tenant.NewHandler(tenantSvc)

	// Create router with public and protected routes
	router := middleware.NewRouterWithPublicRoutes(logger, pool, cfg.RateLimitDefault, cfg.MetricsEnabled, cfg.AppVersion,
		// Public routes (no auth required)
		func(r chi.Router) {
			tenantHandler.RegisterRoutes(r)
		},
		// Protected routes (auth required)
		func(r chi.Router) {
			credHandler.RegisterRoutes(r)
			planHandler.RegisterRoutes(r)
			paymentHandler.RegisterRoutes(r)
			subHandler.RegisterRoutes(r)
			tenantHandler.RegisterAuthRoutes(r)
		},
	)

	// Create context for background tasks (poller, etc.)
	// This will be cancelled during graceful shutdown
	pollerCtx, pollerCancel := context.WithCancel(context.Background())

	// Start background poller for subscription metrics
	go startSubscriptionMetricsPoller(pollerCtx, logger, subRepo)

	// Start HTTP server
	addr := net.JoinHostPort("", fmt.Sprintf("%d", cfg.Port))
	server := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	// Start metrics server if enabled
	var metricsServer *http.Server
	if cfg.MetricsEnabled {
		metricsAddr := net.JoinHostPort("", strconv.Itoa(cfg.MetricsPort))
		metricsServer = &http.Server{
			Addr:    metricsAddr,
			Handler: promhttp.Handler(),
		}
		go func() {
			logger.Info("starting metrics server", "addr", metricsAddr)
			if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.Error("metrics server error", "error", err)
			}
		}()
	}

	// Start app server in a goroutine
	go func() {
		logger.Info("starting server", "addr", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	// Graceful shutdown with timeout
	logger.Info("shutting down server")
	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Cancel background tasks (poller)
	pollerCancel()

	// Shutdown app server
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("shutdown error", "error", err)
		os.Exit(1)
	}

	// Shutdown metrics server if running
	if metricsServer != nil {
		if err := metricsServer.Shutdown(ctx); err != nil {
			logger.Error("metrics server shutdown error", "error", err)
		}
	}

	logger.Info("server stopped")
}

func initLogger(level, appEnv string) *slog.Logger {
	var logLevel slog.Level
	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: logLevel}

	// Use JSON logging in production for log aggregators (ELK, Datadog, etc.)
	// Use text logging in development for human readability
	var handler slog.Handler
	if appEnv == "production" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return slog.New(handler)
}

// startSubscriptionMetricsPoller polls the database for subscription counts
// and updates the active subscriptions gauge every 30 seconds
func startSubscriptionMetricsPoller(ctx context.Context, logger *slog.Logger, subRepo subscription.SubscriptionRepo) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			counts, err := subRepo.CountByStatus(ctx)
			if err != nil {
				logger.Warn("failed to update subscription metrics", "error", err)
				continue
			}

			// Reset all known status gauges, then set observed values
			for _, status := range []string{"trialing", "active", "past_due", "canceled", "expired"} {
				metrics.ActiveSubscriptions.WithLabelValues(status).Set(0)
			}

			// Set observed counts
			for status, count := range counts {
				metrics.ActiveSubscriptions.WithLabelValues(string(status)).Set(float64(count))
			}

		case <-ctx.Done():
			logger.Debug("subscription metrics poller stopped")
			return
		}
	}
}

