package config

import (
	"fmt"

	pkgconfig "github.com/utafrali/EcommerceGo/pkg/config"
)

// Config holds all configuration for the payment service.
type Config struct {
	Environment string `env:"ENVIRONMENT" envDefault:"development"`
	LogLevel    string `env:"LOG_LEVEL" envDefault:"info"`

	// HTTP server
	HTTPPort int `env:"PAYMENT_HTTP_PORT" envDefault:"8005"`

	// gRPC server
	GRPCPort int `env:"PAYMENT_GRPC_PORT" envDefault:"9005"`

	// PostgreSQL
	PostgresHost string `env:"POSTGRES_HOST" envDefault:"localhost"`
	PostgresPort int    `env:"POSTGRES_PORT" envDefault:"5432"`
	PostgresUser string `env:"POSTGRES_USER" envDefault:"ecommerce"`
	PostgresPass string `env:"POSTGRES_PASSWORD" envDefault:"ecommerce_secret"`
	PostgresDB   string `env:"PAYMENT_DB_NAME" envDefault:"payment_db"`
	PostgresSSL  string `env:"POSTGRES_SSL_MODE" envDefault:"disable"`

	// Database pool
	DBMaxConns            int32 `env:"DB_MAX_CONNS" envDefault:"25"`
	DBMinConns            int32 `env:"DB_MIN_CONNS" envDefault:"5"`
	DBMaxConnLifetimeMins int `env:"DB_MAX_CONN_LIFETIME_MINUTES" envDefault:"60"`
	DBMaxConnIdleTimeMins int `env:"DB_MAX_CONN_IDLE_TIME_MINUTES" envDefault:"30"`

	// Kafka
	KafkaBrokers []string `env:"KAFKA_BROKERS" envDefault:"localhost:9092" envSeparator:","`

	// OpenTelemetry
	OTELEnabled    bool    `env:"OTEL_ENABLED" envDefault:"false"`
	OTELEndpoint   string  `env:"OTEL_EXPORTER_OTLP_ENDPOINT" envDefault:"localhost:4318"`
	OTELSampleRate float64 `env:"OTEL_SAMPLE_RATE" envDefault:"1.0"`

	// Pprof debug endpoints (IP allowlist in CIDR notation)
	PprofAllowedCIDRs []string `env:"PPROF_ALLOWED_CIDRS" envDefault:"10.0.0.0/8,172.16.0.0/12,192.168.0.0/16,127.0.0.0/8,::1/128" envSeparator:","`

	// Slow query logging
	SlowQueryThresholdMs int `env:"LOG_SLOW_QUERY_MS" envDefault:"500"`
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	cfg := &Config{}
	if err := pkgconfig.Load(cfg); err != nil {
		return nil, fmt.Errorf("load payment config: %w", err)
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// validate checks configuration invariants.
func (c *Config) validate() error {
	if c.HTTPPort < 1 || c.HTTPPort > 65535 {
		return fmt.Errorf("invalid HTTP port: %d", c.HTTPPort)
	}
	if c.GRPCPort < 1 || c.GRPCPort > 65535 {
		return fmt.Errorf("invalid gRPC port: %d", c.GRPCPort)
	}
	if c.PostgresHost == "" {
		return fmt.Errorf("POSTGRES_HOST is required")
	}
	if c.PostgresUser == "" {
		return fmt.Errorf("POSTGRES_USER is required")
	}
	if len(c.KafkaBrokers) == 0 {
		return fmt.Errorf("KAFKA_BROKERS is required")
	}
	if c.OTELSampleRate < 0 || c.OTELSampleRate > 1.0 {
		return fmt.Errorf("OTEL_SAMPLE_RATE must be between 0.0 and 1.0, got %f", c.OTELSampleRate)
	}
	return nil
}

// PostgresDSN returns the PostgreSQL connection string.
func (c *Config) PostgresDSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.PostgresUser, c.PostgresPass, c.PostgresHost, c.PostgresPort, c.PostgresDB, c.PostgresSSL,
	)
}
