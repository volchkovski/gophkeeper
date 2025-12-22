package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/volchkovski/gophkeeper/internal/server/middleware"
	"github.com/volchkovski/gophkeeper/internal/server/service"
	"github.com/volchkovski/gophkeeper/pkg/logger"
	"golang.org/x/time/rate"
)

// RouterConfig holds router configuration.
type RouterConfig struct {
	RateLimit int
}

// SetupRouter configures the HTTP router.
func SetupRouter(
	handler *Handler,
	authService service.AuthService,
	log *logger.Logger,
	config RouterConfig,
) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()

	// Global middleware
	router.Use(middleware.RecoveryMiddleware(log))
	router.Use(middleware.LoggerMiddleware(log))
	router.Use(middleware.CORSMiddleware(middleware.DefaultCORSConfig()))

	// Rate limiter
	if config.RateLimit > 0 {
		limiter := middleware.NewRateLimiter(rate.Limit(config.RateLimit), config.RateLimit)
		router.Use(middleware.RateLimitMiddleware(limiter))
	}

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Public routes
		v1.GET("/health", handler.Health)
		v1.GET("/version", handler.GetVersion)

		// Auth routes (public)
		auth := v1.Group("/auth")
		{
			auth.POST("/register", handler.Register)
			auth.POST("/login", handler.Login)
			auth.POST("/refresh", handler.RefreshToken)
		}

		// Protected routes
		protected := v1.Group("")
		protected.Use(middleware.AuthMiddleware(authService))
		{
			// Secrets routes
			secrets := protected.Group("/secrets")
			{
				secrets.GET("", handler.ListSecrets)
				secrets.POST("", handler.CreateSecret)
				secrets.GET("/:id", handler.GetSecret)
				secrets.PUT("/:id", handler.UpdateSecret)
				secrets.DELETE("/:id", handler.DeleteSecret)
				secrets.POST("/sync", handler.SyncSecrets)
			}
		}
	}

	return router
}

