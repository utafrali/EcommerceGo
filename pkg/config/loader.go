package config

import (
	"fmt"

	"github.com/caarlos0/env/v10"
)

// Load parses environment variables into the provided struct.
// The struct should use `env` tags to define mappings.
//
// Example:
//
//	type Config struct {
//	    Port     int    `env:"HTTP_PORT" envDefault:"8080"`
//	    LogLevel string `env:"LOG_LEVEL" envDefault:"info"`
//	}
func Load(cfg any) error {
	if err := env.Parse(cfg); err != nil {
		return fmt.Errorf("parse config: %w", err)
	}
	return nil
}
