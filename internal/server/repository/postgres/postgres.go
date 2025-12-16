// Package postgres provides PostgreSQL implementation of repositories.
package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // PostgreSQL driver
)

// DB wraps sqlx.DB for PostgreSQL operations.
type DB struct {
	*sqlx.DB
}

// Config holds database configuration.
type Config struct {
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// NewDB creates a new PostgreSQL database connection.
func NewDB(cfg Config) (*DB, error) {
	db, err := sqlx.Connect("postgres", cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	return &DB{DB: db}, nil
}

// Ping checks database connectivity.
func (db *DB) Ping(ctx context.Context) error {
	return db.PingContext(ctx)
}

// Close closes the database connection.
func (db *DB) Close() error {
	return db.DB.Close()
}

