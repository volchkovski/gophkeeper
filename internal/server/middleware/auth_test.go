package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/volchkovski/gophkeeper/internal/server/models"
	"github.com/volchkovski/gophkeeper/internal/server/service"
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

func setupTestRouter(authService service.AuthService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.Use(AuthMiddleware(authService))

	router.GET("/protected", func(c *gin.Context) {
		userID, _ := c.Get(UserIDKey)
		username, _ := c.Get(UsernameKey)
		c.JSON(http.StatusOK, gin.H{
			"user_id":  userID,
			"username": username,
		})
	})

	return router
}

func TestAuthMiddleware(t *testing.T) {
	userID := uuid.New()
	username := "testuser"

	tests := []struct {
		name           string
		authHeader     string
		setupMock      func(*MockAuthService)
		expectedStatus int
		expectedError  string
	}{
		{
			name:       "valid token",
			authHeader: "Bearer valid-token",
			setupMock: func(m *MockAuthService) {
				claims := &service.Claims{
					UserID:   userID,
					Username: username,
				}
				m.On("ValidateToken", "valid-token").Return(claims, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "missing authorization header",
			authHeader:     "",
			setupMock:      func(m *MockAuthService) {},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "authorization header is required",
		},
		{
			name:           "invalid authorization header format - no Bearer prefix",
			authHeader:     "Basic token123",
			setupMock:      func(m *MockAuthService) {},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "invalid authorization header format",
		},
		{
			name:           "empty token after Bearer prefix",
			authHeader:     "Bearer ",
			setupMock:      func(m *MockAuthService) {},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "token is required",
		},
		{
			name:       "invalid token",
			authHeader: "Bearer invalid-token",
			setupMock: func(m *MockAuthService) {
				m.On("ValidateToken", "invalid-token").Return(nil, assert.AnError)
			},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "invalid or expired token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAuth := new(MockAuthService)
			tt.setupMock(mockAuth)

			router := setupTestRouter(mockAuth)

			req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				assert.Contains(t, w.Body.String(), tt.expectedError)
			}

			mockAuth.AssertExpectations(t)
		})
	}
}

func TestAuthMiddleware_SetsContextValues(t *testing.T) {
	userID := uuid.New()
	username := "testuser"

	mockAuth := new(MockAuthService)
	claims := &service.Claims{
		UserID:   userID,
		Username: username,
	}
	mockAuth.On("ValidateToken", "valid-token").Return(claims, nil)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(AuthMiddleware(mockAuth))

	var capturedUserID uuid.UUID
	var capturedUsername string

	router.GET("/protected", func(c *gin.Context) {
		userIDValue, _ := c.Get(UserIDKey)
		capturedUserID = userIDValue.(uuid.UUID)
		usernameValue, _ := c.Get(UsernameKey)
		capturedUsername = usernameValue.(string)
		c.Status(http.StatusOK)
	})

	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer valid-token")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, userID, capturedUserID)
	assert.Equal(t, username, capturedUsername)

	mockAuth.AssertExpectations(t)
}

