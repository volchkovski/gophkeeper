package handlers

import (
	"bytes"
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
	"github.com/volchkovski/gophkeeper/internal/server/middleware"
	"github.com/volchkovski/gophkeeper/internal/server/models"
	"github.com/volchkovski/gophkeeper/internal/server/service"
	"github.com/volchkovski/gophkeeper/pkg/logger"
)

func setupSecretTestRouter(mockAuth *MockAuthService, mockSecret *MockSecretService) *gin.Engine {
	gin.SetMode(gin.TestMode)

	log := logger.NewNop()
	handler := NewHandler(mockAuth, mockSecret, log)

	router := gin.New()
	v1 := router.Group("/api/v1")
	{
		secrets := v1.Group("/secrets")
		{
			secrets.POST("", handler.CreateSecret)
			secrets.GET("", handler.ListSecrets)
			secrets.GET("/:id", handler.GetSecret)
			secrets.PUT("/:id", handler.UpdateSecret)
			secrets.DELETE("/:id", handler.DeleteSecret)
			secrets.POST("/sync", handler.SyncSecrets)
		}
	}

	return router
}

func setupAuthenticatedSecretRouter(mockAuth *MockAuthService, mockSecret *MockSecretService, userID uuid.UUID) *gin.Engine {
	gin.SetMode(gin.TestMode)

	log := logger.NewNop()
	handler := NewHandler(mockAuth, mockSecret, log)

	router := gin.New()

	// Add middleware to set user ID
	router.Use(func(c *gin.Context) {
		c.Set(middleware.UserIDKey, userID)
		c.Next()
	})

	v1 := router.Group("/api/v1")
	{
		secrets := v1.Group("/secrets")
		{
			secrets.POST("", handler.CreateSecret)
			secrets.GET("", handler.ListSecrets)
			secrets.GET("/:id", handler.GetSecret)
			secrets.PUT("/:id", handler.UpdateSecret)
			secrets.DELETE("/:id", handler.DeleteSecret)
			secrets.POST("/sync", handler.SyncSecrets)
		}
	}

	return router
}

func TestHandler_CreateSecret(t *testing.T) {
	userID := uuid.New()

	tests := []struct {
		name           string
		body           CreateSecretRequest
		setupMock      func(*MockSecretService)
		expectedStatus int
		expectedError  string
	}{
		{
			name: "successful creation",
			body: CreateSecretRequest{
				Type:          "login_password",
				Name:          "test-secret",
				EncryptedData: []byte("encrypted-data"),
				Metadata:      "test metadata",
			},
			setupMock: func(m *MockSecretService) {
				m.On("Create", mock.Anything, userID, mock.AnythingOfType("*models.SecretData")).Return(nil)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "invalid secret type",
			body: CreateSecretRequest{
				Type:          "invalid_type",
				Name:          "test-secret",
				EncryptedData: []byte("encrypted-data"),
			},
			setupMock:      func(m *MockSecretService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid secret type",
		},
		{
			name: "secret name already exists",
			body: CreateSecretRequest{
				Type:          "login_password",
				Name:          "existing-secret",
				EncryptedData: []byte("encrypted-data"),
			},
			setupMock: func(m *MockSecretService) {
				m.On("Create", mock.Anything, userID, mock.AnythingOfType("*models.SecretData")).Return(apperrors.ErrSecretNameExists)
			},
			expectedStatus: http.StatusConflict,
			expectedError:  "secret with this name already exists",
		},
		{
			name: "invalid request - missing name",
			body: CreateSecretRequest{
				Type:          "login_password",
				EncryptedData: []byte("encrypted-data"),
			},
			setupMock:      func(m *MockSecretService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid request body",
		},
		{
			name: "internal server error",
			body: CreateSecretRequest{
				Type:          "login_password",
				Name:          "test-secret",
				EncryptedData: []byte("encrypted-data"),
			},
			setupMock: func(m *MockSecretService) {
				m.On("Create", mock.Anything, userID, mock.AnythingOfType("*models.SecretData")).Return(assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAuth := new(MockAuthService)
			mockSecret := new(MockSecretService)
			tt.setupMock(mockSecret)

			router := setupAuthenticatedSecretRouter(mockAuth, mockSecret, userID)

			body, _ := json.Marshal(tt.body)
			req, _ := http.NewRequest(http.MethodPost, "/api/v1/secrets", bytes.NewBuffer(body))
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

			mockSecret.AssertExpectations(t)
		})
	}
}

func TestHandler_CreateSecret_Unauthorized(t *testing.T) {
	mockAuth := new(MockAuthService)
	mockSecret := new(MockSecretService)

	// Use router without authentication middleware
	router := setupSecretTestRouter(mockAuth, mockSecret)

	body, _ := json.Marshal(CreateSecretRequest{
		Type:          "login_password",
		Name:          "test-secret",
		EncryptedData: []byte("encrypted-data"),
	})
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/secrets", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetSecret(t *testing.T) {
	userID := uuid.New()
	secretID := uuid.New()

	tests := []struct {
		name           string
		secretID       string
		setupMock      func(*MockSecretService)
		expectedStatus int
		expectedError  string
	}{
		{
			name:     "successful get",
			secretID: secretID.String(),
			setupMock: func(m *MockSecretService) {
				secret := &models.SecretData{
					ID:            secretID,
					UserID:        userID,
					Type:          models.TypeLoginPassword,
					Name:          "test-secret",
					EncryptedData: []byte("encrypted-data"),
					Version:       1,
					CreatedAt:     time.Now(),
					UpdatedAt:     time.Now(),
				}
				m.On("Get", mock.Anything, userID, secretID).Return(secret, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid secret ID",
			secretID:       "not-a-uuid",
			setupMock:      func(m *MockSecretService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid secret ID",
		},
		{
			name:     "secret not found",
			secretID: secretID.String(),
			setupMock: func(m *MockSecretService) {
				m.On("Get", mock.Anything, userID, secretID).Return(nil, apperrors.ErrSecretNotFound)
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  "secret not found",
		},
		{
			name:     "access denied",
			secretID: secretID.String(),
			setupMock: func(m *MockSecretService) {
				m.On("Get", mock.Anything, userID, secretID).Return(nil, apperrors.ErrSecretAccessDenied)
			},
			expectedStatus: http.StatusForbidden,
			expectedError:  "access denied",
		},
		{
			name:     "internal server error",
			secretID: secretID.String(),
			setupMock: func(m *MockSecretService) {
				m.On("Get", mock.Anything, userID, secretID).Return(nil, assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAuth := new(MockAuthService)
			mockSecret := new(MockSecretService)
			tt.setupMock(mockSecret)

			router := setupAuthenticatedSecretRouter(mockAuth, mockSecret, userID)

			req, _ := http.NewRequest(http.MethodGet, "/api/v1/secrets/"+tt.secretID, nil)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var resp ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.Contains(t, resp.Error, tt.expectedError)
			}

			mockSecret.AssertExpectations(t)
		})
	}
}

func TestHandler_ListSecrets(t *testing.T) {
	userID := uuid.New()

	tests := []struct {
		name           string
		setupMock      func(*MockSecretService)
		expectedStatus int
		expectedLen    int
	}{
		{
			name: "successful list with secrets",
			setupMock: func(m *MockSecretService) {
				secrets := []*models.SecretData{
					{
						ID:            uuid.New(),
						UserID:        userID,
						Type:          models.TypeLoginPassword,
						Name:          "secret-1",
						EncryptedData: []byte("data-1"),
						Version:       1,
						CreatedAt:     time.Now(),
						UpdatedAt:     time.Now(),
					},
					{
						ID:            uuid.New(),
						UserID:        userID,
						Type:          models.TypeText,
						Name:          "secret-2",
						EncryptedData: []byte("data-2"),
						Version:       1,
						CreatedAt:     time.Now(),
						UpdatedAt:     time.Now(),
					},
				}
				m.On("List", mock.Anything, userID).Return(secrets, nil)
			},
			expectedStatus: http.StatusOK,
			expectedLen:    2,
		},
		{
			name: "empty list",
			setupMock: func(m *MockSecretService) {
				m.On("List", mock.Anything, userID).Return([]*models.SecretData{}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedLen:    0,
		},
		{
			name: "internal server error",
			setupMock: func(m *MockSecretService) {
				m.On("List", mock.Anything, userID).Return(nil, assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAuth := new(MockAuthService)
			mockSecret := new(MockSecretService)
			tt.setupMock(mockSecret)

			router := setupAuthenticatedSecretRouter(mockAuth, mockSecret, userID)

			req, _ := http.NewRequest(http.MethodGet, "/api/v1/secrets", nil)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var resp SecretsListResponse
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.Len(t, resp.Secrets, tt.expectedLen)
			}

			mockSecret.AssertExpectations(t)
		})
	}
}

func TestHandler_UpdateSecret(t *testing.T) {
	userID := uuid.New()
	secretID := uuid.New()

	tests := []struct {
		name           string
		secretID       string
		body           UpdateSecretRequest
		setupMock      func(*MockSecretService)
		expectedStatus int
		expectedError  string
	}{
		{
			name:     "successful update",
			secretID: secretID.String(),
			body: UpdateSecretRequest{
				Type:          "login_password",
				Name:          "updated-secret",
				EncryptedData: []byte("new-encrypted-data"),
			},
			setupMock: func(m *MockSecretService) {
				m.On("Update", mock.Anything, userID, mock.AnythingOfType("*models.SecretData")).Return(nil)
				updatedSecret := &models.SecretData{
					ID:            secretID,
					UserID:        userID,
					Type:          models.TypeLoginPassword,
					Name:          "updated-secret",
					EncryptedData: []byte("new-encrypted-data"),
					Version:       2,
					CreatedAt:     time.Now(),
					UpdatedAt:     time.Now(),
				}
				m.On("Get", mock.Anything, userID, secretID).Return(updatedSecret, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:     "invalid secret ID",
			secretID: "not-a-uuid",
			body: UpdateSecretRequest{
				Type:          "login_password",
				Name:          "updated-secret",
				EncryptedData: []byte("new-encrypted-data"),
			},
			setupMock:      func(m *MockSecretService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid secret ID",
		},
		{
			name:     "invalid secret type",
			secretID: secretID.String(),
			body: UpdateSecretRequest{
				Type:          "invalid_type",
				Name:          "updated-secret",
				EncryptedData: []byte("new-encrypted-data"),
			},
			setupMock:      func(m *MockSecretService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid secret type",
		},
		{
			name:     "secret not found",
			secretID: secretID.String(),
			body: UpdateSecretRequest{
				Type:          "login_password",
				Name:          "updated-secret",
				EncryptedData: []byte("new-encrypted-data"),
			},
			setupMock: func(m *MockSecretService) {
				m.On("Update", mock.Anything, userID, mock.AnythingOfType("*models.SecretData")).Return(apperrors.ErrSecretNotFound)
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  "secret not found",
		},
		{
			name:     "access denied",
			secretID: secretID.String(),
			body: UpdateSecretRequest{
				Type:          "login_password",
				Name:          "updated-secret",
				EncryptedData: []byte("new-encrypted-data"),
			},
			setupMock: func(m *MockSecretService) {
				m.On("Update", mock.Anything, userID, mock.AnythingOfType("*models.SecretData")).Return(apperrors.ErrSecretAccessDenied)
			},
			expectedStatus: http.StatusForbidden,
			expectedError:  "access denied",
		},
		{
			name:     "get after update fails",
			secretID: secretID.String(),
			body: UpdateSecretRequest{
				Type:          "login_password",
				Name:          "updated-secret",
				EncryptedData: []byte("new-encrypted-data"),
			},
			setupMock: func(m *MockSecretService) {
				m.On("Update", mock.Anything, userID, mock.AnythingOfType("*models.SecretData")).Return(nil)
				m.On("Get", mock.Anything, userID, secretID).Return(nil, assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:     "invalid request - missing name",
			secretID: secretID.String(),
			body: UpdateSecretRequest{
				Type:          "login_password",
				EncryptedData: []byte("new-encrypted-data"),
			},
			setupMock:      func(m *MockSecretService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid request body",
		},
		{
			name:     "secret name already exists",
			secretID: secretID.String(),
			body: UpdateSecretRequest{
				Type:          "login_password",
				Name:          "existing-name",
				EncryptedData: []byte("data"),
			},
			setupMock: func(m *MockSecretService) {
				m.On("Update", mock.Anything, userID, mock.AnythingOfType("*models.SecretData")).Return(apperrors.ErrSecretNameExists)
			},
			expectedStatus: http.StatusConflict,
			expectedError:  "secret with this name already exists",
		},
		{
			name:     "invalid secret name from service",
			secretID: secretID.String(),
			body: UpdateSecretRequest{
				Type:          "login_password",
				Name:          "invalid",
				EncryptedData: []byte("data"),
			},
			setupMock: func(m *MockSecretService) {
				m.On("Update", mock.Anything, userID, mock.AnythingOfType("*models.SecretData")).Return(apperrors.ErrInvalidSecretName)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:     "invalid input from service",
			secretID: secretID.String(),
			body: UpdateSecretRequest{
				Type:          "login_password",
				Name:          "name",
				EncryptedData: []byte("data"),
			},
			setupMock: func(m *MockSecretService) {
				m.On("Update", mock.Anything, userID, mock.AnythingOfType("*models.SecretData")).Return(apperrors.ErrInvalidInput)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:     "internal server error",
			secretID: secretID.String(),
			body: UpdateSecretRequest{
				Type:          "login_password",
				Name:          "name",
				EncryptedData: []byte("data"),
			},
			setupMock: func(m *MockSecretService) {
				m.On("Update", mock.Anything, userID, mock.AnythingOfType("*models.SecretData")).Return(assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAuth := new(MockAuthService)
			mockSecret := new(MockSecretService)
			tt.setupMock(mockSecret)

			router := setupAuthenticatedSecretRouter(mockAuth, mockSecret, userID)

			body, _ := json.Marshal(tt.body)
			req, _ := http.NewRequest(http.MethodPut, "/api/v1/secrets/"+tt.secretID, bytes.NewBuffer(body))
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

			mockSecret.AssertExpectations(t)
		})
	}
}

func TestHandler_DeleteSecret(t *testing.T) {
	userID := uuid.New()
	secretID := uuid.New()

	tests := []struct {
		name           string
		secretID       string
		setupMock      func(*MockSecretService)
		expectedStatus int
		expectedError  string
	}{
		{
			name:     "successful delete",
			secretID: secretID.String(),
			setupMock: func(m *MockSecretService) {
				m.On("Delete", mock.Anything, userID, secretID).Return(nil)
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "invalid secret ID",
			secretID:       "not-a-uuid",
			setupMock:      func(m *MockSecretService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid secret ID",
		},
		{
			name:     "secret not found",
			secretID: secretID.String(),
			setupMock: func(m *MockSecretService) {
				m.On("Delete", mock.Anything, userID, secretID).Return(apperrors.ErrSecretNotFound)
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  "secret not found",
		},
		{
			name:     "access denied",
			secretID: secretID.String(),
			setupMock: func(m *MockSecretService) {
				m.On("Delete", mock.Anything, userID, secretID).Return(apperrors.ErrSecretAccessDenied)
			},
			expectedStatus: http.StatusForbidden,
			expectedError:  "access denied",
		},
		{
			name:     "internal server error",
			secretID: secretID.String(),
			setupMock: func(m *MockSecretService) {
				m.On("Delete", mock.Anything, userID, secretID).Return(assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAuth := new(MockAuthService)
			mockSecret := new(MockSecretService)
			tt.setupMock(mockSecret)

			router := setupAuthenticatedSecretRouter(mockAuth, mockSecret, userID)

			req, _ := http.NewRequest(http.MethodDelete, "/api/v1/secrets/"+tt.secretID, nil)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var resp ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.Contains(t, resp.Error, tt.expectedError)
			}

			mockSecret.AssertExpectations(t)
		})
	}
}

func TestHandler_SyncSecrets(t *testing.T) {
	userID := uuid.New()

	tests := []struct {
		name           string
		body           SyncRequest
		setupMock      func(*MockSecretService)
		expectedStatus int
		expectedError  string
	}{
		{
			name: "successful sync with empty list",
			body: SyncRequest{
				Secrets: []SyncSecretRequest{},
			},
			setupMock: func(m *MockSecretService) {
				result := &service.SyncResult{
					UpdatedSecrets: []*models.SecretData{},
					Conflicts:      []*service.Conflict{},
				}
				m.On("Sync", mock.Anything, userID, mock.Anything).Return(result, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "successful sync with secrets",
			body: SyncRequest{
				Secrets: []SyncSecretRequest{
					{
						ID:            uuid.New().String(),
						Type:          "login_password",
						Name:          "secret-1",
						EncryptedData: []byte("data-1"),
						Version:       1,
					},
				},
			},
			setupMock: func(m *MockSecretService) {
				result := &service.SyncResult{
					UpdatedSecrets: []*models.SecretData{
						{
							ID:            uuid.New(),
							UserID:        userID,
							Type:          models.TypeText,
							Name:          "server-secret",
							EncryptedData: []byte("server-data"),
							Version:       2,
							CreatedAt:     time.Now(),
							UpdatedAt:     time.Now(),
						},
					},
					Conflicts: []*service.Conflict{},
				}
				m.On("Sync", mock.Anything, userID, mock.Anything).Return(result, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "sync with conflicts",
			body: SyncRequest{
				Secrets: []SyncSecretRequest{
					{
						ID:            uuid.New().String(),
						Type:          "login_password",
						Name:          "conflict-secret",
						EncryptedData: []byte("client-data"),
						Version:       1,
					},
				},
			},
			setupMock: func(m *MockSecretService) {
				result := &service.SyncResult{
					UpdatedSecrets: []*models.SecretData{},
					Conflicts: []*service.Conflict{
						{
							ClientVersion: &models.SecretData{
								ID:            uuid.New(),
								Type:          models.TypeLoginPassword,
								Name:          "conflict-secret",
								EncryptedData: []byte("client-data"),
								Version:       1,
								CreatedAt:     time.Now(),
								UpdatedAt:     time.Now(),
							},
							ServerVersion: &models.SecretData{
								ID:            uuid.New(),
								Type:          models.TypeLoginPassword,
								Name:          "conflict-secret",
								EncryptedData: []byte("server-data"),
								Version:       1,
								CreatedAt:     time.Now(),
								UpdatedAt:     time.Now(),
							},
						},
					},
				}
				m.On("Sync", mock.Anything, userID, mock.Anything).Return(result, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "internal server error",
			body: SyncRequest{
				Secrets: []SyncSecretRequest{},
			},
			setupMock: func(m *MockSecretService) {
				m.On("Sync", mock.Anything, userID, mock.Anything).Return(nil, assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAuth := new(MockAuthService)
			mockSecret := new(MockSecretService)
			tt.setupMock(mockSecret)

			router := setupAuthenticatedSecretRouter(mockAuth, mockSecret, userID)

			body, _ := json.Marshal(tt.body)
			req, _ := http.NewRequest(http.MethodPost, "/api/v1/secrets/sync", bytes.NewBuffer(body))
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
				var resp SyncResponse
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				assert.NoError(t, err)
			}

			mockSecret.AssertExpectations(t)
		})
	}
}

