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

	// Elasticsearch
	ElasticsearchURL string `env:"ELASTICSEARCH_URL" envDefault:"http://localhost:9200"`

	// Search engine selection (elasticsearch or memory)
	SearchEngine string `env:"SEARCH_ENGINE" envDefault:"elasticsearch"`

	// Product service URL for reindex fetching
	ProductServiceURL string `env:"PRODUCT_SERVICE_URL" envDefault:"http://localhost:8080"`

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
