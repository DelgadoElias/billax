package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	// Required
	DatabaseURL string
	AppEnv      string // development|production

	// Optional with defaults
	Port                     int
	LogLevel                 string
	RateLimitDefault         int
	WebhookDeliveryTimeout   time.Duration
	WebhookMaxRetries        int
	ProvidersConfigPath      string // path to providers.yml
}

// Load reads configuration from environment variables with validation
func Load() (*Config, error) {
	// Load .env file in development
	if os.Getenv("APP_ENV") == "" || os.Getenv("APP_ENV") == "development" {
		_ = godotenv.Load()
	}

	cfg := &Config{
		DatabaseURL:  os.Getenv("DATABASE_URL"),
		AppEnv:       getEnv("APP_ENV", "development"),
		Port:         getEnvInt("PORT", 8080),
		LogLevel:     getEnv("LOG_LEVEL", "info"),
		RateLimitDefault: getEnvInt("RATE_LIMIT_DEFAULT", 100),
		WebhookDeliveryTimeout: getEnvDuration("WEBHOOK_DELIVERY_TIMEOUT", 10*time.Second),
		WebhookMaxRetries: getEnvInt("WEBHOOK_MAX_RETRIES", 5),
		ProvidersConfigPath: getEnv("PROVIDERS_CONFIG_PATH", "providers.yml"),
	}

	// Validate required fields
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
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
