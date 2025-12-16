// Package config provides configuration management for the GophKeeper client.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

const (
	// ConfigDirName is the name of the configuration directory.
	ConfigDirName = ".gophkeeper"
	// ConfigFileName is the name of the configuration file.
	ConfigFileName = "config.json"
	// StorageFileName is the name of the local storage database.
	StorageFileName = "storage.db"
	// KeychainFileName is the name of the encrypted keychain file.
	KeychainFileName = "keychain"
)

// Config holds client configuration.
type Config struct {
	ServerURL   string        `json:"server_url"`
	Timeout     time.Duration `json:"timeout"`
	AutoSync    bool          `json:"auto_sync"`
	SyncOnStart bool          `json:"sync_on_start"`
}

// DefaultConfig returns the default client configuration.
func DefaultConfig() *Config {
	return &Config{
		ServerURL:   "http://localhost:8080",
		Timeout:     30 * time.Second,
		AutoSync:    true,
		SyncOnStart: true,
	}
}

// GetConfigDir returns the path to the configuration directory.
func GetConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ConfigDirName), nil
}

// GetConfigPath returns the path to the configuration file.
func GetConfigPath() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, ConfigFileName), nil
}

// GetStoragePath returns the path to the local storage database.
func GetStoragePath() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, StorageFileName), nil
}

// GetKeychainPath returns the path to the encrypted keychain file.
func GetKeychainPath() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, KeychainFileName), nil
}

// EnsureConfigDir creates the configuration directory if it doesn't exist.
func EnsureConfigDir() error {
	configDir, err := GetConfigDir()
	if err != nil {
		return err
	}
	return os.MkdirAll(configDir, 0700)
}

// Load reads configuration from file.
func Load() (*Config, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Save writes configuration to file.
func (c *Config) Save() error {
	if err := EnsureConfigDir(); err != nil {
		return err
	}

	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0600)
}

