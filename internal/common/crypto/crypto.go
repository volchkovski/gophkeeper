// Package crypto provides encryption and decryption functionality for GophKeeper.
// It uses AES-256-GCM for symmetric encryption and PBKDF2/Argon2id for key derivation.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"io"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/pbkdf2"
)

const (
	// KeySize is the size of the AES-256 key in bytes.
	KeySize = 32
	// NonceSize is the size of the GCM nonce in bytes.
	NonceSize = 12
	// SaltSize is the size of the salt for key derivation.
	SaltSize = 32
	// PBKDF2Iterations is the number of iterations for PBKDF2.
	PBKDF2Iterations = 100000
)

// Argon2Params holds parameters for Argon2id key derivation.
type Argon2Params struct {
	Time    uint32
	Memory  uint32
	Threads uint8
	KeyLen  uint32
}

// DefaultArgon2Params returns recommended Argon2id parameters.
func DefaultArgon2Params() *Argon2Params {
	return &Argon2Params{
		Time:    1,
		Memory:  64 * 1024, // 64 MB
		Threads: 4,
		KeyLen:  KeySize,
	}
}

// Encryptor provides AES-256-GCM encryption and decryption.
type Encryptor struct {
	key []byte
}

// NewEncryptor creates a new Encryptor with the given key.
// The key must be exactly 32 bytes (256 bits) for AES-256.
func NewEncryptor(key []byte) (*Encryptor, error) {
	if len(key) != KeySize {
		return nil, errors.New("invalid key size: must be 32 bytes")
	}
	return &Encryptor{key: key}, nil
}

// Encrypt encrypts plaintext using AES-256-GCM.
// The returned ciphertext includes the nonce prepended to the encrypted data.
//
// Format: [nonce (12 bytes)][ciphertext][auth tag (16 bytes)]
func (e *Encryptor) Encrypt(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, NonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	// Seal appends the ciphertext to nonce
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt decrypts ciphertext using AES-256-GCM.
// It expects the nonce to be prepended to the ciphertext.
func (e *Encryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	if len(ciphertext) < NonceSize {
		return nil, errors.New("ciphertext too short")
	}

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := ciphertext[:NonceSize]
	ciphertext = ciphertext[NonceSize:]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// GenerateKey generates a cryptographically secure random key.
func GenerateKey() ([]byte, error) {
	key := make([]byte, KeySize)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, err
	}
	return key, nil
}

// GenerateSalt generates a cryptographically secure random salt.
func GenerateSalt() ([]byte, error) {
	salt := make([]byte, SaltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, err
	}
	return salt, nil
}

// DeriveKeyPBKDF2 derives a key from a password using PBKDF2-SHA256.
func DeriveKeyPBKDF2(password string, salt []byte) []byte {
	return pbkdf2.Key([]byte(password), salt, PBKDF2Iterations, KeySize, sha256.New)
}

// DeriveKeyArgon2 derives a key from a password using Argon2id.
// Argon2id is recommended for password hashing and key derivation.
func DeriveKeyArgon2(password string, salt []byte, params *Argon2Params) []byte {
	if params == nil {
		params = DefaultArgon2Params()
	}
	return argon2.IDKey([]byte(password), salt, params.Time, params.Memory, params.Threads, params.KeyLen)
}

// HashPassword creates a hash of the password for storage.
// This is different from key derivation - it's for verifying passwords.
func HashPassword(password string, salt []byte) []byte {
	return DeriveKeyArgon2(password, salt, DefaultArgon2Params())
}

// EncryptWithPassword encrypts data using a password.
// Returns: [salt (32 bytes)][nonce (12 bytes)][ciphertext][auth tag (16 bytes)]
func EncryptWithPassword(plaintext []byte, password string) ([]byte, error) {
	salt, err := GenerateSalt()
	if err != nil {
		return nil, err
	}

	key := DeriveKeyArgon2(password, salt, nil)
	encryptor, err := NewEncryptor(key)
	if err != nil {
		return nil, err
	}

	ciphertext, err := encryptor.Encrypt(plaintext)
	if err != nil {
		return nil, err
	}

	// Prepend salt to ciphertext
	result := make([]byte, SaltSize+len(ciphertext))
	copy(result[:SaltSize], salt)
	copy(result[SaltSize:], ciphertext)

	return result, nil
}

// DecryptWithPassword decrypts data that was encrypted with EncryptWithPassword.
func DecryptWithPassword(ciphertext []byte, password string) ([]byte, error) {
	if len(ciphertext) < SaltSize+NonceSize {
		return nil, errors.New("ciphertext too short")
	}

	salt := ciphertext[:SaltSize]
	encryptedData := ciphertext[SaltSize:]

	key := DeriveKeyArgon2(password, salt, nil)
	encryptor, err := NewEncryptor(key)
	if err != nil {
		return nil, err
	}

	return encryptor.Decrypt(encryptedData)
}

