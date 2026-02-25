package config

import (
	"fmt"

	pkgconfig "github.com/utafrali/EcommerceGo/pkg/config"
)

// Config holds all configuration for the search service.
type Config struct {
	Environment string `env:"ENVIRONMENT" envDefault:"development"`
	LogLevel    string `env:"LOG_LEVEL" envDefault:"info"`

	// HTTP server
	HTTPPort int `env:"SEARCH_HTTP_PORT" envDefault:"8010"`

	// gRPC server
	GRPCPort int `env:"SEARCH_GRPC_PORT" envDefault:"9010"`

	// Elasticsearch (not used yet, for future integration)
	ElasticsearchURL string `env:"ELASTICSEARCH_URL" envDefault:"http://localhost:9200"`

	// Kafka
	KafkaBrokers []string `env:"KAFKA_BROKERS" envDefault:"localhost:9092" envSeparator:","`
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	cfg := &Config{}
	if err := pkgconfig.Load(cfg); err != nil {
		return nil, fmt.Errorf("load search config: %w", err)
	}
	return cfg, nil
}
