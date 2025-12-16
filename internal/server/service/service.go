// Package service provides business logic for GophKeeper server.
package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/volchkovski/gophkeeper/internal/server/models"
)

// AuthService handles user authentication and authorization.
// It provides methods for user registration, login, and token management.
type AuthService interface {
	// Register creates a new user account with the given username and password.
	// It returns an error if the username is already taken or if validation fails.
	//
	// Example:
	//   user, err := authService.Register(ctx, "john", "securepassword123")
	Register(ctx context.Context, username, password string) (*models.User, error)

	// Login authenticates a user and returns JWT tokens.
	// It returns an error if credentials are invalid.
	Login(ctx context.Context, username, password string) (*TokenPair, error)

	// RefreshToken generates new tokens using a valid refresh token.
	RefreshToken(ctx context.Context, refreshToken string) (*TokenPair, error)

	// ValidateToken validates an access token and returns the claims.
	ValidateToken(token string) (*Claims, error)
}

// SecretService handles CRUD operations for secrets.
type SecretService interface {
	// Create creates a new secret for the user.
	Create(ctx context.Context, userID uuid.UUID, secret *models.SecretData) error

	// Update updates an existing secret.
	Update(ctx context.Context, userID uuid.UUID, secret *models.SecretData) error

	// Delete performs soft delete on a secret.
	Delete(ctx context.Context, userID uuid.UUID, secretID uuid.UUID) error

	// Get retrieves a secret by ID.
	Get(ctx context.Context, userID uuid.UUID, secretID uuid.UUID) (*models.SecretData, error)

	// GetByName retrieves a secret by name.
	GetByName(ctx context.Context, userID uuid.UUID, name string) (*models.SecretData, error)

	// List retrieves all secrets for a user.
	List(ctx context.Context, userID uuid.UUID) ([]*models.SecretData, error)

	// Sync synchronizes secrets between client and server.
	Sync(ctx context.Context, userID uuid.UUID, clientSecrets []*models.SecretData) (*SyncResult, error)
}

// TokenPair contains access and refresh tokens.
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

// Claims represents JWT token claims.
type Claims struct {
	UserID   uuid.UUID `json:"user_id"`
	Username string    `json:"username"`
}

// SyncResult contains the result of a sync operation.
type SyncResult struct {
	// UpdatedSecrets are secrets that were updated on the server
	UpdatedSecrets []*models.SecretData `json:"updated_secrets"`
	// Conflicts are secrets that have conflicts between client and server
	Conflicts []*Conflict `json:"conflicts,omitempty"`
}

// Conflict represents a sync conflict between client and server versions.
type Conflict struct {
	ClientVersion *models.SecretData `json:"client_version"`
	ServerVersion *models.SecretData `json:"server_version"`
}

