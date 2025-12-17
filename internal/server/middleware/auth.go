// Package middleware provides HTTP middleware for GophKeeper server.
package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/volchkovski/gophkeeper/internal/server/service"
)

const (
	// AuthorizationHeader is the name of the authorization header.
	AuthorizationHeader = "Authorization"
	// BearerPrefix is the prefix for bearer tokens.
	BearerPrefix = "Bearer "
	// UserIDKey is the context key for the user ID.
	UserIDKey = "userID"
	// UsernameKey is the context key for the username.
	UsernameKey = "username"
)

// Common auth errors for composable middleware.
var (
	ErrMissingAuthHeader = errors.New("authorization header is required")
	ErrInvalidAuthFormat = errors.New("invalid authorization header format")
	ErrEmptyToken        = errors.New("token is required")
	ErrInvalidToken      = errors.New("invalid or expired token")
)

// TokenExtractor extracts a token from the request.
// Returns the token string and any error encountered.
type TokenExtractor func(c *gin.Context) (string, error)

// TokenValidator validates a token and returns claims.
// Returns claims and any error encountered.
type TokenValidator func(token string) (*service.Claims, error)

// ContextSetter sets claims data into the gin context.
type ContextSetter func(c *gin.Context, claims *service.Claims)

// ExtractBearerToken extracts a Bearer token from the Authorization header.
// This is a composable function that can be reused independently.
func ExtractBearerToken(c *gin.Context) (string, error) {
	authHeader := c.GetHeader(AuthorizationHeader)
	if authHeader == "" {
		return "", ErrMissingAuthHeader
	}

	if !strings.HasPrefix(authHeader, BearerPrefix) {
		return "", ErrInvalidAuthFormat
	}

	token := strings.TrimPrefix(authHeader, BearerPrefix)
	if token == "" {
		return "", ErrEmptyToken
	}

	return token, nil
}

// SetUserContext sets user claims into the gin context.
// This is a composable function that can be reused independently.
func SetUserContext(c *gin.Context, claims *service.Claims) {
	c.Set(UserIDKey, claims.UserID)
	c.Set(UsernameKey, claims.Username)
}

// AuthMiddlewareConfig holds configuration for composable auth middleware.
type AuthMiddlewareConfig struct {
	TokenExtractor TokenExtractor
	TokenValidator TokenValidator
	ContextSetter  ContextSetter
}

// DefaultAuthMiddlewareConfig returns default configuration using the auth service.
func DefaultAuthMiddlewareConfig(authService service.AuthService) AuthMiddlewareConfig {
	return AuthMiddlewareConfig{
		TokenExtractor: ExtractBearerToken,
		TokenValidator: func(token string) (*service.Claims, error) {
			claims, err := authService.ValidateToken(token)
			if err != nil {
				return nil, ErrInvalidToken
			}
			return claims, nil
		},
		ContextSetter: SetUserContext,
	}
}

// AuthMiddlewareWithConfig creates authentication middleware with custom configuration.
// This allows for maximum composability and testability.
func AuthMiddlewareWithConfig(cfg AuthMiddlewareConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract token
		token, err := cfg.TokenExtractor(c)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": err.Error(),
			})
			return
		}

		// Validate token
		claims, err := cfg.TokenValidator(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": err.Error(),
			})
			return
		}

		// Set context
		cfg.ContextSetter(c, claims)

		c.Next()
	}
}

// AuthMiddleware creates authentication middleware.
// This is a convenience wrapper that uses default configuration.
func AuthMiddleware(authService service.AuthService) gin.HandlerFunc {
	return AuthMiddlewareWithConfig(DefaultAuthMiddlewareConfig(authService))
}

