package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	apperrors "github.com/volchkovski/gophkeeper/internal/common/errors"
	"github.com/volchkovski/gophkeeper/internal/server/models"
	"github.com/volchkovski/gophkeeper/internal/server/service"
	"github.com/volchkovski/gophkeeper/pkg/logger"
)

// MockAuthService is a mock implementation of AuthService.
type MockAuthService struct {
	mock.Mock
}

func (m *MockAuthService) Register(ctx context.Context, username, password string) (*models.User, error) {
	args := m.Called(ctx, username, password)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockAuthService) Login(ctx context.Context, username, password string) (*service.TokenPair, error) {
	args := m.Called(ctx, username, password)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.TokenPair), args.Error(1)
}

func (m *MockAuthService) RefreshToken(ctx context.Context, refreshToken string) (*service.TokenPair, error) {
	args := m.Called(ctx, refreshToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.TokenPair), args.Error(1)
}

func (m *MockAuthService) ValidateToken(token string) (*service.Claims, error) {
	args := m.Called(token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.Claims), args.Error(1)
}

// MockSecretService is a mock implementation of SecretService.
type MockSecretService struct {
	mock.Mock
}

func (m *MockSecretService) Create(ctx context.Context, userID uuid.UUID, secret *models.SecretData) error {
	args := m.Called(ctx, userID, secret)
	return args.Error(0)
}

func (m *MockSecretService) Update(ctx context.Context, userID uuid.UUID, secret *models.SecretData) error {
	args := m.Called(ctx, userID, secret)
	return args.Error(0)
}

func (m *MockSecretService) Delete(ctx context.Context, userID uuid.UUID, secretID uuid.UUID) error {
	args := m.Called(ctx, userID, secretID)
	return args.Error(0)
}

func (m *MockSecretService) Get(ctx context.Context, userID uuid.UUID, secretID uuid.UUID) (*models.SecretData, error) {
	args := m.Called(ctx, userID, secretID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.SecretData), args.Error(1)
}

func (m *MockSecretService) GetByName(ctx context.Context, userID uuid.UUID, name string) (*models.SecretData, error) {
	args := m.Called(ctx, userID, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.SecretData), args.Error(1)
}

func (m *MockSecretService) List(ctx context.Context, userID uuid.UUID) ([]*models.SecretData, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.SecretData), args.Error(1)
}

func (m *MockSecretService) Sync(ctx context.Context, userID uuid.UUID, clientSecrets []*models.SecretData) (*service.SyncResult, error) {
	args := m.Called(ctx, userID, clientSecrets)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.SyncResult), args.Error(1)
}

func setupTestRouter(mockAuth *MockAuthService, mockSecret *MockSecretService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	
	log := logger.NewNop()
	handler := NewHandler(mockAuth, mockSecret, log)
	
	router := gin.New()
	v1 := router.Group("/api/v1")
	{
		auth := v1.Group("/auth")
		{
			auth.POST("/register", handler.Register)
			auth.POST("/login", handler.Login)
			auth.POST("/refresh", handler.RefreshToken)
		}
	}
	
	return router
}

func TestHandler_Register(t *testing.T) {
	tests := []struct {
		name           string
		body           RegisterRequest
		setupMock      func(*MockAuthService)
		expectedStatus int
		expectedError  string
	}{
		{
			name: "successful registration",
			body: RegisterRequest{
				Username: "testuser",
				Password: "password123",
			},
			setupMock: func(m *MockAuthService) {
				user := &models.User{
					ID:        uuid.New(),
					Username:  "testuser",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				m.On("Register", mock.Anything, "testuser", "password123").Return(user, nil)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "user already exists",
			body: RegisterRequest{
				Username: "existinguser",
				Password: "password123",
			},
			setupMock: func(m *MockAuthService) {
				m.On("Register", mock.Anything, "existinguser", "password123").Return(nil, apperrors.ErrUserAlreadyExists)
			},
			expectedStatus: http.StatusConflict,
			expectedError:  "user already exists",
		},
		{
			name: "invalid request - missing username",
			body: RegisterRequest{
				Password: "password123",
			},
			setupMock:      func(m *MockAuthService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid request body",
		},
		{
			name: "password too short from service",
			body: RegisterRequest{
				Username: "testuser",
				Password: "short123", // min 8 chars to pass gin validation
			},
			setupMock: func(m *MockAuthService) {
				m.On("Register", mock.Anything, "testuser", "short123").Return(nil, apperrors.ErrPasswordTooShort)
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAuth := new(MockAuthService)
			mockSecret := new(MockSecretService)
			tt.setupMock(mockAuth)

			router := setupTestRouter(mockAuth, mockSecret)

			body, _ := json.Marshal(tt.body)
			req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var resp ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.Contains(t, resp.Error, tt.expectedError)
			}

			mockAuth.AssertExpectations(t)
		})
	}
}

func TestHandler_Login(t *testing.T) {
	tests := []struct {
		name           string
		body           LoginRequest
		setupMock      func(*MockAuthService)
		expectedStatus int
		expectedError  string
	}{
		{
			name: "successful login",
			body: LoginRequest{
				Username: "testuser",
				Password: "password123",
			},
			setupMock: func(m *MockAuthService) {
				tokenPair := &service.TokenPair{
					AccessToken:  "access-token",
					RefreshToken: "refresh-token",
					ExpiresIn:    3600,
				}
				m.On("Login", mock.Anything, "testuser", "password123").Return(tokenPair, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "invalid credentials",
			body: LoginRequest{
				Username: "testuser",
				Password: "wrongpassword",
			},
			setupMock: func(m *MockAuthService) {
				m.On("Login", mock.Anything, "testuser", "wrongpassword").Return(nil, apperrors.ErrInvalidCredentials)
			},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "invalid credentials",
		},
		{
			name: "invalid request - missing password",
			body: LoginRequest{
				Username: "testuser",
			},
			setupMock:      func(m *MockAuthService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid request body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAuth := new(MockAuthService)
			mockSecret := new(MockSecretService)
			tt.setupMock(mockAuth)

			router := setupTestRouter(mockAuth, mockSecret)

			body, _ := json.Marshal(tt.body)
			req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var resp ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.Contains(t, resp.Error, tt.expectedError)
			}

			if tt.expectedStatus == http.StatusOK {
				var resp LoginResponse
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.NotEmpty(t, resp.AccessToken)
				assert.NotEmpty(t, resp.RefreshToken)
			}

			mockAuth.AssertExpectations(t)
		})
	}
}

func TestHandler_RefreshToken(t *testing.T) {
	tests := []struct {
		name           string
		body           RefreshRequest
		setupMock      func(*MockAuthService)
		expectedStatus int
		expectedError  string
	}{
		{
			name: "successful refresh",
			body: RefreshRequest{
				RefreshToken: "valid-refresh-token",
			},
			setupMock: func(m *MockAuthService) {
				tokenPair := &service.TokenPair{
					AccessToken:  "new-access-token",
					RefreshToken: "new-refresh-token",
					ExpiresIn:    3600,
				}
				m.On("RefreshToken", mock.Anything, "valid-refresh-token").Return(tokenPair, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "invalid refresh token",
			body: RefreshRequest{
				RefreshToken: "invalid-token",
			},
			setupMock: func(m *MockAuthService) {
				m.On("RefreshToken", mock.Anything, "invalid-token").Return(nil, apperrors.ErrInvalidToken)
			},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "invalid or expired token",
		},
		{
			name: "expired token",
			body: RefreshRequest{
				RefreshToken: "expired-token",
			},
			setupMock: func(m *MockAuthService) {
				m.On("RefreshToken", mock.Anything, "expired-token").Return(nil, apperrors.ErrTokenExpired)
			},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "invalid or expired token",
		},
		{
			name: "internal server error",
			body: RefreshRequest{
				RefreshToken: "some-token",
			},
			setupMock: func(m *MockAuthService) {
				m.On("RefreshToken", mock.Anything, "some-token").Return(nil, assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "invalid request - missing token",
			body: RefreshRequest{
				RefreshToken: "",
			},
			setupMock:      func(m *MockAuthService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid request body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAuth := new(MockAuthService)
			mockSecret := new(MockSecretService)
			tt.setupMock(mockAuth)

			router := setupTestRouter(mockAuth, mockSecret)

			body, _ := json.Marshal(tt.body)
			req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var resp ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.Contains(t, resp.Error, tt.expectedError)
			}

			mockAuth.AssertExpectations(t)
		})
	}
}

func TestHandler_Login_AdditionalCases(t *testing.T) {
	tests := []struct {
		name           string
		body           LoginRequest
		setupMock      func(*MockAuthService)
		expectedStatus int
		expectedError  string
	}{
		{
			name: "empty username from service",
			body: LoginRequest{
				Username: "testuser",
				Password: "password123",
			},
			setupMock: func(m *MockAuthService) {
				m.On("Login", mock.Anything, "testuser", "password123").Return(nil, apperrors.ErrEmptyUsername)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "empty password from service",
			body: LoginRequest{
				Username: "testuser",
				Password: "password123",
			},
			setupMock: func(m *MockAuthService) {
				m.On("Login", mock.Anything, "testuser", "password123").Return(nil, apperrors.ErrEmptyPassword)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "internal server error",
			body: LoginRequest{
				Username: "testuser",
				Password: "password123",
			},
			setupMock: func(m *MockAuthService) {
				m.On("Login", mock.Anything, "testuser", "password123").Return(nil, assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "user not found",
			body: LoginRequest{
				Username: "unknownuser",
				Password: "password123",
			},
			setupMock: func(m *MockAuthService) {
				m.On("Login", mock.Anything, "unknownuser", "password123").Return(nil, apperrors.ErrUserNotFound)
			},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "invalid credentials",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAuth := new(MockAuthService)
			mockSecret := new(MockSecretService)
			tt.setupMock(mockAuth)

			router := setupTestRouter(mockAuth, mockSecret)

			body, _ := json.Marshal(tt.body)
			req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var resp ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.Contains(t, resp.Error, tt.expectedError)
			}

			mockAuth.AssertExpectations(t)
		})
	}
}

func TestHandler_Register_AdditionalCases(t *testing.T) {
	tests := []struct {
		name           string
		body           RegisterRequest
		setupMock      func(*MockAuthService)
		expectedStatus int
	}{
		{
			name: "internal server error",
			body: RegisterRequest{
				Username: "testuser",
				Password: "password123",
			},
			setupMock: func(m *MockAuthService) {
				m.On("Register", mock.Anything, "testuser", "password123").Return(nil, assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "empty username from service",
			body: RegisterRequest{
				Username: "abc",
				Password: "password123",
			},
			setupMock: func(m *MockAuthService) {
				m.On("Register", mock.Anything, "abc", "password123").Return(nil, apperrors.ErrEmptyUsername)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "username too long",
			body: RegisterRequest{
				Username: "abcdefghij",
				Password: "password123",
			},
			setupMock: func(m *MockAuthService) {
				m.On("Register", mock.Anything, "abcdefghij", "password123").Return(nil, apperrors.ErrUsernameTooLong)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "empty password from service",
			body: RegisterRequest{
				Username: "testuser",
				Password: "password123",
			},
			setupMock: func(m *MockAuthService) {
				m.On("Register", mock.Anything, "testuser", "password123").Return(nil, apperrors.ErrEmptyPassword)
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAuth := new(MockAuthService)
			mockSecret := new(MockSecretService)
			tt.setupMock(mockAuth)

			router := setupTestRouter(mockAuth, mockSecret)

			body, _ := json.Marshal(tt.body)
			req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			mockAuth.AssertExpectations(t)
		})
	}
}
