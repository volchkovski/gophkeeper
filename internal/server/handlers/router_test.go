package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/volchkovski/gophkeeper/pkg/logger"
)

func TestSetupRouter(t *testing.T) {
	t.Run("router without rate limiting", func(t *testing.T) {
		mockAuth := new(MockAuthService)
		mockSecret := new(MockSecretService)
		log := logger.NewNop()
		handler := NewHandler(mockAuth, mockSecret, log)

		config := RouterConfig{
			RateLimit: 0,
		}

		router := SetupRouter(handler, mockAuth, log, config)

		assert.NotNil(t, router)

		// Test health endpoint
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("router with rate limiting", func(t *testing.T) {
		mockAuth := new(MockAuthService)
		mockSecret := new(MockSecretService)
		log := logger.NewNop()
		handler := NewHandler(mockAuth, mockSecret, log)

		config := RouterConfig{
			RateLimit: 100,
		}

		router := SetupRouter(handler, mockAuth, log, config)

		assert.NotNil(t, router)

		// Test version endpoint
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/version", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("protected routes require authentication", func(t *testing.T) {
		mockAuth := new(MockAuthService)
		mockSecret := new(MockSecretService)
		log := logger.NewNop()
		handler := NewHandler(mockAuth, mockSecret, log)

		config := RouterConfig{
			RateLimit: 0,
		}

		router := SetupRouter(handler, mockAuth, log, config)

		// Test protected endpoint without auth
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/secrets", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

