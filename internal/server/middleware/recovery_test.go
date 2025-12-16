package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/volchkovski/gophkeeper/pkg/logger"
)

func TestRecoveryMiddleware(t *testing.T) {
	t.Run("recovers from panic", func(t *testing.T) {
		gin.SetMode(gin.TestMode)

		log := logger.NewNop()
		router := gin.New()
		router.Use(RecoveryMiddleware(log))
		router.GET("/panic", func(c *gin.Context) {
			panic("test panic")
		})

		req, _ := http.NewRequest(http.MethodGet, "/panic", nil)
		w := httptest.NewRecorder()

		// Should not panic
		assert.NotPanics(t, func() {
			router.ServeHTTP(w, req)
		})

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "internal server error")
	})

	t.Run("normal request without panic", func(t *testing.T) {
		gin.SetMode(gin.TestMode)

		log := logger.NewNop()
		router := gin.New()
		router.Use(RecoveryMiddleware(log))
		router.GET("/normal", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		req, _ := http.NewRequest(http.MethodGet, "/normal", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "ok")
	})

	t.Run("recovers from panic with error value", func(t *testing.T) {
		gin.SetMode(gin.TestMode)

		log := logger.NewNop()
		router := gin.New()
		router.Use(RecoveryMiddleware(log))
		router.GET("/panic-error", func(c *gin.Context) {
			panic(assert.AnError)
		})

		req, _ := http.NewRequest(http.MethodGet, "/panic-error", nil)
		w := httptest.NewRecorder()

		assert.NotPanics(t, func() {
			router.ServeHTTP(w, req)
		})

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

