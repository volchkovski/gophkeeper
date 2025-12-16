package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	apperrors "github.com/volchkovski/gophkeeper/internal/common/errors"
	"github.com/volchkovski/gophkeeper/internal/server/models"
)

// SecretRepository implements repository.SecretRepository for PostgreSQL.
type SecretRepository struct {
	db *DB
}

// NewSecretRepository creates a new SecretRepository.
func NewSecretRepository(db *DB) *SecretRepository {
	return &SecretRepository{db: db}
}

// Create creates a new secret record.
func (r *SecretRepository) Create(ctx context.Context, secret *models.SecretData) error {
	query := `
		INSERT INTO secrets (id, user_id, type, name, encrypted_data, metadata, version, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := r.db.ExecContext(ctx, query,
		secret.ID,
		secret.UserID,
		secret.Type,
		secret.Name,
		secret.EncryptedData,
		secret.Metadata,
		secret.Version,
		secret.CreatedAt,
		secret.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return apperrors.ErrSecretNameExists
		}
		return fmt.Errorf("failed to create secret: %w", err)
	}

	return nil
}

// Update updates an existing secret.
func (r *SecretRepository) Update(ctx context.Context, secret *models.SecretData) error {
	query := `
		UPDATE secrets
		SET type = $2, name = $3, encrypted_data = $4, metadata = $5, version = $6, updated_at = $7
		WHERE id = $1 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query,
		secret.ID,
		secret.Type,
		secret.Name,
		secret.EncryptedData,
		secret.Metadata,
		secret.Version,
		secret.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return apperrors.ErrSecretNameExists
		}
		return fmt.Errorf("failed to update secret: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return apperrors.ErrSecretNotFound
	}

	return nil
}

// Delete performs soft delete on a secret.
func (r *SecretRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE secrets
		SET deleted_at = $2
		WHERE id = $1 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, id, time.Now())
	if err != nil {
		return fmt.Errorf("failed to delete secret: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return apperrors.ErrSecretNotFound
	}

	return nil
}

// FindByID retrieves a secret by ID.
func (r *SecretRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.SecretData, error) {
	query := `
		SELECT id, user_id, type, name, encrypted_data, metadata, version, created_at, updated_at, deleted_at
		FROM secrets
		WHERE id = $1
	`

	var secret models.SecretData
	err := r.db.GetContext(ctx, &secret, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperrors.ErrSecretNotFound
		}
		return nil, fmt.Errorf("failed to find secret by id: %w", err)
	}

	return &secret, nil
}

// FindByUserID retrieves all secrets for a user (excluding deleted).
func (r *SecretRepository) FindByUserID(ctx context.Context, userID uuid.UUID) ([]*models.SecretData, error) {
	query := `
		SELECT id, user_id, type, name, encrypted_data, metadata, version, created_at, updated_at, deleted_at
		FROM secrets
		WHERE user_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
	`

	var secrets []*models.SecretData
	err := r.db.SelectContext(ctx, &secrets, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to find secrets by user id: %w", err)
	}

	return secrets, nil
}

// FindByUserIDIncludeDeleted retrieves all secrets for a user including deleted.
func (r *SecretRepository) FindByUserIDIncludeDeleted(ctx context.Context, userID uuid.UUID) ([]*models.SecretData, error) {
	query := `
		SELECT id, user_id, type, name, encrypted_data, metadata, version, created_at, updated_at, deleted_at
		FROM secrets
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	var secrets []*models.SecretData
	err := r.db.SelectContext(ctx, &secrets, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to find secrets by user id: %w", err)
	}

	return secrets, nil
}

// FindByUserIDAndName retrieves a secret by user ID and name.
func (r *SecretRepository) FindByUserIDAndName(ctx context.Context, userID uuid.UUID, name string) (*models.SecretData, error) {
	query := `
		SELECT id, user_id, type, name, encrypted_data, metadata, version, created_at, updated_at, deleted_at
		FROM secrets
		WHERE user_id = $1 AND name = $2 AND deleted_at IS NULL
	`

	var secret models.SecretData
	err := r.db.GetContext(ctx, &secret, query, userID, name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperrors.ErrSecretNotFound
		}
		return nil, fmt.Errorf("failed to find secret by user id and name: %w", err)
	}

	return &secret, nil
}

// FindByUserIDSinceVersion retrieves secrets updated since given version.
func (r *SecretRepository) FindByUserIDSinceVersion(ctx context.Context, userID uuid.UUID, version int64) ([]*models.SecretData, error) {
	query := `
		SELECT id, user_id, type, name, encrypted_data, metadata, version, created_at, updated_at, deleted_at
		FROM secrets
		WHERE user_id = $1 AND version > $2
		ORDER BY version ASC
	`

	var secrets []*models.SecretData
	err := r.db.SelectContext(ctx, &secrets, query, userID, version)
	if err != nil {
		return nil, fmt.Errorf("failed to find secrets since version: %w", err)
	}

	return secrets, nil
}

