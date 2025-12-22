package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"

	apperrors "github.com/volchkovski/gophkeeper/internal/common/errors"
	"github.com/volchkovski/gophkeeper/internal/server/models"
	"github.com/volchkovski/gophkeeper/internal/server/repository"
	"github.com/volchkovski/gophkeeper/pkg/logger"
)

const (
	// syncConcurrencyLimit limits the number of concurrent operations during sync.
	syncConcurrencyLimit = 10
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

// syncItem represents the result of processing a single client secret during sync.
type syncItem struct {
	updated  *models.SecretData
	conflict *Conflict
}

// Sync synchronizes secrets between client and server.
// Strategy: Last-Write-Wins based on version number.
// Uses errgroup with SetLimit for parallel processing of secrets.
func (s *secretService) Sync(ctx context.Context, userID uuid.UUID, clientSecrets []*models.SecretData) (*SyncResult, error) {
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

	// Mutex for protecting shared data during parallel processing
	var mu sync.Mutex
	processedIDs := make(map[uuid.UUID]struct{})
	var results []syncItem

	// Use errgroup with limit for parallel processing
	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(syncConcurrencyLimit)

	// Process client secrets in parallel
	for _, clientSecret := range clientSecrets {
		clientSecret := clientSecret // capture for goroutine
		clientSecret.UserID = userID

		g.Go(func() error {
			item, err := s.processClientSecret(gCtx, userID, clientSecret, serverSecretsMap)
			if err != nil {
				return err
			}

			mu.Lock()
			defer mu.Unlock()
			if item != nil {
				results = append(results, *item)
			}
			processedIDs[clientSecret.ID] = struct{}{}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	// Build result from collected items
	result := &SyncResult{
		UpdatedSecrets: make([]*models.SecretData, 0),
		Conflicts:      make([]*Conflict, 0),
	}

	for _, item := range results {
		if item.updated != nil {
			result.UpdatedSecrets = append(result.UpdatedSecrets, item.updated)
		}
		if item.conflict != nil {
			result.Conflicts = append(result.Conflicts, item.conflict)
		}
	}

	// Remaining server secrets are new to the client
	for id, serverSecret := range serverSecretsMap {
		if _, processed := processedIDs[id]; !processed {
			result.UpdatedSecrets = append(result.UpdatedSecrets, serverSecret)
		}
	}

	s.logger.Info("sync completed",
		logger.String("user_id", userID.String()),
		logger.Int("updated", len(result.UpdatedSecrets)),
		logger.Int("conflicts", len(result.Conflicts)),
	)

	return result, nil
}

// processClientSecret processes a single client secret during sync.
// Returns a syncItem with either an updated secret or a conflict, or nil if no action needed.
func (s *secretService) processClientSecret(
	ctx context.Context,
	userID uuid.UUID,
	clientSecret *models.SecretData,
	serverSecretsMap map[uuid.UUID]*models.SecretData,
) (*syncItem, error) {
	serverSecret, exists := serverSecretsMap[clientSecret.ID]

	if !exists {
		// New secret from client
		if err := s.secretRepo.Create(ctx, clientSecret); err != nil {
			if errors.Is(err, apperrors.ErrSecretNameExists) {
				// Name conflict - add to conflicts
				existingByName, _ := s.secretRepo.FindByUserIDAndName(ctx, userID, clientSecret.Name)
				if existingByName != nil {
					return &syncItem{
						conflict: &Conflict{
							ClientVersion: clientSecret,
							ServerVersion: existingByName,
						},
					}, nil
				}
				return nil, nil
			}
			return nil, fmt.Errorf("failed to create secret during sync: %w", err)
		}
		return nil, nil
	}

	// Secret exists on server - compare versions
	if clientSecret.Version > serverSecret.Version {
		// Client has newer version - update server
		clientSecret.IncrementVersion()
		if err := s.secretRepo.Update(ctx, clientSecret); err != nil {
			return nil, fmt.Errorf("failed to update secret during sync: %w", err)
		}
		return nil, nil
	} else if clientSecret.Version < serverSecret.Version {
		// Server has newer version - add to result
		return &syncItem{updated: serverSecret}, nil
	}

	// Same version - check timestamps for conflict
	if !clientSecret.UpdatedAt.Equal(serverSecret.UpdatedAt) {
		return &syncItem{
			conflict: &Conflict{
				ClientVersion: clientSecret,
				ServerVersion: serverSecret,
			},
		}, nil
	}

	return nil, nil
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

