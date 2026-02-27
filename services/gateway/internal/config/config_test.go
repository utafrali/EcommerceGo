package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidate_DevelopmentWithDefaultSecret_OK(t *testing.T) {
	cfg := &Config{
		HTTPPort:    8080,
		Environment: "development",
		JWTSecret:   "your-secret-key-change-in-production",
	}
	err := cfg.validate()
	assert.NoError(t, err, "development environment should accept default JWT secret")
}

func TestValidate_ProductionWithDefaultSecret_Error(t *testing.T) {
	cfg := &Config{
		HTTPPort:    8080,
		Environment: "production",
		JWTSecret:   "your-secret-key-change-in-production",
	}
	err := cfg.validate()
	assert.Error(t, err, "production environment should reject default JWT secret")
	assert.Contains(t, err.Error(), "JWT_SECRET must be changed")
	assert.Contains(t, err.Error(), "production")
}

func TestValidate_StagingWithDefaultSecret_Error(t *testing.T) {
	cfg := &Config{
		HTTPPort:    8080,
		Environment: "staging",
		JWTSecret:   "your-secret-key-change-in-production",
	}
	err := cfg.validate()
	assert.Error(t, err, "staging environment should reject default JWT secret")
	assert.Contains(t, err.Error(), "staging")
}

func TestValidate_ProductionWithCustomSecret_OK(t *testing.T) {
	cfg := &Config{
		HTTPPort:    8080,
		Environment: "production",
		JWTSecret:   "my-secure-production-secret-2026",
	}
	err := cfg.validate()
	assert.NoError(t, err, "production with custom secret should pass validation")
}

func TestValidate_InvalidPort_Error(t *testing.T) {
	cfg := &Config{
		HTTPPort:    0,
		Environment: "development",
		JWTSecret:   "your-secret-key-change-in-production",
	}
	err := cfg.validate()
	assert.Error(t, err, "port 0 should be rejected")
	assert.Contains(t, err.Error(), "invalid HTTP port")
}

func TestValidate_EmptyEnvironment_WithDefaultSecret_Error(t *testing.T) {
	cfg := &Config{
		HTTPPort:    8080,
		Environment: "",
		JWTSecret:   "your-secret-key-change-in-production",
	}
	err := cfg.validate()
	assert.Error(t, err, "non-development (empty) environment should reject default secret")
}
