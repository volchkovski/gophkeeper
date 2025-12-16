package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	apperrors "github.com/volchkovski/gophkeeper/internal/common/errors"
	"github.com/volchkovski/gophkeeper/internal/server/models"
	"github.com/volchkovski/gophkeeper/internal/server/repository"
	"github.com/volchkovski/gophkeeper/pkg/logger"
)

// secretService implements SecretService interface.
type secretService struct {
	secretRepo repository.SecretRepository
	logger     *logger.Logger
}

// NewSecretService creates a new SecretService.
func NewSecretService(
	secretRepo repository.SecretRepository,
	logger *logger.Logger,
) SecretService {
	return &secretService{
		secretRepo: secretRepo,
		logger:     logger,
	}
}

// Create creates a new secret for the user.
func (s *secretService) Create(ctx context.Context, userID uuid.UUID, secret *models.SecretData) error {
	// Validate secret
	if err := s.validateSecret(secret); err != nil {
		return err
	}

	// Ensure the secret belongs to the user
	secret.UserID = userID
	secret.ID = uuid.New()
	secret.Version = 1
	secret.CreatedAt = time.Now()
	secret.UpdatedAt = secret.CreatedAt

	if err := s.secretRepo.Create(ctx, secret); err != nil {
		return fmt.Errorf("failed to create secret: %w", err)
	}

	s.logger.Info("secret created",
		logger.String("user_id", userID.String()),
		logger.String("secret_id", secret.ID.String()),
		logger.String("name", secret.Name),
	)

	return nil
}

// Update updates an existing secret.
func (s *secretService) Update(ctx context.Context, userID uuid.UUID, secret *models.SecretData) error {
	// Validate secret
	if err := s.validateSecret(secret); err != nil {
		return err
	}

	// Verify ownership
	existing, err := s.secretRepo.FindByID(ctx, secret.ID)
	if err != nil {
		return err
	}

	if existing.UserID != userID {
		return apperrors.ErrSecretAccessDenied
	}

	if existing.IsDeleted() {
		return apperrors.ErrSecretNotFound
	}

	// Update fields
	existing.Type = secret.Type
	existing.Name = secret.Name
	existing.EncryptedData = secret.EncryptedData
	existing.Metadata = secret.Metadata
	existing.IncrementVersion()

	if err := s.secretRepo.Update(ctx, existing); err != nil {
		return fmt.Errorf("failed to update secret: %w", err)
	}

	s.logger.Info("secret updated",
		logger.String("user_id", userID.String()),
		logger.String("secret_id", secret.ID.String()),
	)

	return nil
}

// Delete performs soft delete on a secret.
func (s *secretService) Delete(ctx context.Context, userID uuid.UUID, secretID uuid.UUID) error {
	// Verify ownership
	existing, err := s.secretRepo.FindByID(ctx, secretID)
	if err != nil {
		return err
	}

	if existing.UserID != userID {
		return apperrors.ErrSecretAccessDenied
	}

	if existing.IsDeleted() {
		return apperrors.ErrSecretNotFound
	}

	if err := s.secretRepo.Delete(ctx, secretID); err != nil {
		return fmt.Errorf("failed to delete secret: %w", err)
	}

	s.logger.Info("secret deleted",
		logger.String("user_id", userID.String()),
		logger.String("secret_id", secretID.String()),
	)

	return nil
}

// Get retrieves a secret by ID.
func (s *secretService) Get(ctx context.Context, userID uuid.UUID, secretID uuid.UUID) (*models.SecretData, error) {
	secret, err := s.secretRepo.FindByID(ctx, secretID)
	if err != nil {
		return nil, err
	}

	if secret.UserID != userID {
		return nil, apperrors.ErrSecretAccessDenied
	}

	if secret.IsDeleted() {
		return nil, apperrors.ErrSecretNotFound
	}

	return secret, nil
}

// GetByName retrieves a secret by name.
func (s *secretService) GetByName(ctx context.Context, userID uuid.UUID, name string) (*models.SecretData, error) {
	return s.secretRepo.FindByUserIDAndName(ctx, userID, name)
}

// List retrieves all secrets for a user.
func (s *secretService) List(ctx context.Context, userID uuid.UUID) ([]*models.SecretData, error) {
	return s.secretRepo.FindByUserID(ctx, userID)
}

// Sync synchronizes secrets between client and server.
// Strategy: Last-Write-Wins based on version number.
func (s *secretService) Sync(ctx context.Context, userID uuid.UUID, clientSecrets []*models.SecretData) (*SyncResult, error) {
	result := &SyncResult{
		UpdatedSecrets: make([]*models.SecretData, 0),
		Conflicts:      make([]*Conflict, 0),
	}

	// Get all server secrets for the user
	serverSecrets, err := s.secretRepo.FindByUserIDIncludeDeleted(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get server secrets: %w", err)
	}

	// Create a map of server secrets by ID
	serverSecretsMap := make(map[uuid.UUID]*models.SecretData)
	for _, secret := range serverSecrets {
		serverSecretsMap[secret.ID] = secret
	}

	// Process client secrets
	for _, clientSecret := range clientSecrets {
		// Ensure the secret belongs to the user
		clientSecret.UserID = userID

		serverSecret, exists := serverSecretsMap[clientSecret.ID]

		if !exists {
			// New secret from client
			if err := s.secretRepo.Create(ctx, clientSecret); err != nil {
				if errors.Is(err, apperrors.ErrSecretNameExists) {
					// Name conflict - add to conflicts
					existingByName, _ := s.secretRepo.FindByUserIDAndName(ctx, userID, clientSecret.Name)
					if existingByName != nil {
						result.Conflicts = append(result.Conflicts, &Conflict{
							ClientVersion: clientSecret,
							ServerVersion: existingByName,
						})
					}
					continue
				}
				return nil, fmt.Errorf("failed to create secret during sync: %w", err)
			}
			continue
		}

		// Secret exists on server - compare versions
		if clientSecret.Version > serverSecret.Version {
			// Client has newer version - update server
			clientSecret.IncrementVersion()
			if err := s.secretRepo.Update(ctx, clientSecret); err != nil {
				return nil, fmt.Errorf("failed to update secret during sync: %w", err)
			}
		} else if clientSecret.Version < serverSecret.Version {
			// Server has newer version - add to result
			result.UpdatedSecrets = append(result.UpdatedSecrets, serverSecret)
		} else {
			// Same version - check timestamps for conflict
			if !clientSecret.UpdatedAt.Equal(serverSecret.UpdatedAt) {
				result.Conflicts = append(result.Conflicts, &Conflict{
					ClientVersion: clientSecret,
					ServerVersion: serverSecret,
				})
			}
		}

		// Mark as processed
		delete(serverSecretsMap, clientSecret.ID)
	}

	// Remaining server secrets are new to the client
	for _, serverSecret := range serverSecretsMap {
		result.UpdatedSecrets = append(result.UpdatedSecrets, serverSecret)
	}

	s.logger.Info("sync completed",
		logger.String("user_id", userID.String()),
		logger.Int("updated", len(result.UpdatedSecrets)),
		logger.Int("conflicts", len(result.Conflicts)),
	)

	return result, nil
}

// validateSecret validates secret data.
func (s *secretService) validateSecret(secret *models.SecretData) error {
	if secret.Name == "" {
		return apperrors.ErrInvalidSecretName
	}
	if !secret.Type.IsValid() {
		return apperrors.ErrInvalidSecretType
	}
	if len(secret.EncryptedData) == 0 {
		return apperrors.ErrInvalidInput
	}
	return nil
}

