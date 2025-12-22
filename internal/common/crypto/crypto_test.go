package crypto

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncryptor_EncryptDecrypt(t *testing.T) {
	key, err := GenerateKey()
	require.NoError(t, err)
	require.Len(t, key, KeySize)

	encryptor, err := NewEncryptor(key)
	require.NoError(t, err)

	tests := []struct {
		name      string
		plaintext []byte
	}{
		{
			name:      "single byte",
			plaintext: []byte{0},
		},
		{
			name:      "short text",
			plaintext: []byte("Hello, World!"),
		},
		{
			name:      "long text",
			plaintext: bytes.Repeat([]byte("A"), 10000),
		},
		{
			name:      "binary data",
			plaintext: []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD},
		},
		{
			name:      "json data",
			plaintext: []byte(`{"login":"user@example.com","password":"secret123"}`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ciphertext, err := encryptor.Encrypt(tt.plaintext)
			require.NoError(t, err)
			assert.NotEqual(t, tt.plaintext, ciphertext)

			decrypted, err := encryptor.Decrypt(ciphertext)
			require.NoError(t, err)
			assert.Equal(t, tt.plaintext, decrypted)
		})
	}
}

func TestEncryptor_InvalidKey(t *testing.T) {
	tests := []struct {
		name    string
		keySize int
	}{
		{"too short", 16},
		{"too long", 64},
		{"empty", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := make([]byte, tt.keySize)
			_, err := NewEncryptor(key)
			assert.Error(t, err)
		})
	}
}

func TestEncryptor_DecryptInvalidCiphertext(t *testing.T) {
	key, err := GenerateKey()
	require.NoError(t, err)

	encryptor, err := NewEncryptor(key)
	require.NoError(t, err)

	tests := []struct {
		name       string
		ciphertext []byte
	}{
		{
			name:       "too short",
			ciphertext: []byte{0x01, 0x02, 0x03},
		},
		{
			name:       "invalid data",
			ciphertext: bytes.Repeat([]byte{0x00}, 100),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := encryptor.Decrypt(tt.ciphertext)
			assert.Error(t, err)
		})
	}
}

func TestEncryptor_DifferentNonces(t *testing.T) {
	key, err := GenerateKey()
	require.NoError(t, err)

	encryptor, err := NewEncryptor(key)
	require.NoError(t, err)

	plaintext := []byte("same plaintext")

	// Encrypt the same plaintext twice
	ciphertext1, err := encryptor.Encrypt(plaintext)
	require.NoError(t, err)

	ciphertext2, err := encryptor.Encrypt(plaintext)
	require.NoError(t, err)

	// Ciphertexts should be different due to different nonces
	assert.NotEqual(t, ciphertext1, ciphertext2)

	// But both should decrypt to the same plaintext
	decrypted1, err := encryptor.Decrypt(ciphertext1)
	require.NoError(t, err)

	decrypted2, err := encryptor.Decrypt(ciphertext2)
	require.NoError(t, err)

	assert.Equal(t, plaintext, decrypted1)
	assert.Equal(t, plaintext, decrypted2)
}

func TestDeriveKeyPBKDF2(t *testing.T) {
	password := "mysecretpassword"
	salt, err := GenerateSalt()
	require.NoError(t, err)

	key1 := DeriveKeyPBKDF2(password, salt)
	key2 := DeriveKeyPBKDF2(password, salt)

	assert.Equal(t, key1, key2, "Same password and salt should produce same key")
	assert.Len(t, key1, KeySize)

	// Different salt should produce different key
	salt2, err := GenerateSalt()
	require.NoError(t, err)
	key3 := DeriveKeyPBKDF2(password, salt2)
	assert.NotEqual(t, key1, key3)
}

func TestDeriveKeyArgon2(t *testing.T) {
	password := "mysecretpassword"
	salt, err := GenerateSalt()
	require.NoError(t, err)

	key1 := DeriveKeyArgon2(password, salt, nil)
	key2 := DeriveKeyArgon2(password, salt, nil)

	assert.Equal(t, key1, key2, "Same password and salt should produce same key")
	assert.Len(t, key1, KeySize)

	// Different password should produce different key
	key3 := DeriveKeyArgon2("differentpassword", salt, nil)
	assert.NotEqual(t, key1, key3)
}

func TestEncryptDecryptWithPassword(t *testing.T) {
	plaintext := []byte("secret data to protect")
	password := "myStrongPassword123!"

	ciphertext, err := EncryptWithPassword(plaintext, password)
	require.NoError(t, err)
	assert.NotEqual(t, plaintext, ciphertext)

	decrypted, err := DecryptWithPassword(ciphertext, password)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)

	// Wrong password should fail
	_, err = DecryptWithPassword(ciphertext, "wrongpassword")
	assert.Error(t, err)
}

func TestGenerateKey(t *testing.T) {
	key1, err := GenerateKey()
	require.NoError(t, err)
	assert.Len(t, key1, KeySize)

	key2, err := GenerateKey()
	require.NoError(t, err)

	// Two generated keys should be different
	assert.NotEqual(t, key1, key2)
}

func TestGenerateSalt(t *testing.T) {
	salt1, err := GenerateSalt()
	require.NoError(t, err)
	assert.Len(t, salt1, SaltSize)

	salt2, err := GenerateSalt()
	require.NoError(t, err)

	// Two generated salts should be different
	assert.NotEqual(t, salt1, salt2)
}

func BenchmarkEncrypt(b *testing.B) {
	key, _ := GenerateKey()
	encryptor, _ := NewEncryptor(key)
	plaintext := bytes.Repeat([]byte("A"), 1024) // 1KB

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = encryptor.Encrypt(plaintext)
	}
}

func BenchmarkDecrypt(b *testing.B) {
	key, _ := GenerateKey()
	encryptor, _ := NewEncryptor(key)
	plaintext := bytes.Repeat([]byte("A"), 1024) // 1KB
	ciphertext, _ := encryptor.Encrypt(plaintext)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = encryptor.Decrypt(ciphertext)
	}
}

func BenchmarkDeriveKeyArgon2(b *testing.B) {
	password := "mypassword"
	salt, _ := GenerateSalt()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DeriveKeyArgon2(password, salt, nil)
	}
}

