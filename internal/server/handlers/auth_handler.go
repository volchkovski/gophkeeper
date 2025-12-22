package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	apperrors "github.com/volchkovski/gophkeeper/internal/common/errors"
	"github.com/volchkovski/gophkeeper/pkg/logger"
)

// RegisterRequest represents a registration request.
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Password string `json:"password" binding:"required,min=8"`
}

// RegisterResponse represents a registration response.
type RegisterResponse struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
}

// LoginRequest represents a login request.
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse represents a login response.
type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

// RefreshRequest represents a token refresh request.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// Register handles user registration.
// @Summary Register new user
// @Tags auth
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "Registration data"
// @Success 201 {object} RegisterResponse
// @Failure 400 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Router /auth/register [post]
func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondWithError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := h.authService.Register(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, apperrors.ErrUserAlreadyExists):
			respondWithError(c, http.StatusConflict, "user already exists")
		case errors.Is(err, apperrors.ErrEmptyUsername),
			errors.Is(err, apperrors.ErrUsernameTooShort),
			errors.Is(err, apperrors.ErrUsernameTooLong),
			errors.Is(err, apperrors.ErrEmptyPassword),
			errors.Is(err, apperrors.ErrPasswordTooShort):
			respondWithError(c, http.StatusBadRequest, err.Error())
		default:
			h.logger.Error("registration failed", logger.Error(err))
			respondWithError(c, http.StatusInternalServerError, "registration failed")
		}
		return
	}

	c.JSON(http.StatusCreated, RegisterResponse{
		UserID:   user.ID.String(),
		Username: user.Username,
	})
}

// Login handles user authentication.
// @Summary Login user
// @Tags auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Login credentials"
// @Success 200 {object} LoginResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /auth/login [post]
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondWithError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	tokenPair, err := h.authService.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, apperrors.ErrInvalidCredentials),
			errors.Is(err, apperrors.ErrUserNotFound):
			respondWithError(c, http.StatusUnauthorized, "invalid credentials")
		case errors.Is(err, apperrors.ErrEmptyUsername),
			errors.Is(err, apperrors.ErrEmptyPassword):
			respondWithError(c, http.StatusBadRequest, err.Error())
		default:
			h.logger.Error("login failed", logger.Error(err))
			respondWithError(c, http.StatusInternalServerError, "login failed")
		}
		return
	}

	c.JSON(http.StatusOK, LoginResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    tokenPair.ExpiresIn,
	})
}

// RefreshToken handles token refresh.
// @Summary Refresh access token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body RefreshRequest true "Refresh token"
// @Success 200 {object} LoginResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /auth/refresh [post]
func (h *Handler) RefreshToken(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondWithError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	tokenPair, err := h.authService.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		switch {
		case errors.Is(err, apperrors.ErrInvalidToken),
			errors.Is(err, apperrors.ErrTokenExpired):
			respondWithError(c, http.StatusUnauthorized, "invalid or expired token")
		default:
			h.logger.Error("token refresh failed", logger.Error(err))
			respondWithError(c, http.StatusInternalServerError, "token refresh failed")
		}
		return
	}

	c.JSON(http.StatusOK, LoginResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    tokenPair.ExpiresIn,
	})
}

