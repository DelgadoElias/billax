package config

import (
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

type Config struct {
	// Required
	DatabaseURL string
	AppEnv      string // development|production
	AppVersion  string // version injected via ldflags during build

	// Optional with defaults
	Port                      int
	LogLevel                  string
	RateLimitDefault          int
	WebhookDeliveryTimeout    time.Duration
	WebhookMaxRetries         int
	ProvidersConfigPath       string // path to providers.yml
	MigrationsPath            string // path to migrations directory
	BillaxConfigPath          string // path to billax_config.yml
	MetricsEnabled            bool
	MetricsPort               int
	CredentialsEncryptionKey  []byte        // 32 bytes from CREDENTIALS_ENCRYPTION_KEY hex env var
	LifecycleJobInterval      time.Duration // interval for subscription lifecycle jobs (renewals, expiry)
	PastDueGracePeriodDays    int           // grace period before expiring past_due subscriptions
	BackofficeJWTSecret       string        // secret for backoffice JWT token signing (required if backoffice is active)
	BackofficeJWTTTL          time.Duration // TTL for backoffice JWT tokens (default 24h)

	// Bootstrap: auto-create first tenant on startup (like Grafana)
	BootstrapTenantName    string // BOOTSTRAP_TENANT_NAME
	BootstrapTenantEmail   string // BOOTSTRAP_TENANT_EMAIL
	BootstrapAdminPassword string // BOOTSTRAP_ADMIN_PASSWORD
	BootstrapTenantSlug    string // BOOTSTRAP_TENANT_SLUG (optional)

	// Billax config from billax_config.yml
	AllowSignup             bool // allow self-service signup via POST /v1/signup
	SignupRequiresPassword  bool // require password field in signup (creates backoffice user)
}

// BillaxConfigFile represents the billax_config.yml structure
type BillaxConfigFile struct {
	Auth struct {
		AllowSignup            bool `yaml:"allow_signup"`
		SignupRequiresPassword bool `yaml:"signup_requires_password"`
	} `yaml:"auth"`
}

// Load reads configuration from environment variables with validation
func Load() (*Config, error) {
	// Load .env file in development
	if os.Getenv("APP_ENV") == "" || os.Getenv("APP_ENV") == "development" {
		_ = godotenv.Load()
	}

	cfg := &Config{
		DatabaseURL:         os.Getenv("DATABASE_URL"),
		AppEnv:              getEnv("APP_ENV", "development"),
		Port:                getEnvInt("PORT", 8080),
		LogLevel:            getEnv("LOG_LEVEL", "info"),
		RateLimitDefault:    getEnvInt("RATE_LIMIT_DEFAULT", 100),
		WebhookDeliveryTimeout: getEnvDuration("WEBHOOK_DELIVERY_TIMEOUT", 10*time.Second),
		WebhookMaxRetries:   getEnvInt("WEBHOOK_MAX_RETRIES", 5),
		ProvidersConfigPath: getEnv("PROVIDERS_CONFIG_PATH", "providers.yml"),
		MigrationsPath:      getEnv("MIGRATIONS_PATH", "migrations"),
		BillaxConfigPath:    getEnv("BILLAX_CONFIG_PATH", "billax_config.yml"),
		MetricsEnabled:      getEnvBool("METRICS_ENABLED", true),
		MetricsPort:         getEnvInt("METRICS_PORT", 9090),
		LifecycleJobInterval:  getEnvDuration("LIFECYCLE_JOB_INTERVAL", 5*time.Minute),
		PastDueGracePeriodDays: getEnvInt("PAST_DUE_GRACE_PERIOD_DAYS", 7),
		BackofficeJWTSecret:    getEnv("BACKOFFICE_JWT_SECRET", ""),
		BackofficeJWTTTL:       getEnvDuration("BACKOFFICE_JWT_TTL", 24*time.Hour),
		BootstrapTenantName:    os.Getenv("BOOTSTRAP_TENANT_NAME"),
		BootstrapTenantEmail:   os.Getenv("BOOTSTRAP_TENANT_EMAIL"),
		BootstrapAdminPassword: os.Getenv("BOOTSTRAP_ADMIN_PASSWORD"),
		BootstrapTenantSlug:    os.Getenv("BOOTSTRAP_TENANT_SLUG"),
		AllowSignup:            true,             // safe default: allow signups
		SignupRequiresPassword: true,             // safe default: require password
	}

	// Load billax_config.yml if it exists
	if _, err := os.Stat(cfg.BillaxConfigPath); err == nil {
		data, err := os.ReadFile(cfg.BillaxConfigPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", cfg.BillaxConfigPath, err)
		}

		var billaxCfg BillaxConfigFile
		if err := yaml.Unmarshal(data, &billaxCfg); err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", cfg.BillaxConfigPath, err)
		}

		cfg.AllowSignup = billaxCfg.Auth.AllowSignup
		cfg.SignupRequiresPassword = billaxCfg.Auth.SignupRequiresPassword
	}

	// Parse encryption key from hex (optional, required in production)
	if hexKey := os.Getenv("CREDENTIALS_ENCRYPTION_KEY"); hexKey != "" {
		key, err := hex.DecodeString(hexKey)
		if err != nil {
			return nil, fmt.Errorf("CREDENTIALS_ENCRYPTION_KEY must be valid hex: %w", err)
		}
		if len(key) != 32 {
			return nil, fmt.Errorf("CREDENTIALS_ENCRYPTION_KEY must be 64 hex chars (32 bytes), got %d chars", len(hexKey))
		}
		cfg.CredentialsEncryptionKey = key
	}

	// Validate required fields
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	// Validate production requirements
	if cfg.AppEnv == "production" && len(cfg.CredentialsEncryptionKey) == 0 {
		return nil, fmt.Errorf("CREDENTIALS_ENCRYPTION_KEY is required in production")
	}

	// Validate backoffice JWT secret if backoffice is enabled
	if cfg.BackofficeJWTSecret == "" {
		if cfg.AppEnv == "production" {
			return nil, fmt.Errorf("BACKOFFICE_JWT_SECRET is required in production")
		}
		// Use a default in development for convenience
		cfg.BackofficeJWTSecret = "dev-jwt-secret-change-in-production"
	}

	return cfg, nil
}

// Helper functions
func getEnv(key, defaultValue string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value, ok := os.LookupEnv(key); ok {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value, ok := os.LookupEnv(key); ok {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value, ok := os.LookupEnv(key); ok {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}
