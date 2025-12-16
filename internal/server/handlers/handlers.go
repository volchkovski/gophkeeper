// Package handlers provides HTTP request handlers for GophKeeper server API.
package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/volchkovski/gophkeeper/internal/server/middleware"
	"github.com/volchkovski/gophkeeper/internal/server/service"
	"github.com/volchkovski/gophkeeper/pkg/logger"
)

// Handler holds all HTTP handlers.
type Handler struct {
	authService   service.AuthService
	secretService service.SecretService
	logger        *logger.Logger
}

// NewHandler creates a new Handler.
func NewHandler(
	authService service.AuthService,
	secretService service.SecretService,
	logger *logger.Logger,
) *Handler {
	return &Handler{
		authService:   authService,
		secretService: secretService,
		logger:        logger,
	}
}

// getUserID extracts user ID from context.
func getUserID(c *gin.Context) (uuid.UUID, error) {
	userIDValue, exists := c.Get(middleware.UserIDKey)
	if !exists {
		return uuid.Nil, ErrUnauthorized
	}

	userID, ok := userIDValue.(uuid.UUID)
	if !ok {
		return uuid.Nil, ErrUnauthorized
	}

	return userID, nil
}

// Error responses
var (
	ErrUnauthorized = &APIError{
		Status:  http.StatusUnauthorized,
		Message: "unauthorized",
	}
)

// APIError represents an API error.
type APIError struct {
	Status  int    `json:"-"`
	Message string `json:"error"`
}

func (e *APIError) Error() string {
	return e.Message
}

// ErrorResponse represents an error response.
type ErrorResponse struct {
	Error string `json:"error"`
}

// respondWithError sends an error response.
func respondWithError(c *gin.Context, status int, message string) {
	c.JSON(status, ErrorResponse{Error: message})
}

