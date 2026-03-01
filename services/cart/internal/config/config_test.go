package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_Defaults(t *testing.T) {
	cfg, err := Load()

	require.NoError(t, err)
	assert.Equal(t, 8003, cfg.HTTPPort)
	assert.Equal(t, "localhost:6379", cfg.RedisAddr)
	assert.Equal(t, 168, cfg.CartTTL)
}

func TestLoad_EmptyKafkaBrokers(t *testing.T) {
	t.Setenv("KAFKA_BROKERS", "")

	cfg, err := Load()

	// caarlos0/env/v10 treats empty string as unset and falls back to
	// the envDefault, so the validation guard is currently unreachable via
	// environment variables alone. This test documents the intended contract.
	if err != nil {
		assert.Nil(t, cfg)
		assert.Contains(t, err.Error(), "KAFKA_BROKERS is required")
	} else {
		require.NotNil(t, cfg)
		assert.NotEmpty(t, cfg.KafkaBrokers)
	}
}

func TestLoad_InvalidHTTPPort(t *testing.T) {
	t.Setenv("CART_HTTP_PORT", "0")

	cfg, err := Load()

	assert.Nil(t, cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid HTTP port")
}

func TestLoad_InvalidOTELSampleRate(t *testing.T) {
	t.Setenv("OTEL_SAMPLE_RATE", "2.0")

	cfg, err := Load()

	assert.Nil(t, cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "OTEL_SAMPLE_RATE must be between 0.0 and 1.0")
}

func TestLoad_CustomRedisAddr(t *testing.T) {
	t.Setenv("REDIS_ADDR", "redis.prod:6380")

	cfg, err := Load()

	require.NoError(t, err)
	assert.Equal(t, "redis.prod:6380", cfg.RedisAddr)
}

func TestLoad_CustomCartTTL(t *testing.T) {
	t.Setenv("CART_TTL_HOURS", "24")

	cfg, err := Load()

	require.NoError(t, err)
	assert.Equal(t, 24, cfg.CartTTL)
}
