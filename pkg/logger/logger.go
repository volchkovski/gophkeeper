// Package logger provides structured logging functionality for GophKeeper.
package logger

import (
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger is a wrapper around zap.Logger.
type Logger struct {
	*zap.Logger
}

// Config holds logger configuration.
type Config struct {
	Level  string
	Format string // "json" or "console"
}

// New creates a new Logger instance.
func New(cfg Config) (*Logger, error) {
	level, err := zapcore.ParseLevel(cfg.Level)
	if err != nil {
		level = zapcore.InfoLevel
	}

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	var encoder zapcore.Encoder
	if cfg.Format == "console" {
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	core := zapcore.NewCore(
		encoder,
		zapcore.AddSync(os.Stdout),
		level,
	)

	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	return &Logger{Logger: logger}, nil
}

// NewNop creates a no-op logger for testing.
func NewNop() *Logger {
	return &Logger{Logger: zap.NewNop()}
}

// With creates a child logger with additional fields.
func (l *Logger) With(fields ...zap.Field) *Logger {
	return &Logger{Logger: l.Logger.With(fields...)}
}

// Sync flushes any buffered log entries.
func (l *Logger) Sync() error {
	return l.Logger.Sync()
}

// Named adds a name to the logger.
func (l *Logger) Named(name string) *Logger {
	return &Logger{Logger: l.Logger.Named(name)}
}

// Field is a type alias for zap.Field.
type Field = zap.Field

// String creates a string field.
func String(key, value string) Field {
	return zap.String(key, value)
}

// Int creates an int field.
func Int(key string, value int) Field {
	return zap.Int(key, value)
}

// Int64 creates an int64 field.
func Int64(key string, value int64) Field {
	return zap.Int64(key, value)
}

// Error creates an error field.
func Error(err error) Field {
	return zap.Error(err)
}

// Duration creates a duration field.
func Duration(key string, value time.Duration) Field {
	return zap.Duration(key, value)
}

// Any creates a field with any value.
func Any(key string, value interface{}) Field {
	return zap.Any(key, value)
}

