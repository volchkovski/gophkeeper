package models

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// SecretType represents the type of secret data.
type SecretType string

// Secret type constants.
const (
	TypeLoginPassword SecretType = "login_password"
	TypeText          SecretType = "text"
	TypeBinary        SecretType = "binary"
	TypeCard          SecretType = "card"
)

// IsValid checks if the secret type is valid.
func (t SecretType) IsValid() bool {
	switch t {
	case TypeLoginPassword, TypeText, TypeBinary, TypeCard:
		return true
	}
	return false
}

// SecretData represents encrypted secret data stored on the server.
// The actual content is encrypted client-side and the server never has access
// to the decryption key (zero-knowledge architecture).
type SecretData struct {
	ID            uuid.UUID    `json:"id" db:"id"`
	UserID        uuid.UUID    `json:"user_id" db:"user_id"`
	Type          SecretType   `json:"type" db:"type"`
	Name          string       `json:"name" db:"name"`
	EncryptedData []byte       `json:"encrypted_data" db:"encrypted_data"`
	Metadata      string       `json:"metadata,omitempty" db:"metadata"`
	Version       int64        `json:"version" db:"version"`
	CreatedAt     time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time    `json:"updated_at" db:"updated_at"`
	DeletedAt     sql.NullTime `json:"deleted_at,omitempty" db:"deleted_at"`
}

// NewSecretData creates a new SecretData instance with a generated UUID.
func NewSecretData(userID uuid.UUID, secretType SecretType, name string, encryptedData []byte) *SecretData {
	now := time.Now()
	return &SecretData{
		ID:            uuid.New(),
		UserID:        userID,
		Type:          secretType,
		Name:          name,
		EncryptedData: encryptedData,
		Version:       1,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

// IsDeleted returns true if the secret has been soft-deleted.
func (s *SecretData) IsDeleted() bool {
	return s.DeletedAt.Valid
}

// IncrementVersion increments the version number and updates the timestamp.
func (s *SecretData) IncrementVersion() {
	s.Version++
	s.UpdatedAt = time.Now()
}

