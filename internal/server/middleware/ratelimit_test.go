package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"golang.org/x/time/rate"
)

func TestNewRateLimiter(t *testing.T) {
	limiter := NewRateLimiter(rate.Limit(10), 5)

	assert.NotNil(t, limiter)
	assert.NotNil(t, limiter.limiters)
	assert.Equal(t, rate.Limit(10), limiter.rate)
	assert.Equal(t, 5, limiter.burst)
}

func TestRateLimiter_GetLimiter(t *testing.T) {
	rl := NewRateLimiter(rate.Limit(10), 5)

	// Get limiter for new IP
	limiter1 := rl.GetLimiter("192.168.1.1")
	assert.NotNil(t, limiter1)

	// Get limiter for same IP - should return same instance
	limiter2 := rl.GetLimiter("192.168.1.1")
	// Compare pointers directly
	assert.True(t, limiter1 == limiter2, "Expected same limiter instance for same IP")

	// Get limiter for different IP - should return different instance
	limiter3 := rl.GetLimiter("192.168.1.2")
	assert.NotNil(t, limiter3)
	// Different IPs should have different limiter instances
	assert.True(t, limiter1 != limiter3, "Expected different limiter instance for different IP")
}

func TestRateLimitMiddleware(t *testing.T) {
	t.Run("allows requests within limit", func(t *testing.T) {
		gin.SetMode(gin.TestMode)

		// Create limiter with high rate to allow requests
		rl := NewRateLimiter(rate.Limit(100), 10)
		router := gin.New()
		router.Use(RateLimitMiddleware(rl))
		router.GET("/test", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		req, _ := http.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("blocks requests over limit", func(t *testing.T) {
		gin.SetMode(gin.TestMode)

		// Create limiter with very low rate
		rl := NewRateLimiter(rate.Limit(0.001), 1)
		router := gin.New()
		router.Use(RateLimitMiddleware(rl))
		router.GET("/test", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		// First request should succeed (uses burst)
		req1, _ := http.NewRequest(http.MethodGet, "/test", nil)
		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, req1)
		assert.Equal(t, http.StatusOK, w1.Code)

		// Second request should be rate limited
		req2, _ := http.NewRequest(http.MethodGet, "/test", nil)
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)
		assert.Equal(t, http.StatusTooManyRequests, w2.Code)
		assert.Contains(t, w2.Body.String(), "rate limit exceeded")
	})

	t.Run("different IPs have separate limits", func(t *testing.T) {
		// This test verifies that different IPs get different rate limiters
		// by testing the GetLimiter functionality directly
		rl := NewRateLimiter(rate.Limit(1), 1)

		// Get limiters for two different IPs
		limiter1 := rl.GetLimiter("10.0.0.1")
		limiter2 := rl.GetLimiter("10.0.0.2")

		// They should be different instances
		assert.True(t, limiter1 != limiter2)

		// Use first limiter's token
		assert.True(t, limiter1.Allow())
		// First limiter should now be exhausted
		assert.False(t, limiter1.Allow())
		// Second limiter should still have token
		assert.True(t, limiter2.Allow())
	})
}

