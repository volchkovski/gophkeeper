package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestDefaultCORSConfig(t *testing.T) {
	config := DefaultCORSConfig()

	assert.Equal(t, []string{"*"}, config.AllowOrigins)
	assert.Contains(t, config.AllowMethods, "GET")
	assert.Contains(t, config.AllowMethods, "POST")
	assert.Contains(t, config.AllowMethods, "PUT")
	assert.Contains(t, config.AllowMethods, "DELETE")
	assert.Contains(t, config.AllowHeaders, "Authorization")
	assert.Contains(t, config.AllowHeaders, "Content-Type")
	assert.True(t, config.AllowCredentials)
	assert.Equal(t, 86400, config.MaxAge)
}

func TestCORSMiddleware(t *testing.T) {
	tests := []struct {
		name            string
		config          CORSConfig
		requestMethod   string
		requestOrigin   string
		expectedStatus  int
		expectedHeaders map[string]string
	}{
		{
			name:           "allow all origins with wildcard",
			config:         DefaultCORSConfig(),
			requestMethod:  "GET",
			requestOrigin:  "http://example.com",
			expectedStatus: http.StatusOK,
			expectedHeaders: map[string]string{
				"Access-Control-Allow-Origin":      "*",
				"Access-Control-Allow-Credentials": "true",
			},
		},
		{
			name: "specific origin allowed",
			config: CORSConfig{
				AllowOrigins:     []string{"http://allowed.com"},
				AllowMethods:     []string{"GET", "POST"},
				AllowHeaders:     []string{"Content-Type"},
				AllowCredentials: true,
			},
			requestMethod:  "GET",
			requestOrigin:  "http://allowed.com",
			expectedStatus: http.StatusOK,
			expectedHeaders: map[string]string{
				"Access-Control-Allow-Origin":      "http://allowed.com",
				"Access-Control-Allow-Credentials": "true",
			},
		},
		{
			name: "origin not in allowed list",
			config: CORSConfig{
				AllowOrigins:     []string{"http://allowed.com"},
				AllowMethods:     []string{"GET", "POST"},
				AllowHeaders:     []string{"Content-Type"},
				AllowCredentials: false,
			},
			requestMethod:  "GET",
			requestOrigin:  "http://notallowed.com",
			expectedStatus: http.StatusOK,
			expectedHeaders: map[string]string{
				"Access-Control-Allow-Origin": "*",
			},
		},
		{
			name:           "preflight OPTIONS request",
			config:         DefaultCORSConfig(),
			requestMethod:  "OPTIONS",
			requestOrigin:  "http://example.com",
			expectedStatus: http.StatusNoContent,
			expectedHeaders: map[string]string{
				"Access-Control-Allow-Origin":      "*",
				"Access-Control-Allow-Credentials": "true",
			},
		},
		{
			name: "credentials disabled",
			config: CORSConfig{
				AllowOrigins:     []string{"*"},
				AllowMethods:     []string{"GET"},
				AllowHeaders:     []string{"Content-Type"},
				AllowCredentials: false,
			},
			requestMethod:  "GET",
			requestOrigin:  "http://example.com",
			expectedStatus: http.StatusOK,
			expectedHeaders: map[string]string{
				"Access-Control-Allow-Origin": "*",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			router := gin.New()
			router.Use(CORSMiddleware(tt.config))
			router.GET("/test", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})
			router.OPTIONS("/test", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req, _ := http.NewRequest(tt.requestMethod, "/test", nil)
			if tt.requestOrigin != "" {
				req.Header.Set("Origin", tt.requestOrigin)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			for header, value := range tt.expectedHeaders {
				assert.Equal(t, value, w.Header().Get(header), "header %s", header)
			}
		})
	}
}

func TestJoinStrings(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		sep      string
		expected string
	}{
		{
			name:     "empty slice",
			input:    []string{},
			sep:      ", ",
			expected: "",
		},
		{
			name:     "single element",
			input:    []string{"one"},
			sep:      ", ",
			expected: "one",
		},
		{
			name:     "multiple elements",
			input:    []string{"one", "two", "three"},
			sep:      ", ",
			expected: "one, two, three",
		},
		{
			name:     "different separator",
			input:    []string{"a", "b"},
			sep:      "-",
			expected: "a-b",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := joinStrings(tt.input, tt.sep)
			assert.Equal(t, tt.expected, result)
		})
	}
}

