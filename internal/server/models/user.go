// Package models defines data structures for the GophKeeper server.
package models

import (
	"time"

	"github.com/google/uuid"
)

// User represents a registered user in the system.
type User struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	Username     string     `json:"username" db:"username"`
	PasswordHash string     `json:"-" db:"password_hash"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
}

// NewUser creates a new User instance with a generated UUID.
func NewUser(username, passwordHash string) *User {
	now := time.Now()
	return &User{
		ID:           uuid.New(),
		Username:     username,
		PasswordHash: passwordHash,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

