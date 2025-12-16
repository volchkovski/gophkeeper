package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	require.NotNil(t, cfg)
	assert.Equal(t, "http://localhost:8080", cfg.ServerURL)
	assert.Equal(t, 30*time.Second, cfg.Timeout)
	assert.True(t, cfg.AutoSync)
	assert.True(t, cfg.SyncOnStart)
}

func TestGetConfigDir(t *testing.T) {
	dir, err := GetConfigDir()

	require.NoError(t, err)
	assert.Contains(t, dir, ConfigDirName)
}

func TestGetConfigPath(t *testing.T) {
	path, err := GetConfigPath()

	require.NoError(t, err)
	assert.Contains(t, path, ConfigDirName)
	assert.Contains(t, path, ConfigFileName)
}

func TestGetStoragePath(t *testing.T) {
	path, err := GetStoragePath()

	require.NoError(t, err)
	assert.Contains(t, path, ConfigDirName)
	assert.Contains(t, path, StorageFileName)
}

func TestGetKeychainPath(t *testing.T) {
	path, err := GetKeychainPath()

	require.NoError(t, err)
	assert.Contains(t, path, ConfigDirName)
	assert.Contains(t, path, KeychainFileName)
}

func TestEnsureConfigDir(t *testing.T) {
	// Create a temporary home directory for testing
	tmpHome, err := os.MkdirTemp("", "gophkeeper-test-home")
	require.NoError(t, err)
	defer os.RemoveAll(tmpHome)

	// Save and restore HOME
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", originalHome)

	err = EnsureConfigDir()
	require.NoError(t, err)

	// Verify directory was created
	configDir := filepath.Join(tmpHome, ConfigDirName)
	info, err := os.Stat(configDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestLoad(t *testing.T) {
	t.Run("returns default config when file doesn't exist", func(t *testing.T) {
		// Create a temporary home directory
		tmpHome, err := os.MkdirTemp("", "gophkeeper-test-home")
		require.NoError(t, err)
		defer os.RemoveAll(tmpHome)

		// Save and restore HOME
		originalHome := os.Getenv("HOME")
		os.Setenv("HOME", tmpHome)
		defer os.Setenv("HOME", originalHome)

		cfg, err := Load()

		require.NoError(t, err)
		assert.Equal(t, DefaultConfig().ServerURL, cfg.ServerURL)
	})

	t.Run("loads config from file", func(t *testing.T) {
		// Create a temporary home directory
		tmpHome, err := os.MkdirTemp("", "gophkeeper-test-home")
		require.NoError(t, err)
		defer os.RemoveAll(tmpHome)

		// Save and restore HOME
		originalHome := os.Getenv("HOME")
		os.Setenv("HOME", tmpHome)
		defer os.Setenv("HOME", originalHome)

		// Create config directory and file
		configDir := filepath.Join(tmpHome, ConfigDirName)
		err = os.MkdirAll(configDir, 0700)
		require.NoError(t, err)

		customConfig := &Config{
			ServerURL:   "http://custom:9090",
			Timeout:     60 * time.Second,
			AutoSync:    false,
			SyncOnStart: false,
		}

		data, err := json.Marshal(customConfig)
		require.NoError(t, err)

		configPath := filepath.Join(configDir, ConfigFileName)
		err = os.WriteFile(configPath, data, 0600)
		require.NoError(t, err)

		cfg, err := Load()

		require.NoError(t, err)
		assert.Equal(t, "http://custom:9090", cfg.ServerURL)
		assert.Equal(t, 60*time.Second, cfg.Timeout)
		assert.False(t, cfg.AutoSync)
		assert.False(t, cfg.SyncOnStart)
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		// Create a temporary home directory
		tmpHome, err := os.MkdirTemp("", "gophkeeper-test-home")
		require.NoError(t, err)
		defer os.RemoveAll(tmpHome)

		// Save and restore HOME
		originalHome := os.Getenv("HOME")
		os.Setenv("HOME", tmpHome)
		defer os.Setenv("HOME", originalHome)

		// Create config directory and invalid file
		configDir := filepath.Join(tmpHome, ConfigDirName)
		err = os.MkdirAll(configDir, 0700)
		require.NoError(t, err)

		configPath := filepath.Join(configDir, ConfigFileName)
		err = os.WriteFile(configPath, []byte("invalid json"), 0600)
		require.NoError(t, err)

		cfg, err := Load()

		require.Error(t, err)
		assert.Nil(t, cfg)
	})
}

func TestConfig_Save(t *testing.T) {
	t.Run("saves config to file", func(t *testing.T) {
		// Create a temporary home directory
		tmpHome, err := os.MkdirTemp("", "gophkeeper-test-home")
		require.NoError(t, err)
		defer os.RemoveAll(tmpHome)

		// Save and restore HOME
		originalHome := os.Getenv("HOME")
		os.Setenv("HOME", tmpHome)
		defer os.Setenv("HOME", originalHome)

		cfg := &Config{
			ServerURL:   "http://test:8080",
			Timeout:     45 * time.Second,
			AutoSync:    true,
			SyncOnStart: false,
		}

		err = cfg.Save()
		require.NoError(t, err)

		// Verify file was created
		configPath := filepath.Join(tmpHome, ConfigDirName, ConfigFileName)
		data, err := os.ReadFile(configPath)
		require.NoError(t, err)

		var loadedCfg Config
		err = json.Unmarshal(data, &loadedCfg)
		require.NoError(t, err)

		assert.Equal(t, cfg.ServerURL, loadedCfg.ServerURL)
		assert.Equal(t, cfg.Timeout, loadedCfg.Timeout)
		assert.Equal(t, cfg.AutoSync, loadedCfg.AutoSync)
		assert.Equal(t, cfg.SyncOnStart, loadedCfg.SyncOnStart)
	})

	t.Run("creates config directory if not exists", func(t *testing.T) {
		// Create a temporary home directory
		tmpHome, err := os.MkdirTemp("", "gophkeeper-test-home")
		require.NoError(t, err)
		defer os.RemoveAll(tmpHome)

		// Save and restore HOME
		originalHome := os.Getenv("HOME")
		os.Setenv("HOME", tmpHome)
		defer os.Setenv("HOME", originalHome)

		cfg := DefaultConfig()
		err = cfg.Save()
		require.NoError(t, err)

		// Verify directory was created
		configDir := filepath.Join(tmpHome, ConfigDirName)
		info, err := os.Stat(configDir)
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})
}

func TestConfigConstants(t *testing.T) {
	assert.Equal(t, ".gophkeeper", ConfigDirName)
	assert.Equal(t, "config.json", ConfigFileName)
	assert.Equal(t, "storage.db", StorageFileName)
	assert.Equal(t, "keychain", KeychainFileName)
}

func TestConfig_Save_Error(t *testing.T) {
	// Create a temporary home directory
	tmpHome, err := os.MkdirTemp("", "gophkeeper-test-home-save-err")
	require.NoError(t, err)
	defer os.RemoveAll(tmpHome)

	// Save and restore HOME
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", originalHome)

	// Create a file where directory should be to cause MkdirAll error
	configDir := filepath.Join(tmpHome, ConfigDirName)
	err = os.WriteFile(configDir, []byte("not a directory"), 0600)
	require.NoError(t, err)

	cfg := DefaultConfig()
	err = cfg.Save()
	require.Error(t, err)
}

func TestLoad_ReadError(t *testing.T) {
	// Create a temporary home directory
	tmpHome, err := os.MkdirTemp("", "gophkeeper-test-home-read-err")
	require.NoError(t, err)
	defer os.RemoveAll(tmpHome)

	// Save and restore HOME
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", originalHome)

	// Create config directory
	configDir := filepath.Join(tmpHome, ConfigDirName)
	err = os.MkdirAll(configDir, 0700)
	require.NoError(t, err)

	// Create a directory where config file should be to cause read error
	configPath := filepath.Join(configDir, ConfigFileName)
	err = os.MkdirAll(configPath, 0700)
	require.NoError(t, err)

	cfg, err := Load()
	require.Error(t, err)
	assert.Nil(t, cfg)
}

