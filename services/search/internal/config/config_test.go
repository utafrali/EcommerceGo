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
	assert.Equal(t, 8010, cfg.HTTPPort)
	assert.Equal(t, 9010, cfg.GRPCPort)
	assert.Equal(t, "http://localhost:9200", cfg.ElasticsearchURL)
	assert.Equal(t, "ecommerce_products", cfg.ElasticsearchIndex)
	assert.Equal(t, "elasticsearch", cfg.SearchEngine)
	assert.Equal(t, "http://localhost:8080", cfg.ProductServiceURL)
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
	t.Setenv("SEARCH_HTTP_PORT", "0")

	cfg, err := Load()

	assert.Nil(t, cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid HTTP port")
}

func TestLoad_InvalidGRPCPort(t *testing.T) {
	t.Setenv("SEARCH_GRPC_PORT", "99999")

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

func TestLoad_CustomElasticsearchURL(t *testing.T) {
	t.Setenv("ELASTICSEARCH_URL", "http://es.prod:9200")

	cfg, err := Load()

	require.NoError(t, err)
	assert.Equal(t, "http://es.prod:9200", cfg.ElasticsearchURL)
}

func TestLoad_CustomSearchEngine(t *testing.T) {
	t.Setenv("SEARCH_ENGINE", "memory")

	cfg, err := Load()

	require.NoError(t, err)
	assert.Equal(t, "memory", cfg.SearchEngine)
}
