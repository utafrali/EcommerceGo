package config

import (
	"fmt"

	pkgconfig "github.com/utafrali/EcommerceGo/pkg/config"
)

// Config holds all configuration for the order service.
type Config struct {
	Environment string `env:"ENVIRONMENT" envDefault:"development"`
	LogLevel    string `env:"LOG_LEVEL" envDefault:"info"`

	// HTTP server
	HTTPPort int `env:"ORDER_HTTP_PORT" envDefault:"8004"`

	// gRPC server
	GRPCPort int `env:"ORDER_GRPC_PORT" envDefault:"9004"`

	// PostgreSQL
	PostgresHost string `env:"POSTGRES_HOST" envDefault:"localhost"`
	PostgresPort int    `env:"POSTGRES_PORT" envDefault:"5432"`
	PostgresUser string `env:"POSTGRES_USER" envDefault:"ecommerce"`
	PostgresPass string `env:"POSTGRES_PASSWORD" envDefault:"ecommerce_secret"`
	PostgresDB   string `env:"ORDER_DB_NAME" envDefault:"order_db"`
	PostgresSSL  string `env:"POSTGRES_SSL_MODE" envDefault:"disable"`

	// Kafka
	KafkaBrokers []string `env:"KAFKA_BROKERS" envDefault:"localhost:9092" envSeparator:","`
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	cfg := &Config{}
	if err := pkgconfig.Load(cfg); err != nil {
		return nil, fmt.Errorf("load order config: %w", err)
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
