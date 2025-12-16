package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	apperrors "github.com/volchkovski/gophkeeper/internal/common/errors"
	"github.com/volchkovski/gophkeeper/internal/server/models"
)

// UserRepository implements repository.UserRepository for PostgreSQL.
type UserRepository struct {
	db *DB
}

// NewUserRepository creates a new UserRepository.
func NewUserRepository(db *DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create creates a new user record.
func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	query := `
		INSERT INTO users (id, username, password_hash, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := r.db.ExecContext(ctx, query,
		user.ID,
		user.Username,
		user.PasswordHash,
		user.CreatedAt,
		user.UpdatedAt,
	)
	if err != nil {
		// Check for unique constraint violation
		if isUniqueViolation(err) {
			return apperrors.ErrUserAlreadyExists
		}
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// FindByUsername retrieves a user by username.
func (r *UserRepository) FindByUsername(ctx context.Context, username string) (*models.User, error) {
	query := `
		SELECT id, username, password_hash, created_at, updated_at
		FROM users
		WHERE username = $1
	`

	var user models.User
	err := r.db.GetContext(ctx, &user, query, username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperrors.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to find user by username: %w", err)
	}

	return &user, nil
}

// FindByID retrieves a user by ID.
func (r *UserRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	query := `
		SELECT id, username, password_hash, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	var user models.User
	err := r.db.GetContext(ctx, &user, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperrors.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to find user by id: %w", err)
	}

	return &user, nil
}

// Update updates user information.
func (r *UserRepository) Update(ctx context.Context, user *models.User) error {
	query := `
		UPDATE users
		SET username = $2, password_hash = $3, updated_at = $4
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query,
		user.ID,
		user.Username,
		user.PasswordHash,
		user.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return apperrors.ErrUserAlreadyExists
		}
		return fmt.Errorf("failed to update user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return apperrors.ErrUserNotFound
	}

	return nil
}

// isUniqueViolation checks if error is a PostgreSQL unique constraint violation.
func isUniqueViolation(err error) bool {
	return err != nil && (contains(err.Error(), "unique constraint") ||
		contains(err.Error(), "duplicate key") ||
		contains(err.Error(), "23505"))
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

