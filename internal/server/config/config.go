// Package config provides configuration management for the GophKeeper server.
package config

import (
	"time"

	"github.com/spf13/viper"
)

// Config holds all server configuration.
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	JWT      JWTConfig
	Security SecurityConfig
	TLS      TLSConfig
	Log      LogConfig
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Host         string        `mapstructure:"host"`
	Port         int           `mapstructure:"port"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
}

// DatabaseConfig holds database connection settings.
type DatabaseConfig struct {
	DSN             string        `mapstructure:"dsn"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}

// JWTConfig holds JWT token settings.
type JWTConfig struct {
	Secret          string        `mapstructure:"secret"`
	AccessTokenTTL  time.Duration `mapstructure:"access_token_ttl"`
	RefreshTokenTTL time.Duration `mapstructure:"refresh_token_ttl"`
}

// SecurityConfig holds security-related settings.
type SecurityConfig struct {
	BcryptCost int `mapstructure:"bcrypt_cost"`
	RateLimit  int `mapstructure:"rate_limit"`
}

// TLSConfig holds TLS settings.
type TLSConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	CertFile string `mapstructure:"cert_file"`
	KeyFile  string `mapstructure:"key_file"`
}

// LogConfig holds logging settings.
type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// Load reads configuration from environment variables and config file.
func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("/etc/gophkeeper")

	// Set default values
	setDefaults()

	// Read environment variables
	viper.AutomaticEnv()
	bindEnvVariables()

	// Try to read config file (optional)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// setDefaults sets default configuration values.
func setDefaults() {
	// Server defaults
	viper.SetDefault("server.host", "localhost")
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.read_timeout", 30*time.Second)
	viper.SetDefault("server.write_timeout", 30*time.Second)

	// Database defaults
	viper.SetDefault("database.dsn", "postgres://gophkeeper:secret@localhost:5432/gophkeeper?sslmode=disable")
	viper.SetDefault("database.max_open_conns", 25)
	viper.SetDefault("database.max_idle_conns", 5)
	viper.SetDefault("database.conn_max_lifetime", 5*time.Minute)

	// JWT defaults
	viper.SetDefault("jwt.secret", "change-me-in-production")
	viper.SetDefault("jwt.access_token_ttl", 1*time.Hour)
	viper.SetDefault("jwt.refresh_token_ttl", 30*24*time.Hour)

	// Security defaults
	viper.SetDefault("security.bcrypt_cost", 12)
	viper.SetDefault("security.rate_limit", 100)

	// TLS defaults
	viper.SetDefault("tls.enabled", false)
	viper.SetDefault("tls.cert_file", "./certs/server.crt")
	viper.SetDefault("tls.key_file", "./certs/server.key")

	// Log defaults
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.format", "json")
}

// bindEnvVariables binds environment variables to config keys.
func bindEnvVariables() {
	// Server
	_ = viper.BindEnv("server.host", "SERVER_HOST")
	_ = viper.BindEnv("server.port", "SERVER_PORT")
	_ = viper.BindEnv("server.read_timeout", "SERVER_READ_TIMEOUT")
	_ = viper.BindEnv("server.write_timeout", "SERVER_WRITE_TIMEOUT")

	// Database
	_ = viper.BindEnv("database.dsn", "DATABASE_DSN")
	_ = viper.BindEnv("database.max_open_conns", "DATABASE_MAX_OPEN_CONNS")
	_ = viper.BindEnv("database.max_idle_conns", "DATABASE_MAX_IDLE_CONNS")
	_ = viper.BindEnv("database.conn_max_lifetime", "DATABASE_CONN_MAX_LIFETIME")

	// JWT
	_ = viper.BindEnv("jwt.secret", "JWT_SECRET")
	_ = viper.BindEnv("jwt.access_token_ttl", "JWT_ACCESS_TOKEN_TTL")
	_ = viper.BindEnv("jwt.refresh_token_ttl", "JWT_REFRESH_TOKEN_TTL")

	// Security
	_ = viper.BindEnv("security.bcrypt_cost", "SECURITY_BCRYPT_COST")
	_ = viper.BindEnv("security.rate_limit", "SECURITY_RATE_LIMIT")

	// TLS
	_ = viper.BindEnv("tls.enabled", "TLS_ENABLED")
	_ = viper.BindEnv("tls.cert_file", "TLS_CERT_FILE")
	_ = viper.BindEnv("tls.key_file", "TLS_KEY_FILE")

	// Log
	_ = viper.BindEnv("log.level", "LOG_LEVEL")
	_ = viper.BindEnv("log.format", "LOG_FORMAT")
}
