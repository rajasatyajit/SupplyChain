package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all application configuration
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Pipeline PipelineConfig
	Logging  LoggingConfig
	Metrics  MetricsConfig
	Auth     AuthConfig
	Admin    AdminConfig
	Redis    RedisConfig
	Billing  BillingConfig
}

type ServerConfig struct {
	Host                    string
	Port                    int
	ReadTimeout             time.Duration
	WriteTimeout            time.Duration
	IdleTimeout             time.Duration
	GracefulShutdownTimeout time.Duration
}

type DatabaseConfig struct {
	URL             string
	MaxConns        int
	MinConns        int
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
}

type PipelineConfig struct {
	RateLimit     float64
	WorkerCount   int
	BatchSize     int
	RetryAttempts int
	RetryDelay    time.Duration
}

type LoggingConfig struct {
	Level  string
	Format string // json or text
}

type MetricsConfig struct {
	Enabled bool
	Port    int
	Path    string
}

type AuthConfig struct {
	RequireAPIKeys   bool
	KeyHeader        string // default: Authorization Bearer <key>
	AgentHeaderName  string // optional: X-Client-Type
	EnableAgentHeader bool  // if true, require AgentHeaderName to be one of [agent,human]
}

type RedisConfig struct {
	URL      string
	Password string
	DB       int
}

type AdminConfig struct {
	AdminSecret string
}

type BillingConfig struct {
	StripePublicKey string
	StripeSecretKey string
	StripeWebhookSecret string
	PriceLiteMonthly string
	PriceLiteAnnual  string
	PriceProMonthly  string
	PriceProAnnual   string
	PriceOverageMetered string
	CheckoutSuccessURL string
	CheckoutCancelURL  string
	PortalReturnURL    string
	OveragePricePerRequestUSD float64 // e.g., 0.000033 for 0.0033 cents
}

// Load loads configuration from environment variables with sensible defaults
func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Host:                    getEnv("SERVER_HOST", "0.0.0.0"),
			Port:                    getEnvInt("SERVER_PORT", 8080),
			ReadTimeout:             getEnvDuration("SERVER_READ_TIMEOUT", 30*time.Second),
			WriteTimeout:            getEnvDuration("SERVER_WRITE_TIMEOUT", 30*time.Second),
			IdleTimeout:             getEnvDuration("SERVER_IDLE_TIMEOUT", 120*time.Second),
			GracefulShutdownTimeout: getEnvDuration("SERVER_GRACEFUL_SHUTDOWN_TIMEOUT", 30*time.Second),
		},
		Database: DatabaseConfig{
			URL:             getEnv("DATABASE_URL", ""),
			MaxConns:        getEnvInt("DB_MAX_CONNS", 25),
			MinConns:        getEnvInt("DB_MIN_CONNS", 5),
			MaxConnLifetime: getEnvDuration("DB_MAX_CONN_LIFETIME", 1*time.Hour),
			MaxConnIdleTime: getEnvDuration("DB_MAX_CONN_IDLE_TIME", 30*time.Minute),
		},
		Pipeline: PipelineConfig{
			RateLimit:     getEnvFloat("PIPELINE_RATE_LIMIT", 5.0),
			WorkerCount:   getEnvInt("PIPELINE_WORKER_COUNT", 4),
			BatchSize:     getEnvInt("PIPELINE_BATCH_SIZE", 100),
			RetryAttempts: getEnvInt("PIPELINE_RETRY_ATTEMPTS", 3),
			RetryDelay:    getEnvDuration("PIPELINE_RETRY_DELAY", 5*time.Second),
		},
		Logging: LoggingConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"),
		},
		Metrics: MetricsConfig{
			Enabled: getEnvBool("METRICS_ENABLED", true),
			Port:    getEnvInt("METRICS_PORT", 9090),
			Path:    getEnv("METRICS_PATH", "/metrics"),
		},
		Auth: AuthConfig{
			RequireAPIKeys:    getEnvBool("AUTH_REQUIRE_API_KEYS", false),
			KeyHeader:         getEnv("AUTH_KEY_HEADER", "Authorization"),
			AgentHeaderName:   getEnv("AUTH_AGENT_HEADER", "X-Client-Type"),
			EnableAgentHeader: getEnvBool("AUTH_ENABLE_AGENT_HEADER", true),
		},
		Redis: RedisConfig{
			URL:      getEnv("REDIS_URL", ""),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
		},
		Admin: AdminConfig{
			AdminSecret: getEnv("ADMIN_SECRET", ""),
		},
		Billing: BillingConfig{
			StripePublicKey:        getEnv("STRIPE_PUBLIC_KEY", ""),
			StripeSecretKey:        getEnv("STRIPE_SECRET_KEY", ""),
			StripeWebhookSecret:    getEnv("STRIPE_WEBHOOK_SECRET", ""),
			PriceLiteMonthly:       getEnv("STRIPE_PRICE_LITE_MONTHLY", ""),
			PriceLiteAnnual:        getEnv("STRIPE_PRICE_LITE_ANNUAL", ""),
			PriceProMonthly:        getEnv("STRIPE_PRICE_PRO_MONTHLY", ""),
			PriceProAnnual:         getEnv("STRIPE_PRICE_PRO_ANNUAL", ""),
			PriceOverageMetered:    getEnv("STRIPE_PRICE_OVERAGE_METERED", ""),
			CheckoutSuccessURL:     getEnv("STRIPE_CHECKOUT_SUCCESS_URL", "https://dashboard.example.com/billing/success"),
			CheckoutCancelURL:      getEnv("STRIPE_CHECKOUT_CANCEL_URL", "https://dashboard.example.com/billing/cancel"),
			PortalReturnURL:        getEnv("STRIPE_PORTAL_RETURN_URL", "https://dashboard.example.com/billing"),
			OveragePricePerRequestUSD: getEnvFloat("BILLING_OVERAGE_PRICE_PER_REQUEST_USD", 0.000033),
		},
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}
	if c.Database.MaxConns < 1 {
		return fmt.Errorf("database max connections must be at least 1")
	}
	if c.Pipeline.WorkerCount < 1 {
		return fmt.Errorf("pipeline worker count must be at least 1")
	}
	return nil
}

// Helper functions for environment variable parsing
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseFloat(value, 64); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if parsed, err := time.ParseDuration(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}
