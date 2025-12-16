package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestSecretType_IsValid(t *testing.T) {
	tests := []struct {
		name       string
		secretType SecretType
		want       bool
	}{
		{
			name:       "valid login_password",
			secretType: TypeLoginPassword,
			want:       true,
		},
		{
			name:       "valid text",
			secretType: TypeText,
			want:       true,
		},
		{
			name:       "valid binary",
			secretType: TypeBinary,
			want:       true,
		},
		{
			name:       "valid card",
			secretType: TypeCard,
			want:       true,
		},
		{
			name:       "invalid type",
			secretType: SecretType("invalid"),
			want:       false,
		},
		{
			name:       "empty type",
			secretType: SecretType(""),
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.secretType.IsValid()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNewSecretData(t *testing.T) {
	userID := uuid.New()
	secretType := TypeLoginPassword
	name := "test-secret"
	encryptedData := []byte("encrypted")

	secret := NewSecretData(userID, secretType, name, encryptedData)

	assert.NotNil(t, secret)
	assert.NotEqual(t, uuid.Nil, secret.ID)
	assert.Equal(t, userID, secret.UserID)
	assert.Equal(t, secretType, secret.Type)
	assert.Equal(t, name, secret.Name)
	assert.Equal(t, encryptedData, secret.EncryptedData)
	assert.Equal(t, int64(1), secret.Version)
	assert.False(t, secret.CreatedAt.IsZero())
	assert.False(t, secret.UpdatedAt.IsZero())
	assert.False(t, secret.DeletedAt.Valid)
}

func TestSecretData_IsDeleted(t *testing.T) {
	tests := []struct {
		name   string
		secret *SecretData
		want   bool
	}{
		{
			name: "not deleted",
			secret: &SecretData{
				ID: uuid.New(),
			},
			want: false,
		},
		{
			name: "deleted",
			secret: func() *SecretData {
				s := &SecretData{ID: uuid.New()}
				s.DeletedAt.Valid = true
				s.DeletedAt.Time = time.Now()
				return s
			}(),
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.secret.IsDeleted()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSecretData_IncrementVersion(t *testing.T) {
	secret := &SecretData{
		ID:        uuid.New(),
		Version:   1,
		UpdatedAt: time.Now().Add(-time.Hour),
	}

	oldUpdatedAt := secret.UpdatedAt
	secret.IncrementVersion()

	assert.Equal(t, int64(2), secret.Version)
	assert.True(t, secret.UpdatedAt.After(oldUpdatedAt))
}

func TestNewUser(t *testing.T) {
	username := "testuser"
	passwordHash := "hashedpassword"

	user := NewUser(username, passwordHash)

	assert.NotNil(t, user)
	assert.NotEqual(t, uuid.Nil, user.ID)
	assert.Equal(t, username, user.Username)
	assert.Equal(t, passwordHash, user.PasswordHash)
	assert.False(t, user.CreatedAt.IsZero())
	assert.False(t, user.UpdatedAt.IsZero())
}
