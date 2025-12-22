package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	apperrors "github.com/volchkovski/gophkeeper/internal/common/errors"
	"github.com/volchkovski/gophkeeper/internal/server/models"
	"github.com/volchkovski/gophkeeper/pkg/logger"
)

// MockUserRepository is a mock implementation of UserRepository.
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) FindByUsername(ctx context.Context, username string) (*models.User, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) Update(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func newTestAuthService(repo *MockUserRepository) AuthService {
	log := logger.NewNop()
	return NewAuthService(repo, AuthServiceConfig{
		JWTSecret:       "test-secret",
		AccessTokenTTL:  time.Hour,
		RefreshTokenTTL: 24 * time.Hour,
		BcryptCost:      bcrypt.MinCost,
	}, log)
}

func TestAuthService_Register(t *testing.T) {
	tests := []struct {
		name     string
		username string
		password string
		setup    func(*MockUserRepository)
		wantErr  error
	}{
		{
			name:     "successful registration",
			username: "testuser",
			password: "password123",
			setup: func(m *MockUserRepository) {
				m.On("FindByUsername", mock.Anything, "testuser").Return(nil, apperrors.ErrUserNotFound)
				m.On("Create", mock.Anything, mock.AnythingOfType("*models.User")).Return(nil)
			},
			wantErr: nil,
		},
		{
			name:     "empty username",
			username: "",
			password: "password123",
			setup:    func(m *MockUserRepository) {},
			wantErr:  apperrors.ErrEmptyUsername,
		},
		{
			name:     "username too short",
			username: "ab",
			password: "password123",
			setup:    func(m *MockUserRepository) {},
			wantErr:  apperrors.ErrUsernameTooShort,
		},
		{
			name:     "empty password",
			username: "testuser",
			password: "",
			setup:    func(m *MockUserRepository) {},
			wantErr:  apperrors.ErrEmptyPassword,
		},
		{
			name:     "password too short",
			username: "testuser",
			password: "short",
			setup:    func(m *MockUserRepository) {},
			wantErr:  apperrors.ErrPasswordTooShort,
		},
		{
			name:     "user already exists",
			username: "existinguser",
			password: "password123",
			setup: func(m *MockUserRepository) {
				existingUser := &models.User{
					ID:       uuid.New(),
					Username: "existinguser",
				}
				m.On("FindByUsername", mock.Anything, "existinguser").Return(existingUser, nil)
			},
			wantErr: apperrors.ErrUserAlreadyExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockUserRepository)
			tt.setup(mockRepo)

			service := newTestAuthService(mockRepo)
			ctx := context.Background()

			user, err := service.Register(ctx, tt.username, tt.password)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, tt.username, user.Username)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestAuthService_Login(t *testing.T) {
	// Create a valid password hash
	passwordHash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.MinCost)

	tests := []struct {
		name     string
		username string
		password string
		setup    func(*MockUserRepository)
		wantErr  error
	}{
		{
			name:     "successful login",
			username: "testuser",
			password: "password123",
			setup: func(m *MockUserRepository) {
				user := &models.User{
					ID:           uuid.New(),
					Username:     "testuser",
					PasswordHash: string(passwordHash),
				}
				m.On("FindByUsername", mock.Anything, "testuser").Return(user, nil)
			},
			wantErr: nil,
		},
		{
			name:     "user not found",
			username: "nonexistent",
			password: "password123",
			setup: func(m *MockUserRepository) {
				m.On("FindByUsername", mock.Anything, "nonexistent").Return(nil, apperrors.ErrUserNotFound)
			},
			wantErr: apperrors.ErrInvalidCredentials,
		},
		{
			name:     "wrong password",
			username: "testuser",
			password: "wrongpassword",
			setup: func(m *MockUserRepository) {
				user := &models.User{
					ID:           uuid.New(),
					Username:     "testuser",
					PasswordHash: string(passwordHash),
				}
				m.On("FindByUsername", mock.Anything, "testuser").Return(user, nil)
			},
			wantErr: apperrors.ErrInvalidCredentials,
		},
		{
			name:     "empty username",
			username: "",
			password: "password123",
			setup:    func(m *MockUserRepository) {},
			wantErr:  apperrors.ErrEmptyUsername,
		},
		{
			name:     "empty password",
			username: "testuser",
			password: "",
			setup:    func(m *MockUserRepository) {},
			wantErr:  apperrors.ErrEmptyPassword,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockUserRepository)
			tt.setup(mockRepo)

			service := newTestAuthService(mockRepo)
			ctx := context.Background()

			tokenPair, err := service.Login(ctx, tt.username, tt.password)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, tokenPair)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, tokenPair)
				assert.NotEmpty(t, tokenPair.AccessToken)
				assert.NotEmpty(t, tokenPair.RefreshToken)
				assert.Greater(t, tokenPair.ExpiresIn, int64(0))
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestAuthService_ValidateToken(t *testing.T) {
	// Create a user and generate a valid token
	passwordHash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.MinCost)

	mockRepo := new(MockUserRepository)
	user := &models.User{
		ID:           uuid.New(),
		Username:     "testuser",
		PasswordHash: string(passwordHash),
	}
	mockRepo.On("FindByUsername", mock.Anything, "testuser").Return(user, nil)

	service := newTestAuthService(mockRepo)
	ctx := context.Background()

	// Login to get valid tokens
	tokenPair, err := service.Login(ctx, "testuser", "password123")
	require.NoError(t, err)

	tests := []struct {
		name    string
		token   string
		wantErr bool
	}{
		{
			name:    "valid token",
			token:   tokenPair.AccessToken,
			wantErr: false,
		},
		{
			name:    "invalid token",
			token:   "invalid.token.here",
			wantErr: true,
		},
		{
			name:    "empty token",
			token:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := service.ValidateToken(tt.token)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, claims)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, claims)
				assert.Equal(t, user.ID, claims.UserID)
				assert.Equal(t, user.Username, claims.Username)
			}
		})
	}
}

func TestAuthService_RefreshToken(t *testing.T) {
	passwordHash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.MinCost)
	userID := uuid.New()

	mockRepo := new(MockUserRepository)
	user := &models.User{
		ID:           userID,
		Username:     "testuser",
		PasswordHash: string(passwordHash),
	}
	mockRepo.On("FindByUsername", mock.Anything, "testuser").Return(user, nil)
	mockRepo.On("FindByID", mock.Anything, userID).Return(user, nil)

	service := newTestAuthService(mockRepo)
	ctx := context.Background()

	// Login to get valid tokens
	tokenPair, err := service.Login(ctx, "testuser", "password123")
	require.NoError(t, err)

	// Test refresh
	newTokenPair, err := service.RefreshToken(ctx, tokenPair.RefreshToken)
	assert.NoError(t, err)
	assert.NotNil(t, newTokenPair)
	assert.NotEmpty(t, newTokenPair.AccessToken)
	// Note: tokens might be same if generated within same second, so just check they exist
	assert.NotEmpty(t, newTokenPair.RefreshToken)
}

func TestAuthService_RefreshToken_InvalidToken(t *testing.T) {
	mockRepo := new(MockUserRepository)
	service := newTestAuthService(mockRepo)
	ctx := context.Background()

	// Test with invalid token
	tokenPair, err := service.RefreshToken(ctx, "invalid-token")
	assert.Error(t, err)
	assert.Nil(t, tokenPair)
	assert.ErrorIs(t, err, apperrors.ErrInvalidToken)
}

func TestAuthService_RefreshToken_UserNotFound(t *testing.T) {
	passwordHash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.MinCost)
	userID := uuid.New()

	mockRepo := new(MockUserRepository)
	user := &models.User{
		ID:           userID,
		Username:     "testuser",
		PasswordHash: string(passwordHash),
	}
	mockRepo.On("FindByUsername", mock.Anything, "testuser").Return(user, nil)
	mockRepo.On("FindByID", mock.Anything, userID).Return(nil, apperrors.ErrUserNotFound)

	service := newTestAuthService(mockRepo)
	ctx := context.Background()

	// Login to get valid tokens
	tokenPair, err := service.Login(ctx, "testuser", "password123")
	require.NoError(t, err)

	// User deleted after login
	newTokenPair, err := service.RefreshToken(ctx, tokenPair.RefreshToken)
	assert.Error(t, err)
	assert.Nil(t, newTokenPair)
	assert.ErrorIs(t, err, apperrors.ErrInvalidToken)
}

func TestAuthService_RefreshToken_DBError(t *testing.T) {
	passwordHash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.MinCost)
	userID := uuid.New()

	mockRepo := new(MockUserRepository)
	user := &models.User{
		ID:           userID,
		Username:     "testuser",
		PasswordHash: string(passwordHash),
	}
	mockRepo.On("FindByUsername", mock.Anything, "testuser").Return(user, nil)
	mockRepo.On("FindByID", mock.Anything, userID).Return(nil, assert.AnError)

	service := newTestAuthService(mockRepo)
	ctx := context.Background()

	// Login to get valid tokens
	tokenPair, err := service.Login(ctx, "testuser", "password123")
	require.NoError(t, err)

	// DB error on refresh
	newTokenPair, err := service.RefreshToken(ctx, tokenPair.RefreshToken)
	assert.Error(t, err)
	assert.Nil(t, newTokenPair)
}

func TestAuthService_Register_UsernameTooLong(t *testing.T) {
	mockRepo := new(MockUserRepository)
	service := newTestAuthService(mockRepo)
	ctx := context.Background()

	longUsername := "a"
	for i := 0; i < 60; i++ {
		longUsername += "a"
	}

	user, err := service.Register(ctx, longUsername, "password123")
	assert.ErrorIs(t, err, apperrors.ErrUsernameTooLong)
	assert.Nil(t, user)
}

func TestAuthService_Register_DBError(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockRepo.On("FindByUsername", mock.Anything, "testuser").Return(nil, apperrors.ErrUserNotFound)
	mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.User")).Return(assert.AnError)

	service := newTestAuthService(mockRepo)
	ctx := context.Background()

	user, err := service.Register(ctx, "testuser", "password123")
	assert.Error(t, err)
	assert.Nil(t, user)

	mockRepo.AssertExpectations(t)
}

func TestAuthService_Register_FindByUsernameDBError(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockRepo.On("FindByUsername", mock.Anything, "testuser").Return(nil, assert.AnError)

	service := newTestAuthService(mockRepo)
	ctx := context.Background()

	user, err := service.Register(ctx, "testuser", "password123")
	assert.Error(t, err)
	assert.Nil(t, user)

	mockRepo.AssertExpectations(t)
}

func TestAuthService_Login_DBError(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockRepo.On("FindByUsername", mock.Anything, "testuser").Return(nil, assert.AnError)

	service := newTestAuthService(mockRepo)
	ctx := context.Background()

	tokenPair, err := service.Login(ctx, "testuser", "password123")
	assert.Error(t, err)
	assert.Nil(t, tokenPair)

	mockRepo.AssertExpectations(t)
}

