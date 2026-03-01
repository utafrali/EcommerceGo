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

func TestLoad_Development_AcceptsDefaultSecret(t *testing.T) {
	// In development mode, the default JWT secret is accepted.
	setEnvs(t, map[string]string{
		"ENVIRONMENT": "development",
		"JWT_SECRET":  "change-this-to-a-secure-secret",
	})

	cfg, err := Load()

	require.NoError(t, err)
	assert.Equal(t, "development", cfg.Environment)
	assert.Equal(t, "change-this-to-a-secure-secret", cfg.JWTSecret)
}

func TestLoad_Production_RejectsDefaultSecret(t *testing.T) {
	setEnvs(t, map[string]string{
		"ENVIRONMENT": "production",
		"JWT_SECRET":  "change-this-to-a-secure-secret",
	})

	cfg, err := Load()

	assert.Nil(t, cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "JWT_SECRET must be explicitly set")
}

func TestLoad_Staging_RejectsDefaultSecret(t *testing.T) {
	setEnvs(t, map[string]string{
		"ENVIRONMENT": "staging",
		"JWT_SECRET":  "change-this-to-a-secure-secret",
	})

	cfg, err := Load()

	assert.Nil(t, cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "JWT_SECRET must be explicitly set")
}

func TestLoad_Production_RejectsShortSecret(t *testing.T) {
	setEnvs(t, map[string]string{
		"ENVIRONMENT": "production",
		"JWT_SECRET":  "short-but-not-default-secret",
	})

	cfg, err := Load()

	assert.Nil(t, cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "JWT_SECRET must be at least 32 characters")
}

func TestLoad_Production_AcceptsStrongSecret(t *testing.T) {
	// A secret that is 32+ chars and not the default sentinel value.
	strongSecret := "this-is-a-very-secure-secret-key-for-production-use-1234"
	setEnvs(t, map[string]string{
		"ENVIRONMENT": "production",
		"JWT_SECRET":  strongSecret,
	})

	cfg, err := Load()

	require.NoError(t, err)
	assert.Equal(t, "production", cfg.Environment)
	assert.Equal(t, strongSecret, cfg.JWTSecret)
}

func TestLoad_Production_RejectsExactly31CharSecret(t *testing.T) {
	// 31 characters -- just under the limit.
	secret := "abcdefghijklmnopqrstuvwxyz12345"
	assert.Equal(t, 31, len(secret), "test fixture must be exactly 31 chars")

	setEnvs(t, map[string]string{
		"ENVIRONMENT": "production",
		"JWT_SECRET":  secret,
	})

	cfg, err := Load()

	assert.Nil(t, cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "JWT_SECRET must be at least 32 characters")
}

func TestLoad_Production_AcceptsExactly32CharSecret(t *testing.T) {
	// Exactly 32 characters -- boundary case.
	secret := "abcdefghijklmnopqrstuvwxyz123456"
	assert.Equal(t, 32, len(secret), "test fixture must be exactly 32 chars")

	setEnvs(t, map[string]string{
		"ENVIRONMENT": "production",
		"JWT_SECRET":  secret,
	})

	cfg, err := Load()

	require.NoError(t, err)
	assert.Equal(t, secret, cfg.JWTSecret)
}

func TestLoad_DefaultPorts(t *testing.T) {
	setEnvs(t, map[string]string{
		"ENVIRONMENT": "development",
	})

	cfg, err := Load()

	require.NoError(t, err)
	assert.Equal(t, 8006, cfg.HTTPPort)
	assert.Equal(t, 9006, cfg.GRPCPort)
}
