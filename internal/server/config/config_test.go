package config

import (
	"os"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func resetViper() {
	viper.Reset()
}

func TestLoad_Defaults(t *testing.T) {
	resetViper()

	cfg, err := Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Check server defaults
	assert.Equal(t, "localhost", cfg.Server.Host)
	assert.Equal(t, 8080, cfg.Server.Port)
	assert.Equal(t, 30*time.Second, cfg.Server.ReadTimeout)
	assert.Equal(t, 30*time.Second, cfg.Server.WriteTimeout)

	// Check database defaults
	assert.Contains(t, cfg.Database.DSN, "postgres://")
	assert.Equal(t, 25, cfg.Database.MaxOpenConns)
	assert.Equal(t, 5, cfg.Database.MaxIdleConns)
	assert.Equal(t, 5*time.Minute, cfg.Database.ConnMaxLifetime)

	// Check JWT defaults
	assert.NotEmpty(t, cfg.JWT.Secret)
	assert.Equal(t, 1*time.Hour, cfg.JWT.AccessTokenTTL)
	assert.Equal(t, 30*24*time.Hour, cfg.JWT.RefreshTokenTTL)

	// Check security defaults
	assert.Equal(t, 12, cfg.Security.BcryptCost)
	assert.Equal(t, 100, cfg.Security.RateLimit)

	// Check TLS defaults
	assert.False(t, cfg.TLS.Enabled)
	assert.NotEmpty(t, cfg.TLS.CertFile)
	assert.NotEmpty(t, cfg.TLS.KeyFile)

	// Check log defaults
	assert.Equal(t, "info", cfg.Log.Level)
	assert.Equal(t, "json", cfg.Log.Format)
}

func TestLoad_WithEnvVariables(t *testing.T) {
	resetViper()

	// Set environment variables
	os.Setenv("SERVER_HOST", "0.0.0.0")
	os.Setenv("SERVER_PORT", "9090")
	os.Setenv("JWT_SECRET", "test-secret")
	os.Setenv("LOG_LEVEL", "debug")
	defer func() {
		os.Unsetenv("SERVER_HOST")
		os.Unsetenv("SERVER_PORT")
		os.Unsetenv("JWT_SECRET")
		os.Unsetenv("LOG_LEVEL")
	}()

	cfg, err := Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "0.0.0.0", cfg.Server.Host)
	assert.Equal(t, 9090, cfg.Server.Port)
	assert.Equal(t, "test-secret", cfg.JWT.Secret)
	assert.Equal(t, "debug", cfg.Log.Level)
}

func TestLoad_WithConfigFile(t *testing.T) {
	resetViper()

	// Create a temporary directory with valid config file
	tmpDir, err := os.MkdirTemp("", "config-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Change to temp directory
	originalDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalDir)

	// Create valid YAML config file
	configContent := `
server:
  host: "0.0.0.0"
  port: 9999
jwt:
  secret: "file-secret"
log:
  level: "warn"
`
	err = os.WriteFile("config.yaml", []byte(configContent), 0600)
	require.NoError(t, err)

	cfg, err := Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "0.0.0.0", cfg.Server.Host)
	assert.Equal(t, 9999, cfg.Server.Port)
	assert.Equal(t, "file-secret", cfg.JWT.Secret)
	assert.Equal(t, "warn", cfg.Log.Level)
}

func TestSetDefaults(t *testing.T) {
	resetViper()

	setDefaults()

	assert.Equal(t, "localhost", viper.GetString("server.host"))
	assert.Equal(t, 8080, viper.GetInt("server.port"))
	assert.Equal(t, 30*time.Second, viper.GetDuration("server.read_timeout"))
}

func TestBindEnvVariables(t *testing.T) {
	resetViper()
	setDefaults()

	// Set env var before binding
	os.Setenv("DATABASE_DSN", "postgres://custom:connection@localhost/test")
	defer os.Unsetenv("DATABASE_DSN")

	bindEnvVariables()

	// After binding, env var should override default
	assert.Equal(t, "postgres://custom:connection@localhost/test", viper.GetString("database.dsn"))
}

