package config

import (
	"fmt"

	pkgconfig "github.com/utafrali/EcommerceGo/pkg/config"
)

// Config holds all configuration for the cart service.
type Config struct {
	Environment string `env:"ENVIRONMENT" envDefault:"development"`
	LogLevel    string `env:"LOG_LEVEL" envDefault:"info"`

	// HTTP server
	HTTPPort int `env:"CART_HTTP_PORT" envDefault:"8003"`

	// Redis
	RedisAddr string `env:"REDIS_ADDR" envDefault:"localhost:6379"`
	RedisPass string `env:"REDIS_PASSWORD" envDefault:""`
	RedisDB   int    `env:"REDIS_DB" envDefault:"0"`

	// Cart TTL in hours (default: 7 days)
	CartTTL int `env:"CART_TTL_HOURS" envDefault:"168"`

	// Kafka
	KafkaBrokers []string `env:"KAFKA_BROKERS" envDefault:"localhost:9092" envSeparator:","`

	// OpenTelemetry
	OTELEnabled    bool    `env:"OTEL_ENABLED" envDefault:"false"`
	OTELEndpoint   string  `env:"OTEL_EXPORTER_OTLP_ENDPOINT" envDefault:"localhost:4318"`
	OTELSampleRate float64 `env:"OTEL_SAMPLE_RATE" envDefault:"1.0"`

	// Pprof debug endpoints (IP allowlist in CIDR notation)
	PprofAllowedCIDRs []string `env:"PPROF_ALLOWED_CIDRS" envDefault:"10.0.0.0/8,172.16.0.0/12,192.168.0.0/16,127.0.0.0/8,::1/128" envSeparator:","`
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	cfg := &Config{}
	if err := pkgconfig.Load(cfg); err != nil {
		return nil, fmt.Errorf("load cart config: %w", err)
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
	if len(c.KafkaBrokers) == 0 {
		return fmt.Errorf("KAFKA_BROKERS is required")
	}
	if c.OTELSampleRate < 0 || c.OTELSampleRate > 1.0 {
		return fmt.Errorf("OTEL_SAMPLE_RATE must be between 0.0 and 1.0, got %f", c.OTELSampleRate)
	}
	return nil
}
