package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/volchkovski/gophkeeper/pkg/logger"
)

func TestLoggerMiddleware(t *testing.T) {
	t.Run("logs successful request", func(t *testing.T) {
		gin.SetMode(gin.TestMode)

		log := logger.NewNop()
		router := gin.New()
		router.Use(LoggerMiddleware(log))
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		req, _ := http.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("logs request with query params", func(t *testing.T) {
		gin.SetMode(gin.TestMode)

		log := logger.NewNop()
		router := gin.New()
		router.Use(LoggerMiddleware(log))
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		req, _ := http.NewRequest(http.MethodGet, "/test?param=value", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("logs 4xx error", func(t *testing.T) {
		gin.SetMode(gin.TestMode)

		log := logger.NewNop()
		router := gin.New()
		router.Use(LoggerMiddleware(log))
		router.GET("/notfound", func(c *gin.Context) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		})

		req, _ := http.NewRequest(http.MethodGet, "/notfound", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("logs 5xx error", func(t *testing.T) {
		gin.SetMode(gin.TestMode)

		log := logger.NewNop()
		router := gin.New()
		router.Use(LoggerMiddleware(log))
		router.GET("/error", func(c *gin.Context) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
		})

		req, _ := http.NewRequest(http.MethodGet, "/error", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("logs request with user ID in context", func(t *testing.T) {
		gin.SetMode(gin.TestMode)

		log := logger.NewNop()
		router := gin.New()

		// Set user ID in context
		router.Use(func(c *gin.Context) {
			c.Set(UserIDKey, "test-user-id")
			c.Next()
		})
		router.Use(LoggerMiddleware(log))
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		req, _ := http.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("logs request with gin error", func(t *testing.T) {
		gin.SetMode(gin.TestMode)

		log := logger.NewNop()
		router := gin.New()
		router.Use(LoggerMiddleware(log))
		router.GET("/witherror", func(c *gin.Context) {
			c.Error(assert.AnError)
			c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
		})

		req, _ := http.NewRequest(http.MethodGet, "/witherror", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

