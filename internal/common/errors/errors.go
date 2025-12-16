// Package errors provides common error definitions for GophKeeper.
package errors

import "errors"

// Authentication errors.
var (
	ErrUserNotFound       = errors.New("user not found")
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid token")
	ErrTokenExpired       = errors.New("token expired")
)

// Secret errors.
var (
	ErrSecretNotFound     = errors.New("secret not found")
	ErrSecretAccessDenied = errors.New("access denied to secret")
	ErrInvalidSecretType  = errors.New("invalid secret type")
	ErrSecretNameExists   = errors.New("secret with this name already exists")
)

// Validation errors.
var (
	ErrValidation        = errors.New("validation error")
	ErrInvalidInput      = errors.New("invalid input")
	ErrEmptyUsername     = errors.New("username cannot be empty")
	ErrEmptyPassword     = errors.New("password cannot be empty")
	ErrPasswordTooShort  = errors.New("password must be at least 8 characters")
	ErrUsernameTooShort  = errors.New("username must be at least 3 characters")
	ErrUsernameTooLong   = errors.New("username must be at most 50 characters")
	ErrInvalidSecretName = errors.New("invalid secret name")
)

// Sync errors.
var (
	ErrSyncConflict = errors.New("sync conflict detected")
	ErrSyncFailed   = errors.New("sync failed")
)

// Crypto errors.
var (
	ErrEncryptionFailed = errors.New("encryption failed")
	ErrDecryptionFailed = errors.New("decryption failed")
	ErrInvalidKey       = errors.New("invalid encryption key")
)

// Storage errors.
var (
	ErrStorageRead  = errors.New("failed to read from storage")
	ErrStorageWrite = errors.New("failed to write to storage")
)

