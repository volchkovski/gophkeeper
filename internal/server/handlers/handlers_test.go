package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/volchkovski/gophkeeper/internal/server/middleware"
	"github.com/volchkovski/gophkeeper/pkg/logger"
)

func TestNewHandler(t *testing.T) {
	log := logger.NewNop()
	mockAuth := new(MockAuthService)
	mockSecret := new(MockSecretService)

	handler := NewHandler(mockAuth, mockSecret, log)

	assert.NotNil(t, handler)
}

func TestGetUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("valid user ID in context", func(t *testing.T) {
		userID := uuid.New()

		router := gin.New()
		router.GET("/test", func(c *gin.Context) {
			c.Set(middleware.UserIDKey, userID)

			retrievedID, err := getUserID(c)
			assert.NoError(t, err)
			assert.Equal(t, userID, retrievedID)

			c.Status(http.StatusOK)
		})

		req, _ := http.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("missing user ID in context", func(t *testing.T) {
		router := gin.New()
		router.GET("/test", func(c *gin.Context) {
			retrievedID, err := getUserID(c)
			assert.Error(t, err)
			assert.Equal(t, uuid.Nil, retrievedID)
			c.Status(http.StatusOK)
		})

		req, _ := http.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	})

	t.Run("wrong type in context", func(t *testing.T) {
		router := gin.New()
		router.GET("/test", func(c *gin.Context) {
			c.Set(middleware.UserIDKey, "not-a-uuid")

			retrievedID, err := getUserID(c)
			assert.Error(t, err)
			assert.Equal(t, uuid.Nil, retrievedID)
			c.Status(http.StatusOK)
		})

		req, _ := http.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	})
}

func TestAPIError_Error(t *testing.T) {
	err := &APIError{
		Status:  http.StatusBadRequest,
		Message: "test error",
	}

	assert.Equal(t, "test error", err.Error())
}

func TestRespondWithError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		respondWithError(c, http.StatusBadRequest, "test error message")
	})

	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "test error message")
}

