// Package middleware provides HTTP middleware for GophKeeper server.
package middleware

import (
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

// AuthMiddleware creates authentication middleware.
func AuthMiddleware(authService service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader(AuthorizationHeader)
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "authorization header is required",
			})
			return
		}

		if !strings.HasPrefix(authHeader, BearerPrefix) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid authorization header format",
			})
			return
		}

		token := strings.TrimPrefix(authHeader, BearerPrefix)
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "token is required",
			})
			return
		}

		claims, err := authService.ValidateToken(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid or expired token",
			})
			return
		}

		// Set user info in context
		c.Set(UserIDKey, claims.UserID)
		c.Set(UsernameKey, claims.Username)

		c.Next()
	}
}

