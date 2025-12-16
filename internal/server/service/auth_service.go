package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	apperrors "github.com/volchkovski/gophkeeper/internal/common/errors"
	"github.com/volchkovski/gophkeeper/internal/server/models"
	"github.com/volchkovski/gophkeeper/internal/server/repository"
	"github.com/volchkovski/gophkeeper/pkg/logger"
)

// JWTClaims extends jwt.RegisteredClaims with custom fields.
type JWTClaims struct {
	UserID   uuid.UUID `json:"user_id"`
	Username string    `json:"username"`
	jwt.RegisteredClaims
}

// AuthServiceConfig holds configuration for AuthService.
type AuthServiceConfig struct {
	JWTSecret       string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	BcryptCost      int
}

// authService implements AuthService interface.
type authService struct {
	userRepo repository.UserRepository
	config   AuthServiceConfig
	logger   *logger.Logger
}

// NewAuthService creates a new AuthService.
func NewAuthService(
	userRepo repository.UserRepository,
	config AuthServiceConfig,
	logger *logger.Logger,
) AuthService {
	return &authService{
		userRepo: userRepo,
		config:   config,
		logger:   logger,
	}
}

// Register creates a new user account.
func (s *authService) Register(ctx context.Context, username, password string) (*models.User, error) {
	// Validate input
	if err := s.validateRegistration(username, password); err != nil {
		return nil, err
	}

	// Check if user already exists
	_, err := s.userRepo.FindByUsername(ctx, username)
	if err == nil {
		return nil, apperrors.ErrUserAlreadyExists
	}
	if !errors.Is(err, apperrors.ErrUserNotFound) {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), s.config.BcryptCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	user := models.NewUser(username, string(hashedPassword))

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	s.logger.Info("user registered",
		logger.String("username", username),
		logger.String("user_id", user.ID.String()),
	)

	return user, nil
}

// Login authenticates a user and returns JWT tokens.
func (s *authService) Login(ctx context.Context, username, password string) (*TokenPair, error) {
	// Validate input
	if username == "" {
		return nil, apperrors.ErrEmptyUsername
	}
	if password == "" {
		return nil, apperrors.ErrEmptyPassword
	}

	// Find user
	user, err := s.userRepo.FindByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, apperrors.ErrUserNotFound) {
			return nil, apperrors.ErrInvalidCredentials
		}
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, apperrors.ErrInvalidCredentials
	}

	// Generate tokens
	tokenPair, err := s.generateTokenPair(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	s.logger.Info("user logged in",
		logger.String("username", username),
		logger.String("user_id", user.ID.String()),
	)

	return tokenPair, nil
}

// RefreshToken generates new tokens using a valid refresh token.
func (s *authService) RefreshToken(ctx context.Context, refreshToken string) (*TokenPair, error) {
	claims, err := s.parseToken(refreshToken)
	if err != nil {
		return nil, err
	}

	// Verify user still exists
	user, err := s.userRepo.FindByID(ctx, claims.UserID)
	if err != nil {
		if errors.Is(err, apperrors.ErrUserNotFound) {
			return nil, apperrors.ErrInvalidToken
		}
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	// Generate new tokens
	tokenPair, err := s.generateTokenPair(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	return tokenPair, nil
}

// ValidateToken validates an access token and returns claims.
func (s *authService) ValidateToken(token string) (*Claims, error) {
	jwtClaims, err := s.parseToken(token)
	if err != nil {
		return nil, err
	}

	return &Claims{
		UserID:   jwtClaims.UserID,
		Username: jwtClaims.Username,
	}, nil
}

// validateRegistration validates registration input.
func (s *authService) validateRegistration(username, password string) error {
	if username == "" {
		return apperrors.ErrEmptyUsername
	}
	if len(username) < 3 {
		return apperrors.ErrUsernameTooShort
	}
	if len(username) > 50 {
		return apperrors.ErrUsernameTooLong
	}
	if password == "" {
		return apperrors.ErrEmptyPassword
	}
	if len(password) < 8 {
		return apperrors.ErrPasswordTooShort
	}
	return nil
}

// generateTokenPair generates access and refresh tokens.
func (s *authService) generateTokenPair(user *models.User) (*TokenPair, error) {
	now := time.Now()

	// Access token
	accessClaims := JWTClaims{
		UserID:   user.ID,
		Username: user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(s.config.AccessTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "gophkeeper",
			Subject:   user.ID.String(),
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString([]byte(s.config.JWTSecret))
	if err != nil {
		return nil, fmt.Errorf("failed to sign access token: %w", err)
	}

	// Refresh token
	refreshClaims := JWTClaims{
		UserID:   user.ID,
		Username: user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(s.config.RefreshTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "gophkeeper",
			Subject:   user.ID.String(),
		},
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString([]byte(s.config.JWTSecret))
	if err != nil {
		return nil, fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
		ExpiresIn:    int64(s.config.AccessTokenTTL.Seconds()),
	}, nil
}

// parseToken parses and validates a JWT token.
func (s *authService) parseToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.config.JWTSecret), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, apperrors.ErrTokenExpired
		}
		return nil, apperrors.ErrInvalidToken
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, apperrors.ErrInvalidToken
	}

	return claims, nil
}

// Helper functions for logger
type logField = logger.Field

func String(key, value string) logField {
	return logger.String(key, value)
}

