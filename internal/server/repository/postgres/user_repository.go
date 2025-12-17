package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	apperrors "github.com/volchkovski/gophkeeper/internal/common/errors"
	"github.com/volchkovski/gophkeeper/internal/server/models"
)

const (
	usersTable   = "users"
	usersColumns = "id, username, password_hash, created_at, updated_at"
)

// UserRepository implements repository.UserRepository for PostgreSQL.
// It embeds GenericRepository for common CRUD operations.
type UserRepository struct {
	*GenericRepository[models.User, *models.User]
}

// NewUserRepository creates a new UserRepository.
func NewUserRepository(db *DB) *UserRepository {
	return &UserRepository{
		GenericRepository: NewGenericRepository[models.User, *models.User](db, usersTable, usersColumns),
	}
}

// Create creates a new user record.
func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	query := `
		INSERT INTO users (id, username, password_hash, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := r.ExecContext(ctx, query,
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
	query := fmt.Sprintf("SELECT %s FROM %s WHERE username = $1", usersColumns, usersTable)

	user, err := r.FindOneByQuery(ctx, query, username)
	if err != nil {
		return nil, fmt.Errorf("failed to find user by username: %w", err)
	}
	if user == nil {
		return nil, apperrors.ErrUserNotFound
	}

	return user, nil
}

// FindByID retrieves a user by ID.
func (r *UserRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	user, err := r.GenericRepository.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to find user by id: %w", err)
	}
	if user == nil {
		return nil, apperrors.ErrUserNotFound
	}

	return user, nil
}

// Update updates user information.
func (r *UserRepository) Update(ctx context.Context, user *models.User) error {
	query := `
		UPDATE users
		SET username = $2, password_hash = $3, updated_at = $4
		WHERE id = $1
	`

	result, err := r.ExecContext(ctx, query,
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

