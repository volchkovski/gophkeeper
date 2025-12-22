package service

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	apperrors "github.com/volchkovski/gophkeeper/internal/common/errors"
	"github.com/volchkovski/gophkeeper/internal/server/models"
	"github.com/volchkovski/gophkeeper/pkg/logger"
)

// MockSecretRepository is a mock implementation of SecretRepository.
type MockSecretRepository struct {
	mock.Mock
}

func (m *MockSecretRepository) Create(ctx context.Context, secret *models.SecretData) error {
	args := m.Called(ctx, secret)
	return args.Error(0)
}

func (m *MockSecretRepository) Update(ctx context.Context, secret *models.SecretData) error {
	args := m.Called(ctx, secret)
	return args.Error(0)
}

func (m *MockSecretRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockSecretRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.SecretData, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.SecretData), args.Error(1)
}

func (m *MockSecretRepository) FindByUserID(ctx context.Context, userID uuid.UUID) ([]*models.SecretData, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.SecretData), args.Error(1)
}

func (m *MockSecretRepository) FindByUserIDIncludeDeleted(ctx context.Context, userID uuid.UUID) ([]*models.SecretData, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.SecretData), args.Error(1)
}

func (m *MockSecretRepository) FindByUserIDAndName(ctx context.Context, userID uuid.UUID, name string) (*models.SecretData, error) {
	args := m.Called(ctx, userID, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.SecretData), args.Error(1)
}

func (m *MockSecretRepository) FindByUserIDSinceVersion(ctx context.Context, userID uuid.UUID, version int64) ([]*models.SecretData, error) {
	args := m.Called(ctx, userID, version)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.SecretData), args.Error(1)
}

func newTestSecretService(repo *MockSecretRepository) SecretService {
	log := logger.NewNop()
	return NewSecretService(repo, log)
}

func TestSecretService_Create(t *testing.T) {
	userID := uuid.New()

	tests := []struct {
		name    string
		secret  *models.SecretData
		setup   func(*MockSecretRepository)
		wantErr error
	}{
		{
			name: "successful creation",
			secret: &models.SecretData{
				Type:          models.TypeLoginPassword,
				Name:          "test-secret",
				EncryptedData: []byte("encrypted-data"),
			},
			setup: func(m *MockSecretRepository) {
				m.On("Create", mock.Anything, mock.AnythingOfType("*models.SecretData")).Return(nil)
			},
			wantErr: nil,
		},
		{
			name: "empty name",
			secret: &models.SecretData{
				Type:          models.TypeLoginPassword,
				Name:          "",
				EncryptedData: []byte("encrypted-data"),
			},
			setup:   func(m *MockSecretRepository) {},
			wantErr: apperrors.ErrInvalidSecretName,
		},
		{
			name: "invalid type",
			secret: &models.SecretData{
				Type:          "invalid_type",
				Name:          "test-secret",
				EncryptedData: []byte("encrypted-data"),
			},
			setup:   func(m *MockSecretRepository) {},
			wantErr: apperrors.ErrInvalidSecretType,
		},
		{
			name: "empty data",
			secret: &models.SecretData{
				Type:          models.TypeLoginPassword,
				Name:          "test-secret",
				EncryptedData: []byte{},
			},
			setup:   func(m *MockSecretRepository) {},
			wantErr: apperrors.ErrInvalidInput,
		},
		{
			name: "duplicate name",
			secret: &models.SecretData{
				Type:          models.TypeLoginPassword,
				Name:          "existing-secret",
				EncryptedData: []byte("encrypted-data"),
			},
			setup: func(m *MockSecretRepository) {
				m.On("Create", mock.Anything, mock.AnythingOfType("*models.SecretData")).Return(apperrors.ErrSecretNameExists)
			},
			wantErr: apperrors.ErrSecretNameExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockSecretRepository)
			tt.setup(mockRepo)

			service := newTestSecretService(mockRepo)
			ctx := context.Background()

			err := service.Create(ctx, userID, tt.secret)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, userID, tt.secret.UserID)
				assert.NotEqual(t, uuid.Nil, tt.secret.ID)
				assert.Equal(t, int64(1), tt.secret.Version)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestSecretService_Get(t *testing.T) {
	userID := uuid.New()
	secretID := uuid.New()
	otherUserID := uuid.New()

	tests := []struct {
		name     string
		userID   uuid.UUID
		secretID uuid.UUID
		setup    func(*MockSecretRepository)
		wantErr  error
	}{
		{
			name:     "successful get",
			userID:   userID,
			secretID: secretID,
			setup: func(m *MockSecretRepository) {
				secret := &models.SecretData{
					ID:            secretID,
					UserID:        userID,
					Type:          models.TypeLoginPassword,
					Name:          "test-secret",
					EncryptedData: []byte("encrypted-data"),
				}
				m.On("FindByID", mock.Anything, secretID).Return(secret, nil)
			},
			wantErr: nil,
		},
		{
			name:     "secret not found",
			userID:   userID,
			secretID: uuid.New(),
			setup: func(m *MockSecretRepository) {
				m.On("FindByID", mock.Anything, mock.AnythingOfType("uuid.UUID")).Return(nil, apperrors.ErrSecretNotFound)
			},
			wantErr: apperrors.ErrSecretNotFound,
		},
		{
			name:     "access denied - wrong user",
			userID:   otherUserID,
			secretID: secretID,
			setup: func(m *MockSecretRepository) {
				secret := &models.SecretData{
					ID:            secretID,
					UserID:        userID, // Different user
					Type:          models.TypeLoginPassword,
					Name:          "test-secret",
					EncryptedData: []byte("encrypted-data"),
				}
				m.On("FindByID", mock.Anything, secretID).Return(secret, nil)
			},
			wantErr: apperrors.ErrSecretAccessDenied,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockSecretRepository)
			tt.setup(mockRepo)

			service := newTestSecretService(mockRepo)
			ctx := context.Background()

			secret, err := service.Get(ctx, tt.userID, tt.secretID)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, secret)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, secret)
				assert.Equal(t, tt.secretID, secret.ID)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestSecretService_Update(t *testing.T) {
	userID := uuid.New()
	secretID := uuid.New()

	tests := []struct {
		name    string
		secret  *models.SecretData
		setup   func(*MockSecretRepository)
		wantErr error
	}{
		{
			name: "successful update",
			secret: &models.SecretData{
				ID:            secretID,
				Type:          models.TypeLoginPassword,
				Name:          "updated-secret",
				EncryptedData: []byte("new-encrypted-data"),
			},
			setup: func(m *MockSecretRepository) {
				existingSecret := &models.SecretData{
					ID:            secretID,
					UserID:        userID,
					Type:          models.TypeLoginPassword,
					Name:          "test-secret",
					EncryptedData: []byte("encrypted-data"),
					Version:       1,
				}
				m.On("FindByID", mock.Anything, secretID).Return(existingSecret, nil)
				m.On("Update", mock.Anything, mock.AnythingOfType("*models.SecretData")).Return(nil)
			},
			wantErr: nil,
		},
		{
			name: "secret not found",
			secret: &models.SecretData{
				ID:            uuid.New(),
				Type:          models.TypeLoginPassword,
				Name:          "updated-secret",
				EncryptedData: []byte("new-encrypted-data"),
			},
			setup: func(m *MockSecretRepository) {
				m.On("FindByID", mock.Anything, mock.AnythingOfType("uuid.UUID")).Return(nil, apperrors.ErrSecretNotFound)
			},
			wantErr: apperrors.ErrSecretNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockSecretRepository)
			tt.setup(mockRepo)

			service := newTestSecretService(mockRepo)
			ctx := context.Background()

			err := service.Update(ctx, userID, tt.secret)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestSecretService_Delete(t *testing.T) {
	userID := uuid.New()
	secretID := uuid.New()

	tests := []struct {
		name     string
		userID   uuid.UUID
		secretID uuid.UUID
		setup    func(*MockSecretRepository)
		wantErr  error
	}{
		{
			name:     "successful delete",
			userID:   userID,
			secretID: secretID,
			setup: func(m *MockSecretRepository) {
				existingSecret := &models.SecretData{
					ID:            secretID,
					UserID:        userID,
					Type:          models.TypeLoginPassword,
					Name:          "test-secret",
					EncryptedData: []byte("encrypted-data"),
				}
				m.On("FindByID", mock.Anything, secretID).Return(existingSecret, nil)
				m.On("Delete", mock.Anything, secretID).Return(nil)
			},
			wantErr: nil,
		},
		{
			name:     "secret not found",
			userID:   userID,
			secretID: uuid.New(),
			setup: func(m *MockSecretRepository) {
				m.On("FindByID", mock.Anything, mock.AnythingOfType("uuid.UUID")).Return(nil, apperrors.ErrSecretNotFound)
			},
			wantErr: apperrors.ErrSecretNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockSecretRepository)
			tt.setup(mockRepo)

			service := newTestSecretService(mockRepo)
			ctx := context.Background()

			err := service.Delete(ctx, tt.userID, tt.secretID)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestSecretService_List(t *testing.T) {
	userID := uuid.New()

	tests := []struct {
		name     string
		setup    func(*MockSecretRepository)
		wantLen  int
		wantErr  bool
	}{
		{
			name: "list with secrets",
			setup: func(m *MockSecretRepository) {
				secrets := []*models.SecretData{
					{
						ID:            uuid.New(),
						UserID:        userID,
						Type:          models.TypeLoginPassword,
						Name:          "secret-1",
						EncryptedData: []byte("data-1"),
					},
					{
						ID:            uuid.New(),
						UserID:        userID,
						Type:          models.TypeText,
						Name:          "secret-2",
						EncryptedData: []byte("data-2"),
					},
				}
				m.On("FindByUserID", mock.Anything, userID).Return(secrets, nil)
			},
			wantLen: 2,
			wantErr: false,
		},
		{
			name: "empty list",
			setup: func(m *MockSecretRepository) {
				m.On("FindByUserID", mock.Anything, userID).Return([]*models.SecretData{}, nil)
			},
			wantLen: 0,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockSecretRepository)
			tt.setup(mockRepo)

			service := newTestSecretService(mockRepo)
			ctx := context.Background()

			secrets, err := service.List(ctx, userID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, secrets, tt.wantLen)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestSecretService_Sync(t *testing.T) {
	userID := uuid.New()
	now := time.Now()

	t.Run("sync with new client secrets", func(t *testing.T) {
		mockRepo := new(MockSecretRepository)

		// Server has no secrets
		mockRepo.On("FindByUserIDIncludeDeleted", mock.Anything, userID).Return([]*models.SecretData{}, nil)

		// Client has a new secret
		clientSecret := &models.SecretData{
			ID:            uuid.New(),
			Type:          models.TypeLoginPassword,
			Name:          "new-secret",
			EncryptedData: []byte("data"),
			Version:       1,
			CreatedAt:     now,
			UpdatedAt:     now,
		}

		mockRepo.On("Create", mock.Anything, clientSecret).Return(nil)

		service := newTestSecretService(mockRepo)
		ctx := context.Background()

		result, err := service.Sync(ctx, userID, []*models.SecretData{clientSecret})

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result.UpdatedSecrets, 0)
		assert.Len(t, result.Conflicts, 0)

		mockRepo.AssertExpectations(t)
	})

	t.Run("sync with server having newer version", func(t *testing.T) {
		mockRepo := new(MockSecretRepository)
		secretID := uuid.New()

		// Server has a newer version
		serverSecret := &models.SecretData{
			ID:            secretID,
			UserID:        userID,
			Type:          models.TypeLoginPassword,
			Name:          "secret",
			EncryptedData: []byte("server-data"),
			Version:       3,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		mockRepo.On("FindByUserIDIncludeDeleted", mock.Anything, userID).Return([]*models.SecretData{serverSecret}, nil)

		// Client has an older version
		clientSecret := &models.SecretData{
			ID:            secretID,
			Type:          models.TypeLoginPassword,
			Name:          "secret",
			EncryptedData: []byte("client-data"),
			Version:       1,
			CreatedAt:     now,
			UpdatedAt:     now,
		}

		service := newTestSecretService(mockRepo)
		ctx := context.Background()

		result, err := service.Sync(ctx, userID, []*models.SecretData{clientSecret})

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result.UpdatedSecrets, 1)
		assert.Equal(t, serverSecret.Version, result.UpdatedSecrets[0].Version)

		mockRepo.AssertExpectations(t)
	})

	t.Run("sync with client having newer version", func(t *testing.T) {
		mockRepo := new(MockSecretRepository)
		secretID := uuid.New()

		// Server has an older version
		serverSecret := &models.SecretData{
			ID:            secretID,
			UserID:        userID,
			Type:          models.TypeLoginPassword,
			Name:          "secret",
			EncryptedData: []byte("server-data"),
			Version:       1,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		mockRepo.On("FindByUserIDIncludeDeleted", mock.Anything, userID).Return([]*models.SecretData{serverSecret}, nil)

		// Client has a newer version
		clientSecret := &models.SecretData{
			ID:            secretID,
			Type:          models.TypeLoginPassword,
			Name:          "secret",
			EncryptedData: []byte("client-data"),
			Version:       3,
			CreatedAt:     now,
			UpdatedAt:     now,
		}

		mockRepo.On("Update", mock.Anything, clientSecret).Return(nil)

		service := newTestSecretService(mockRepo)
		ctx := context.Background()

		result, err := service.Sync(ctx, userID, []*models.SecretData{clientSecret})

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result.UpdatedSecrets, 0)

		mockRepo.AssertExpectations(t)
	})

	t.Run("sync with same version different timestamp - conflict", func(t *testing.T) {
		mockRepo := new(MockSecretRepository)
		secretID := uuid.New()

		// Server has same version but different timestamp
		serverSecret := &models.SecretData{
			ID:            secretID,
			UserID:        userID,
			Type:          models.TypeLoginPassword,
			Name:          "secret",
			EncryptedData: []byte("server-data"),
			Version:       2,
			CreatedAt:     now,
			UpdatedAt:     now.Add(time.Hour),
		}
		mockRepo.On("FindByUserIDIncludeDeleted", mock.Anything, userID).Return([]*models.SecretData{serverSecret}, nil)

		// Client has same version but different timestamp
		clientSecret := &models.SecretData{
			ID:            secretID,
			Type:          models.TypeLoginPassword,
			Name:          "secret",
			EncryptedData: []byte("client-data"),
			Version:       2,
			CreatedAt:     now,
			UpdatedAt:     now,
		}

		service := newTestSecretService(mockRepo)
		ctx := context.Background()

		result, err := service.Sync(ctx, userID, []*models.SecretData{clientSecret})

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result.Conflicts, 1)

		mockRepo.AssertExpectations(t)
	})

	t.Run("sync with name conflict on new secret", func(t *testing.T) {
		mockRepo := new(MockSecretRepository)

		existingSecret := &models.SecretData{
			ID:            uuid.New(),
			UserID:        userID,
			Type:          models.TypeLoginPassword,
			Name:          "existing-name",
			EncryptedData: []byte("existing-data"),
			Version:       1,
		}

		// Server has no secrets with same ID
		mockRepo.On("FindByUserIDIncludeDeleted", mock.Anything, userID).Return([]*models.SecretData{}, nil)

		// Client has a new secret
		clientSecret := &models.SecretData{
			ID:            uuid.New(),
			Type:          models.TypeLoginPassword,
			Name:          "existing-name",
			EncryptedData: []byte("data"),
			Version:       1,
			CreatedAt:     now,
			UpdatedAt:     now,
		}

		mockRepo.On("Create", mock.Anything, clientSecret).Return(apperrors.ErrSecretNameExists)
		mockRepo.On("FindByUserIDAndName", mock.Anything, userID, "existing-name").Return(existingSecret, nil)

		service := newTestSecretService(mockRepo)
		ctx := context.Background()

		result, err := service.Sync(ctx, userID, []*models.SecretData{clientSecret})

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result.Conflicts, 1)

		mockRepo.AssertExpectations(t)
	})

	t.Run("sync DB error on FindByUserIDIncludeDeleted", func(t *testing.T) {
		mockRepo := new(MockSecretRepository)
		mockRepo.On("FindByUserIDIncludeDeleted", mock.Anything, userID).Return(nil, assert.AnError)

		service := newTestSecretService(mockRepo)
		ctx := context.Background()

		result, err := service.Sync(ctx, userID, []*models.SecretData{})

		assert.Error(t, err)
		assert.Nil(t, result)

		mockRepo.AssertExpectations(t)
	})

	t.Run("sync with server-only secrets", func(t *testing.T) {
		mockRepo := new(MockSecretRepository)

		// Server has a secret that client doesn't have
		serverSecret := &models.SecretData{
			ID:            uuid.New(),
			UserID:        userID,
			Type:          models.TypeLoginPassword,
			Name:          "server-only-secret",
			EncryptedData: []byte("server-data"),
			Version:       1,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		mockRepo.On("FindByUserIDIncludeDeleted", mock.Anything, userID).Return([]*models.SecretData{serverSecret}, nil)

		service := newTestSecretService(mockRepo)
		ctx := context.Background()

		// Client sends empty list
		result, err := service.Sync(ctx, userID, []*models.SecretData{})

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result.UpdatedSecrets, 1)
		assert.Equal(t, serverSecret.ID, result.UpdatedSecrets[0].ID)

		mockRepo.AssertExpectations(t)
	})

	t.Run("sync update error", func(t *testing.T) {
		mockRepo := new(MockSecretRepository)
		secretID := uuid.New()

		serverSecret := &models.SecretData{
			ID:            secretID,
			UserID:        userID,
			Type:          models.TypeLoginPassword,
			Name:          "secret",
			EncryptedData: []byte("server-data"),
			Version:       1,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		mockRepo.On("FindByUserIDIncludeDeleted", mock.Anything, userID).Return([]*models.SecretData{serverSecret}, nil)

		clientSecret := &models.SecretData{
			ID:            secretID,
			Type:          models.TypeLoginPassword,
			Name:          "secret",
			EncryptedData: []byte("client-data"),
			Version:       3,
			CreatedAt:     now,
			UpdatedAt:     now,
		}

		mockRepo.On("Update", mock.Anything, clientSecret).Return(assert.AnError)

		service := newTestSecretService(mockRepo)
		ctx := context.Background()

		result, err := service.Sync(ctx, userID, []*models.SecretData{clientSecret})

		assert.Error(t, err)
		assert.Nil(t, result)

		mockRepo.AssertExpectations(t)
	})

	t.Run("sync create error", func(t *testing.T) {
		mockRepo := new(MockSecretRepository)
		mockRepo.On("FindByUserIDIncludeDeleted", mock.Anything, userID).Return([]*models.SecretData{}, nil)

		clientSecret := &models.SecretData{
			ID:            uuid.New(),
			Type:          models.TypeLoginPassword,
			Name:          "new-secret",
			EncryptedData: []byte("data"),
			Version:       1,
			CreatedAt:     now,
			UpdatedAt:     now,
		}

		mockRepo.On("Create", mock.Anything, clientSecret).Return(assert.AnError)

		service := newTestSecretService(mockRepo)
		ctx := context.Background()

		result, err := service.Sync(ctx, userID, []*models.SecretData{clientSecret})

		assert.Error(t, err)
		assert.Nil(t, result)

		mockRepo.AssertExpectations(t)
	})
}

func TestSecretService_GetByName(t *testing.T) {
	userID := uuid.New()

	t.Run("successful get by name", func(t *testing.T) {
		mockRepo := new(MockSecretRepository)
		secret := &models.SecretData{
			ID:            uuid.New(),
			UserID:        userID,
			Type:          models.TypeLoginPassword,
			Name:          "my-secret",
			EncryptedData: []byte("data"),
		}
		mockRepo.On("FindByUserIDAndName", mock.Anything, userID, "my-secret").Return(secret, nil)

		service := newTestSecretService(mockRepo)
		ctx := context.Background()

		result, err := service.GetByName(ctx, userID, "my-secret")

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "my-secret", result.Name)

		mockRepo.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		mockRepo := new(MockSecretRepository)
		mockRepo.On("FindByUserIDAndName", mock.Anything, userID, "nonexistent").Return(nil, apperrors.ErrSecretNotFound)

		service := newTestSecretService(mockRepo)
		ctx := context.Background()

		result, err := service.GetByName(ctx, userID, "nonexistent")

		assert.ErrorIs(t, err, apperrors.ErrSecretNotFound)
		assert.Nil(t, result)

		mockRepo.AssertExpectations(t)
	})
}

func TestSecretService_Update_AccessDenied(t *testing.T) {
	userID := uuid.New()
	otherUserID := uuid.New()
	secretID := uuid.New()

	mockRepo := new(MockSecretRepository)
	existingSecret := &models.SecretData{
		ID:            secretID,
		UserID:        otherUserID, // Different user
		Type:          models.TypeLoginPassword,
		Name:          "test-secret",
		EncryptedData: []byte("data"),
		Version:       1,
	}
	mockRepo.On("FindByID", mock.Anything, secretID).Return(existingSecret, nil)

	service := newTestSecretService(mockRepo)
	ctx := context.Background()

	secret := &models.SecretData{
		ID:            secretID,
		Type:          models.TypeLoginPassword,
		Name:          "updated-secret",
		EncryptedData: []byte("new-data"),
	}

	err := service.Update(ctx, userID, secret)

	assert.ErrorIs(t, err, apperrors.ErrSecretAccessDenied)

	mockRepo.AssertExpectations(t)
}

func TestSecretService_Update_DeletedSecret(t *testing.T) {
	userID := uuid.New()
	secretID := uuid.New()

	mockRepo := new(MockSecretRepository)
	existingSecret := &models.SecretData{
		ID:            secretID,
		UserID:        userID,
		Type:          models.TypeLoginPassword,
		Name:          "test-secret",
		EncryptedData: []byte("data"),
		Version:       1,
		DeletedAt:     sql.NullTime{Time: time.Now(), Valid: true},
	}
	mockRepo.On("FindByID", mock.Anything, secretID).Return(existingSecret, nil)

	service := newTestSecretService(mockRepo)
	ctx := context.Background()

	secret := &models.SecretData{
		ID:            secretID,
		Type:          models.TypeLoginPassword,
		Name:          "updated-secret",
		EncryptedData: []byte("new-data"),
	}

	err := service.Update(ctx, userID, secret)

	assert.ErrorIs(t, err, apperrors.ErrSecretNotFound)

	mockRepo.AssertExpectations(t)
}

func TestSecretService_Update_DBError(t *testing.T) {
	userID := uuid.New()
	secretID := uuid.New()

	mockRepo := new(MockSecretRepository)
	existingSecret := &models.SecretData{
		ID:            secretID,
		UserID:        userID,
		Type:          models.TypeLoginPassword,
		Name:          "test-secret",
		EncryptedData: []byte("data"),
		Version:       1,
	}
	mockRepo.On("FindByID", mock.Anything, secretID).Return(existingSecret, nil)
	mockRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.SecretData")).Return(assert.AnError)

	service := newTestSecretService(mockRepo)
	ctx := context.Background()

	secret := &models.SecretData{
		ID:            secretID,
		Type:          models.TypeLoginPassword,
		Name:          "updated-secret",
		EncryptedData: []byte("new-data"),
	}

	err := service.Update(ctx, userID, secret)

	assert.Error(t, err)

	mockRepo.AssertExpectations(t)
}

func TestSecretService_Delete_AccessDenied(t *testing.T) {
	userID := uuid.New()
	otherUserID := uuid.New()
	secretID := uuid.New()

	mockRepo := new(MockSecretRepository)
	existingSecret := &models.SecretData{
		ID:            secretID,
		UserID:        otherUserID, // Different user
		Type:          models.TypeLoginPassword,
		Name:          "test-secret",
		EncryptedData: []byte("data"),
	}
	mockRepo.On("FindByID", mock.Anything, secretID).Return(existingSecret, nil)

	service := newTestSecretService(mockRepo)
	ctx := context.Background()

	err := service.Delete(ctx, userID, secretID)

	assert.ErrorIs(t, err, apperrors.ErrSecretAccessDenied)

	mockRepo.AssertExpectations(t)
}

func TestSecretService_Delete_AlreadyDeleted(t *testing.T) {
	userID := uuid.New()
	secretID := uuid.New()

	mockRepo := new(MockSecretRepository)
	existingSecret := &models.SecretData{
		ID:            secretID,
		UserID:        userID,
		Type:          models.TypeLoginPassword,
		Name:          "test-secret",
		EncryptedData: []byte("data"),
		DeletedAt:     sql.NullTime{Time: time.Now(), Valid: true},
	}
	mockRepo.On("FindByID", mock.Anything, secretID).Return(existingSecret, nil)

	service := newTestSecretService(mockRepo)
	ctx := context.Background()

	err := service.Delete(ctx, userID, secretID)

	assert.ErrorIs(t, err, apperrors.ErrSecretNotFound)

	mockRepo.AssertExpectations(t)
}

func TestSecretService_Delete_DBError(t *testing.T) {
	userID := uuid.New()
	secretID := uuid.New()

	mockRepo := new(MockSecretRepository)
	existingSecret := &models.SecretData{
		ID:            secretID,
		UserID:        userID,
		Type:          models.TypeLoginPassword,
		Name:          "test-secret",
		EncryptedData: []byte("data"),
	}
	mockRepo.On("FindByID", mock.Anything, secretID).Return(existingSecret, nil)
	mockRepo.On("Delete", mock.Anything, secretID).Return(assert.AnError)

	service := newTestSecretService(mockRepo)
	ctx := context.Background()

	err := service.Delete(ctx, userID, secretID)

	assert.Error(t, err)

	mockRepo.AssertExpectations(t)
}

func TestSecretService_Get_DeletedSecret(t *testing.T) {
	userID := uuid.New()
	secretID := uuid.New()

	mockRepo := new(MockSecretRepository)
	existingSecret := &models.SecretData{
		ID:            secretID,
		UserID:        userID,
		Type:          models.TypeLoginPassword,
		Name:          "test-secret",
		EncryptedData: []byte("data"),
		DeletedAt:     sql.NullTime{Time: time.Now(), Valid: true},
	}
	mockRepo.On("FindByID", mock.Anything, secretID).Return(existingSecret, nil)

	service := newTestSecretService(mockRepo)
	ctx := context.Background()

	secret, err := service.Get(ctx, userID, secretID)

	assert.ErrorIs(t, err, apperrors.ErrSecretNotFound)
	assert.Nil(t, secret)

	mockRepo.AssertExpectations(t)
}

func TestSecretService_List_DBError(t *testing.T) {
	userID := uuid.New()

	mockRepo := new(MockSecretRepository)
	mockRepo.On("FindByUserID", mock.Anything, userID).Return(nil, assert.AnError)

	service := newTestSecretService(mockRepo)
	ctx := context.Background()

	secrets, err := service.List(ctx, userID)

	assert.Error(t, err)
	assert.Nil(t, secrets)

	mockRepo.AssertExpectations(t)
}

func TestSecretService_Create_DBError(t *testing.T) {
	userID := uuid.New()

	mockRepo := new(MockSecretRepository)
	mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.SecretData")).Return(assert.AnError)

	service := newTestSecretService(mockRepo)
	ctx := context.Background()

	secret := &models.SecretData{
		Type:          models.TypeLoginPassword,
		Name:          "test-secret",
		EncryptedData: []byte("data"),
	}

	err := service.Create(ctx, userID, secret)

	assert.Error(t, err)

	mockRepo.AssertExpectations(t)
}

