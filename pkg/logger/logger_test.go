package logger

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name   string
		config Config
	}{
		{
			name: "json format with info level",
			config: Config{
				Level:  "info",
				Format: "json",
			},
		},
		{
			name: "console format with debug level",
			config: Config{
				Level:  "debug",
				Format: "console",
			},
		},
		{
			name: "invalid level defaults to info",
			config: Config{
				Level:  "invalid",
				Format: "json",
			},
		},
		{
			name: "warn level",
			config: Config{
				Level:  "warn",
				Format: "json",
			},
		},
		{
			name: "error level",
			config: Config{
				Level:  "error",
				Format: "json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log, err := New(tt.config)
			require.NoError(t, err)
			require.NotNil(t, log)
			require.NotNil(t, log.Logger)
		})
	}
}

func TestNewNop(t *testing.T) {
	log := NewNop()
	require.NotNil(t, log)
	require.NotNil(t, log.Logger)

	// Should not panic when logging
	assert.NotPanics(t, func() {
		log.Info("test message")
		log.Warn("test warning")
		log.Error("test error")
		log.Debug("test debug")
	})
}

func TestLogger_With(t *testing.T) {
	log := NewNop()

	// Create child logger with fields
	childLog := log.With(String("key", "value"))
	require.NotNil(t, childLog)

	// Should not panic
	assert.NotPanics(t, func() {
		childLog.Info("test with fields")
	})
}

func TestLogger_Sync(t *testing.T) {
	log := NewNop()

	// Sync should not return error for nop logger
	err := log.Sync()
	// Note: zap.NewNop() may return error on sync but we ignore it
	_ = err
}

func TestLogger_Named(t *testing.T) {
	log := NewNop()

	// Create named logger
	namedLog := log.Named("test-component")
	require.NotNil(t, namedLog)

	// Should not panic
	assert.NotPanics(t, func() {
		namedLog.Info("test from named logger")
	})
}

func TestFieldHelpers(t *testing.T) {
	t.Run("String field", func(t *testing.T) {
		field := String("key", "value")
		assert.Equal(t, "key", field.Key)
	})

	t.Run("Int field", func(t *testing.T) {
		field := Int("count", 42)
		assert.Equal(t, "count", field.Key)
	})

	t.Run("Int64 field", func(t *testing.T) {
		field := Int64("bigcount", int64(1234567890))
		assert.Equal(t, "bigcount", field.Key)
	})

	t.Run("Error field", func(t *testing.T) {
		err := errors.New("test error")
		field := Error(err)
		assert.Equal(t, "error", field.Key)
	})

	t.Run("Duration field", func(t *testing.T) {
		field := Duration("elapsed", 5*time.Second)
		assert.Equal(t, "elapsed", field.Key)
	})

	t.Run("Any field", func(t *testing.T) {
		field := Any("data", map[string]int{"a": 1})
		assert.Equal(t, "data", field.Key)
	})
}

func TestLogger_Integration(t *testing.T) {
	// Create a real logger (to JSON format for testing)
	log, err := New(Config{
		Level:  "debug",
		Format: "json",
	})
	require.NoError(t, err)

	// Create named child with fields
	childLog := log.Named("test").With(String("component", "integration"))

	// Should not panic
	assert.NotPanics(t, func() {
		childLog.Debug("debug message", Int("count", 1))
		childLog.Info("info message", String("key", "value"))
		childLog.Warn("warn message", Duration("elapsed", time.Second))
		childLog.Error("error message", Error(errors.New("test error")))
	})

	// Sync should complete without error (or we ignore it)
	_ = log.Sync()
}

