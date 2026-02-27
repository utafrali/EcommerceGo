package config

import (
	"fmt"

	pkgconfig "github.com/utafrali/EcommerceGo/pkg/config"
)

// Config holds all configuration for the API gateway service.
type Config struct {
	Environment string `env:"ENVIRONMENT" envDefault:"development"`
	LogLevel    string `env:"LOG_LEVEL" envDefault:"info"`
	HTTPPort    int    `env:"GATEWAY_HTTP_PORT" envDefault:"8080"`

	// JWT authentication
	JWTSecret string `env:"JWT_SECRET" envDefault:"your-secret-key-change-in-production"`

	// Backend service URLs
	ProductServiceURL      string `env:"PRODUCT_SERVICE_URL" envDefault:"http://localhost:8001"`
	CartServiceURL         string `env:"CART_SERVICE_URL" envDefault:"http://localhost:8002"`
	OrderServiceURL        string `env:"ORDER_SERVICE_URL" envDefault:"http://localhost:8003"`
	CheckoutServiceURL     string `env:"CHECKOUT_SERVICE_URL" envDefault:"http://localhost:8004"`
	PaymentServiceURL      string `env:"PAYMENT_SERVICE_URL" envDefault:"http://localhost:8005"`
	UserServiceURL         string `env:"USER_SERVICE_URL" envDefault:"http://localhost:8006"`
	InventoryServiceURL    string `env:"INVENTORY_SERVICE_URL" envDefault:"http://localhost:8007"`
	CampaignServiceURL     string `env:"CAMPAIGN_SERVICE_URL" envDefault:"http://localhost:8008"`
	NotificationServiceURL string `env:"NOTIFICATION_SERVICE_URL" envDefault:"http://localhost:8009"`
	SearchServiceURL       string `env:"SEARCH_SERVICE_URL" envDefault:"http://localhost:8010"`
	MediaServiceURL        string `env:"MEDIA_SERVICE_URL" envDefault:"http://localhost:8011"`

	// Rate limiting
	RateLimitRPS   int `env:"RATE_LIMIT_RPS" envDefault:"100"`
	RateLimitBurst int `env:"RATE_LIMIT_BURST" envDefault:"200"`
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	cfg := &Config{}
	if err := pkgconfig.Load(cfg); err != nil {
		return nil, fmt.Errorf("load gateway config: %w", err)
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// validate checks configuration invariants.
func (c *Config) validate() error {
	if c.Environment != "development" && c.JWTSecret == "your-secret-key-change-in-production" {
		return fmt.Errorf("JWT_SECRET must be changed from default value in %s environment", c.Environment)
	}
	return nil
}
