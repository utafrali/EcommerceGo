package config

import (
	"fmt"
	"time"

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

	// Proxy transport
	ProxyDialTimeout     time.Duration `env:"PROXY_DIAL_TIMEOUT" envDefault:"5s"`
	ProxyResponseTimeout time.Duration `env:"PROXY_RESPONSE_TIMEOUT" envDefault:"30s"`
	ProxyIdleTimeout     time.Duration `env:"PROXY_IDLE_TIMEOUT" envDefault:"90s"`
	ProxyMaxIdleConns    int           `env:"PROXY_MAX_IDLE_CONNS" envDefault:"100"`

	// Rate limiting
	RateLimitRPS   int `env:"RATE_LIMIT_RPS" envDefault:"100"`
	RateLimitBurst int `env:"RATE_LIMIT_BURST" envDefault:"200"`

	// CORS
	CORSAllowedOrigins []string `env:"CORS_ALLOWED_ORIGINS" envDefault:"*" envSeparator:","`
	CORSAllowedMethods []string `env:"CORS_ALLOWED_METHODS" envDefault:"GET,POST,PUT,PATCH,DELETE,OPTIONS" envSeparator:","`
	CORSAllowedHeaders []string `env:"CORS_ALLOWED_HEADERS" envDefault:"Accept,Authorization,Content-Type,X-Correlation-ID,X-User-ID" envSeparator:","`
	CORSMaxAge         int      `env:"CORS_MAX_AGE" envDefault:"3600"`

	// Metrics endpoint protection (IP allowlist in CIDR notation)
	MetricsAllowedCIDRs []string `env:"METRICS_ALLOWED_CIDRS" envDefault:"10.0.0.0/8,172.16.0.0/12,192.168.0.0/16,127.0.0.0/8,::1/128" envSeparator:","`

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
		return nil, fmt.Errorf("load gateway config: %w", err)
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
	if c.Environment != "development" && c.JWTSecret == "your-secret-key-change-in-production" {
		return fmt.Errorf("JWT_SECRET must be changed from default value in %s environment", c.Environment)
	}
	if c.OTELSampleRate < 0 || c.OTELSampleRate > 1.0 {
		return fmt.Errorf("OTEL_SAMPLE_RATE must be between 0.0 and 1.0, got %f", c.OTELSampleRate)
	}
	return nil
}
