package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/volchkovski/gophkeeper/pkg/logger"
)

// LoggerMiddleware creates a logging middleware.
func LoggerMiddleware(log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)

		// Get status code and error
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method
		errorMessage := c.Errors.ByType(gin.ErrorTypePrivate).String()

		// Log request
		if query != "" {
			path = path + "?" + query
		}

		fields := []logger.Field{
			logger.Int("status", statusCode),
			logger.String("method", method),
			logger.String("path", path),
			logger.String("ip", clientIP),
			logger.Duration("latency", latency),
		}

		if errorMessage != "" {
			fields = append(fields, logger.String("error", errorMessage))
		}

		if userID, exists := c.Get(UserIDKey); exists {
			fields = append(fields, logger.String("user_id", userID.(string)))
		}

		switch {
		case statusCode >= 500:
			log.Error("server error", fields...)
		case statusCode >= 400:
			log.Warn("client error", fields...)
		default:
			log.Info("request completed", fields...)
		}
	}
}

