// Package postgres provides PostgreSQL implementation of repositories.
package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Entity represents a database entity with an ID.
type Entity interface {
	GetID() uuid.UUID
}

// GenericRepository provides common CRUD operations for entities.
// It uses Go generics to reduce code duplication between repositories.
// The type parameter T must be a pointer type that implements Entity.
type GenericRepository[T any, PT interface {
	*T
	Entity
}] struct {
	db        *DB
	tableName string
	columns   string
}

// NewGenericRepository creates a new GenericRepository.
func NewGenericRepository[T any, PT interface {
	*T
	Entity
}](db *DB, tableName, columns string) *GenericRepository[T, PT] {
	return &GenericRepository[T, PT]{
		db:        db,
		tableName: tableName,
		columns:   columns,
	}
}

// FindByID retrieves an entity by ID using the generic type.
func (r *GenericRepository[T, PT]) FindByID(ctx context.Context, id uuid.UUID) (*T, error) {
	query := fmt.Sprintf("SELECT %s FROM %s WHERE id = $1", r.columns, r.tableName)

	var entity T
	err := r.db.GetContext(ctx, &entity, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find %s by id: %w", r.tableName, err)
	}

	return &entity, nil
}

// FindAll retrieves all entities.
func (r *GenericRepository[T, PT]) FindAll(ctx context.Context) ([]*T, error) {
	query := fmt.Sprintf("SELECT %s FROM %s", r.columns, r.tableName)

	var entities []*T
	err := r.db.SelectContext(ctx, &entities, query)
	if err != nil {
		return nil, fmt.Errorf("failed to find all %s: %w", r.tableName, err)
	}

	return entities, nil
}

// Exists checks if an entity with the given ID exists.
func (r *GenericRepository[T, PT]) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	query := fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s WHERE id = $1)", r.tableName)

	var exists bool
	err := r.db.GetContext(ctx, &exists, query, id)
	if err != nil {
		return false, fmt.Errorf("failed to check existence in %s: %w", r.tableName, err)
	}

	return exists, nil
}

// Count returns the total count of entities.
func (r *GenericRepository[T, PT]) Count(ctx context.Context) (int64, error) {
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", r.tableName)

	var count int64
	err := r.db.GetContext(ctx, &count, query)
	if err != nil {
		return 0, fmt.Errorf("failed to count %s: %w", r.tableName, err)
	}

	return count, nil
}

// DeleteByID deletes an entity by ID.
func (r *GenericRepository[T, PT]) DeleteByID(ctx context.Context, id uuid.UUID) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE id = $1", r.tableName)

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete from %s: %w", r.tableName, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// FindByQuery executes a custom query and returns matching entities.
func (r *GenericRepository[T, PT]) FindByQuery(ctx context.Context, query string, args ...interface{}) ([]*T, error) {
	var entities []*T
	err := r.db.SelectContext(ctx, &entities, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query on %s: %w", r.tableName, err)
	}

	return entities, nil
}

// FindOneByQuery executes a custom query and returns a single entity.
func (r *GenericRepository[T, PT]) FindOneByQuery(ctx context.Context, query string, args ...interface{}) (*T, error) {
	var entity T
	err := r.db.GetContext(ctx, &entity, query, args...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to execute query on %s: %w", r.tableName, err)
	}

	return &entity, nil
}

// ExecContext executes a query that doesn't return rows.
func (r *GenericRepository[T, PT]) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return r.db.ExecContext(ctx, query, args...)
}

// DB returns the underlying database connection for complex operations.
func (r *GenericRepository[T, PT]) DB() *sqlx.DB {
	return r.db.DB
}

