package config

import (
	"fmt"

	pkgconfig "github.com/utafrali/EcommerceGo/pkg/config"
)

// Config holds all configuration for the user service.
type Config struct {
	Environment string `env:"ENVIRONMENT" envDefault:"development"`
	LogLevel    string `env:"LOG_LEVEL" envDefault:"info"`

	// HTTP server
	HTTPPort int `env:"USER_HTTP_PORT" envDefault:"8006"`

	// gRPC server
	GRPCPort int `env:"USER_GRPC_PORT" envDefault:"9006"`

	// PostgreSQL
	PostgresHost string `env:"POSTGRES_HOST" envDefault:"localhost"`
	PostgresPort int    `env:"POSTGRES_PORT" envDefault:"5432"`
	PostgresUser string `env:"POSTGRES_USER" envDefault:"ecommerce"`
	PostgresPass string `env:"POSTGRES_PASSWORD" envDefault:"ecommerce_secret"`
	PostgresDB   string `env:"USER_DB_NAME" envDefault:"user_db"`
	PostgresSSL  string `env:"POSTGRES_SSL_MODE" envDefault:"disable"`

	// Kafka
	KafkaBrokers []string `env:"KAFKA_BROKERS" envDefault:"localhost:9092" envSeparator:","`

	// JWT
	JWTSecret        string `env:"JWT_SECRET" envDefault:"change-this-to-a-secure-secret"`
	JWTAccessExpiry  string `env:"JWT_ACCESS_TOKEN_EXPIRY" envDefault:"15m"`
	JWTRefreshExpiry string `env:"JWT_REFRESH_TOKEN_EXPIRY" envDefault:"168h"`

	// CORS
	CORSAllowedOrigins []string `env:"CORS_ALLOWED_ORIGINS" envDefault:"*" envSeparator:","`
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	cfg := &Config{}
	if err := pkgconfig.Load(cfg); err != nil {
		return nil, fmt.Errorf("load user config: %w", err)
	}
	if cfg.HTTPPort < 1 || cfg.HTTPPort > 65535 {
		return nil, fmt.Errorf("invalid HTTP port: %d", cfg.HTTPPort)
	}

	// In non-development environments, require an explicitly set, strong JWT secret.
	if cfg.Environment != "development" {
		if cfg.JWTSecret == "change-this-to-a-secure-secret" {
			return nil, fmt.Errorf("JWT_SECRET must be explicitly set via environment variable in %q mode", cfg.Environment)
		}
		if len(cfg.JWTSecret) < 32 {
			return nil, fmt.Errorf("JWT_SECRET must be at least 32 characters long, got %d", len(cfg.JWTSecret))
		}
	}

	return cfg, nil
}

// PostgresDSN returns the PostgreSQL connection string.
func (c *Config) PostgresDSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.PostgresUser, c.PostgresPass, c.PostgresHost, c.PostgresPort, c.PostgresDB, c.PostgresSSL,
	)
}
