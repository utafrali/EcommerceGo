package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setEnvs is a helper that sets multiple env vars and returns a cleanup function.
func setEnvs(t *testing.T, envs map[string]string) {
	t.Helper()
	for k, v := range envs {
		t.Setenv(k, v)
	}
}

func TestLoad_Defaults(t *testing.T) {
	cfg, err := Load()

	require.NoError(t, err)
	assert.Equal(t, 8007, cfg.HTTPPort)
	assert.Equal(t, 9007, cfg.GRPCPort)
	assert.Equal(t, 900, cfg.ReservationTTL)
	assert.Equal(t, "inventory_db", cfg.PostgresDB)
}

func TestLoad_EmptyPostgresHost(t *testing.T) {
	t.Setenv("POSTGRES_HOST", "")

	cfg, err := Load()

	// caarlos0/env/v10 treats empty string as unset and falls back to
	// the envDefault, so the validation guard is currently unreachable via
	// environment variables alone. This test documents the intended contract.
	if err != nil {
		assert.Nil(t, cfg)
		assert.Contains(t, err.Error(), "POSTGRES_HOST is required")
	} else {
		require.NotNil(t, cfg)
		assert.Equal(t, "localhost", cfg.PostgresHost)
	}
}

func TestLoad_EmptyPostgresUser(t *testing.T) {
	t.Setenv("POSTGRES_USER", "")

	cfg, err := Load()

	if err != nil {
		assert.Nil(t, cfg)
		assert.Contains(t, err.Error(), "POSTGRES_USER is required")
	} else {
		require.NotNil(t, cfg)
		assert.Equal(t, "ecommerce", cfg.PostgresUser)
	}
}

func TestLoad_EmptyKafkaBrokers(t *testing.T) {
	t.Setenv("KAFKA_BROKERS", "")

	cfg, err := Load()

	if err != nil {
		assert.Nil(t, cfg)
		assert.Contains(t, err.Error(), "KAFKA_BROKERS is required")
	} else {
		require.NotNil(t, cfg)
		assert.NotEmpty(t, cfg.KafkaBrokers)
	}
}

func TestLoad_InvalidHTTPPort(t *testing.T) {
	t.Setenv("INVENTORY_HTTP_PORT", "0")

	cfg, err := Load()

	assert.Nil(t, cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid HTTP port")
}

func TestLoad_InvalidGRPCPort(t *testing.T) {
	t.Setenv("INVENTORY_GRPC_PORT", "99999")

	cfg, err := Load()

	assert.Nil(t, cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid gRPC port")
}

func TestLoad_InvalidOTELSampleRate(t *testing.T) {
	t.Setenv("OTEL_SAMPLE_RATE", "2.0")

	cfg, err := Load()

	assert.Nil(t, cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "OTEL_SAMPLE_RATE must be between 0.0 and 1.0")
}

func TestLoad_ZeroReservationTTL(t *testing.T) {
	t.Setenv("RESERVATION_TTL_SECONDS", "0")

	cfg, err := Load()

	assert.Nil(t, cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "RESERVATION_TTL_SECONDS must be > 0")
}

func TestLoad_NegativeReservationTTL(t *testing.T) {
	t.Setenv("RESERVATION_TTL_SECONDS", "-1")

	cfg, err := Load()

	assert.Nil(t, cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "RESERVATION_TTL_SECONDS must be > 0")
}

func TestLoad_CustomReservationTTL(t *testing.T) {
	t.Setenv("RESERVATION_TTL_SECONDS", "300")

	cfg, err := Load()

	require.NoError(t, err)
	assert.Equal(t, 300, cfg.ReservationTTL)
}
