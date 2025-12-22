// Package repository provides data access layer for GophKeeper server.
package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/volchkovski/gophkeeper/internal/server/models"
)

// UserRepository defines operations for user data access.
type UserRepository interface {
	// Create creates a new user record.
	Create(ctx context.Context, user *models.User) error

	// FindByUsername retrieves a user by username.
	FindByUsername(ctx context.Context, username string) (*models.User, error)

	// FindByID retrieves a user by ID.
	FindByID(ctx context.Context, id uuid.UUID) (*models.User, error)

	// Update updates user information.
	Update(ctx context.Context, user *models.User) error
}

// SecretRepository defines operations for secret data access.
type SecretRepository interface {
	// Create creates a new secret record.
	Create(ctx context.Context, secret *models.SecretData) error

	// Update updates an existing secret.
	Update(ctx context.Context, secret *models.SecretData) error

	// Delete performs soft delete on a secret.
	Delete(ctx context.Context, id uuid.UUID) error

	// FindByID retrieves a secret by ID.
	FindByID(ctx context.Context, id uuid.UUID) (*models.SecretData, error)

	// FindByUserID retrieves all secrets for a user (excluding deleted).
	FindByUserID(ctx context.Context, userID uuid.UUID) ([]*models.SecretData, error)

	// FindByUserIDIncludeDeleted retrieves all secrets for a user including deleted.
	FindByUserIDIncludeDeleted(ctx context.Context, userID uuid.UUID) ([]*models.SecretData, error)

	// FindByUserIDAndName retrieves a secret by user ID and name.
	FindByUserIDAndName(ctx context.Context, userID uuid.UUID, name string) (*models.SecretData, error)

	// FindByUserIDSinceVersion retrieves secrets updated since given version.
	FindByUserIDSinceVersion(ctx context.Context, userID uuid.UUID, version int64) ([]*models.SecretData, error)
}

