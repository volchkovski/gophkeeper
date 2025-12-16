package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/volchkovski/gophkeeper/pkg/logger"
)

func setupHealthTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)

	log := logger.NewNop()
	mockAuth := new(MockAuthService)
	mockSecret := new(MockSecretService)
	handler := NewHandler(mockAuth, mockSecret, log)

	router := gin.New()
	v1 := router.Group("/api/v1")
	{
		v1.GET("/health", handler.Health)
		v1.GET("/version", handler.GetVersion)
	}

	return router
}

func TestHandler_Health(t *testing.T) {
	router := setupHealthTestRouter()

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp HealthResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "ok", resp.Status)
}

func TestHandler_GetVersion(t *testing.T) {
	// Set version for test
	originalVersion := Version
	originalBuildDate := BuildDate
	Version = "1.0.0-test"
	BuildDate = "2024-01-01"
	defer func() {
		Version = originalVersion
		BuildDate = originalBuildDate
	}()

	router := setupHealthTestRouter()

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/version", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp VersionResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "1.0.0-test", resp.Version)
	assert.Equal(t, "2024-01-01", resp.BuildDate)
}

