package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/volchkovski/gophkeeper/pkg/logger"
)

// RecoveryMiddleware creates a panic recovery middleware.
func RecoveryMiddleware(log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Log stack trace
				log.Error("panic recovered",
					logger.Any("error", err),
					logger.String("stack", string(debug.Stack())),
					logger.String("path", c.Request.URL.Path),
					logger.String("method", c.Request.Method),
				)

				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": "internal server error",
				})
			}
		}()

		c.Next()
	}
}

