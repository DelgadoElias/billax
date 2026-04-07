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
	"github.com/DelgadoElias/billax/internal/middleware"
	"github.com/DelgadoElias/billax/internal/payment"
	"github.com/DelgadoElias/billax/internal/plan"
	"github.com/DelgadoElias/billax/internal/provider"
	"github.com/DelgadoElias/billax/internal/provider/mercadopago"
	"github.com/DelgadoElias/billax/internal/providercredentials"
	"github.com/DelgadoElias/billax/internal/subscription"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Initialize logger
	logger := initLogger(cfg.LogLevel)

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

	// Initialize services
	planSvc := plan.NewService(planRepo)
	subSvc := subscription.NewService(subRepo, planRepo, paymentRepo, adapter)
	paySvc := payment.NewService(paymentRepo, adapter)
	credSvc := providercredentials.NewService(credRepo, adapter)

	// Initialize handlers
	planHandler := plan.NewHandler(planSvc)
	subHandler := subscription.NewHandler(subSvc)
	paymentHandler := payment.NewHandler(paySvc)
	credHandler := providercredentials.NewHandler(credSvc)

	// Create router with domain routes registration callback
	router := middleware.NewRouter(logger, pool, cfg.RateLimitDefault, cfg.MetricsEnabled, func(r chi.Router) {
		credHandler.RegisterRoutes(r)
		planHandler.RegisterRoutes(r)
		subHandler.RegisterRoutes(r)
		paymentHandler.RegisterRoutes(r)
	})

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

func initLogger(level string) *slog.Logger {
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

	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))
}

