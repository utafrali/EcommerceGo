package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testConfig struct {
	Port     int    `env:"TEST_CFG_PORT" envDefault:"8080"`
	Host     string `env:"TEST_CFG_HOST" envDefault:"localhost"`
	LogLevel string `env:"TEST_CFG_LOG_LEVEL" envDefault:"info"`
	Debug    bool   `env:"TEST_CFG_DEBUG" envDefault:"false"`
}

func TestLoad_Defaults(t *testing.T) {
	var cfg testConfig
	err := Load(&cfg)

	require.NoError(t, err)
	assert.Equal(t, 8080, cfg.Port)
	assert.Equal(t, "localhost", cfg.Host)
	assert.Equal(t, "info", cfg.LogLevel)
	assert.False(t, cfg.Debug)
}

func TestLoad_FromEnvVars(t *testing.T) {
	t.Setenv("TEST_CFG_PORT", "9090")
	t.Setenv("TEST_CFG_HOST", "0.0.0.0")
	t.Setenv("TEST_CFG_LOG_LEVEL", "debug")
	t.Setenv("TEST_CFG_DEBUG", "true")

	var cfg testConfig
	err := Load(&cfg)

	require.NoError(t, err)
	assert.Equal(t, 9090, cfg.Port)
	assert.Equal(t, "0.0.0.0", cfg.Host)
	assert.Equal(t, "debug", cfg.LogLevel)
	assert.True(t, cfg.Debug)
}

type requiredConfig struct {
	APIKey string `env:"TEST_CFG_API_KEY,required"`
}

func TestLoad_RequiredFieldMissing(t *testing.T) {
	var cfg requiredConfig
	err := Load(&cfg)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse config")
}

func TestLoad_RequiredFieldPresent(t *testing.T) {
	t.Setenv("TEST_CFG_API_KEY", "secret-123")

	var cfg requiredConfig
	err := Load(&cfg)

	require.NoError(t, err)
	assert.Equal(t, "secret-123", cfg.APIKey)
}

func TestLoad_InvalidType(t *testing.T) {
	t.Setenv("TEST_CFG_PORT", "not-a-number")

	var cfg testConfig
	err := Load(&cfg)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse config")
}
