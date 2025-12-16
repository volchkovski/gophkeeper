package storage

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/volchkovski/gophkeeper/internal/common/crypto"
)

// TestStorage wraps Storage for testing with in-memory database.
type TestStorage struct {
	*Storage
	tmpDir string
}

// NewTestStorage creates a new Storage for testing.
func NewTestStorage(t *testing.T) *TestStorage {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "gophkeeper-storage-test")
	require.NoError(t, err)

	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := sql.Open("sqlite3", dbPath)
	require.NoError(t, err)

	s := &Storage{
		db:           db,
		tokenFile:    filepath.Join(tmpDir, "token"),
		keychainFile: filepath.Join(tmpDir, "keychain"),
	}

	err = s.initDB()
	require.NoError(t, err)

	return &TestStorage{
		Storage: s,
		tmpDir:  tmpDir,
	}
}

// Cleanup removes temporary files.
func (ts *TestStorage) Cleanup() {
	ts.Close()
	os.RemoveAll(ts.tmpDir)
}

func TestStorage_Token(t *testing.T) {
	ts := NewTestStorage(t)
	defer ts.Cleanup()

	t.Run("save and get token", func(t *testing.T) {
		err := ts.SaveToken("test-token-123")
		require.NoError(t, err)

		token, err := ts.GetToken()
		require.NoError(t, err)
		assert.Equal(t, "test-token-123", token)
	})

	t.Run("get non-existent token returns empty string", func(t *testing.T) {
		ts2 := NewTestStorage(t)
		defer ts2.Cleanup()

		token, err := ts2.GetToken()
		require.NoError(t, err)
		assert.Empty(t, token)
	})

	t.Run("delete token", func(t *testing.T) {
		err := ts.SaveToken("to-be-deleted")
		require.NoError(t, err)

		err = ts.DeleteToken()
		require.NoError(t, err)

		token, err := ts.GetToken()
		require.NoError(t, err)
		assert.Empty(t, token)
	})

	t.Run("delete non-existent token does not error", func(t *testing.T) {
		ts2 := NewTestStorage(t)
		defer ts2.Cleanup()

		err := ts2.DeleteToken()
		require.NoError(t, err)
	})
}

func TestStorage_MasterKey(t *testing.T) {
	ts := NewTestStorage(t)
	defer ts.Cleanup()

	t.Run("save and get master key", func(t *testing.T) {
		masterKey := []byte("0123456789abcdef0123456789abcdef") // 32 bytes
		password := "testpassword123"

		err := ts.SaveMasterKey(masterKey, password)
		require.NoError(t, err)

		retrievedKey, err := ts.GetMasterKey(password)
		require.NoError(t, err)
		assert.Equal(t, masterKey, retrievedKey)
	})

	t.Run("wrong password fails", func(t *testing.T) {
		masterKey := []byte("0123456789abcdef0123456789abcdef")
		password := "correctpassword"
		wrongPassword := "wrongpassword"

		err := ts.SaveMasterKey(masterKey, password)
		require.NoError(t, err)

		_, err = ts.GetMasterKey(wrongPassword)
		require.Error(t, err)
	})

	t.Run("has master key", func(t *testing.T) {
		ts2 := NewTestStorage(t)
		defer ts2.Cleanup()

		// Initially no key
		assert.False(t, ts2.HasMasterKey())

		// Save a key
		err := ts2.SaveMasterKey([]byte("0123456789abcdef0123456789abcdef"), "password")
		require.NoError(t, err)

		// Now has key
		assert.True(t, ts2.HasMasterKey())
	})

	t.Run("set master key", func(t *testing.T) {
		ts2 := NewTestStorage(t)
		defer ts2.Cleanup()

		key := []byte("0123456789abcdef0123456789abcdef")
		ts2.SetMasterKey(key)

		assert.Equal(t, key, ts2.masterKey)
	})
}

func TestStorage_Secret(t *testing.T) {
	ts := NewTestStorage(t)
	defer ts.Cleanup()

	userID := uuid.New()
	now := time.Now().Truncate(time.Second)

	t.Run("save and get secret", func(t *testing.T) {
		secret := &SecretData{
			ID:            uuid.New(),
			UserID:        userID,
			Type:          "login_password",
			Name:          "test-secret",
			EncryptedData: []byte("encrypted-data"),
			Metadata:      "test metadata",
			Version:       1,
			CreatedAt:     now,
			UpdatedAt:     now,
			IsSynced:      false,
		}

		err := ts.SaveSecret(secret)
		require.NoError(t, err)

		retrieved, err := ts.GetSecret(secret.ID)
		require.NoError(t, err)
		assert.Equal(t, secret.ID, retrieved.ID)
		assert.Equal(t, secret.Name, retrieved.Name)
		assert.Equal(t, secret.Type, retrieved.Type)
		assert.Equal(t, secret.EncryptedData, retrieved.EncryptedData)
		assert.Equal(t, secret.Metadata, retrieved.Metadata)
		assert.Equal(t, secret.Version, retrieved.Version)
		assert.False(t, retrieved.IsSynced)
	})

	t.Run("get secret by name", func(t *testing.T) {
		secret := &SecretData{
			ID:            uuid.New(),
			UserID:        userID,
			Type:          "text",
			Name:          "unique-name",
			EncryptedData: []byte("data"),
			Version:       1,
			CreatedAt:     now,
			UpdatedAt:     now,
		}

		err := ts.SaveSecret(secret)
		require.NoError(t, err)

		retrieved, err := ts.GetSecretByName("unique-name")
		require.NoError(t, err)
		assert.Equal(t, secret.ID, retrieved.ID)
	})

	t.Run("get non-existent secret returns error", func(t *testing.T) {
		_, err := ts.GetSecret(uuid.New())
		require.Error(t, err)
	})

	t.Run("get secrets list", func(t *testing.T) {
		ts2 := NewTestStorage(t)
		defer ts2.Cleanup()

		// Add multiple secrets
		for i := 0; i < 3; i++ {
			secret := &SecretData{
				ID:            uuid.New(),
				UserID:        userID,
				Type:          "text",
				Name:          "secret-" + string(rune('a'+i)),
				EncryptedData: []byte("data"),
				Version:       1,
				CreatedAt:     now,
				UpdatedAt:     now,
			}
			err := ts2.SaveSecret(secret)
			require.NoError(t, err)
		}

		secrets, err := ts2.GetSecrets()
		require.NoError(t, err)
		assert.Len(t, secrets, 3)
	})

	t.Run("update secret", func(t *testing.T) {
		secret := &SecretData{
			ID:            uuid.New(),
			UserID:        userID,
			Type:          "text",
			Name:          "to-update",
			EncryptedData: []byte("original"),
			Version:       1,
			CreatedAt:     now,
			UpdatedAt:     now,
		}

		err := ts.SaveSecret(secret)
		require.NoError(t, err)

		// Update
		secret.EncryptedData = []byte("updated")
		secret.Version = 2
		secret.UpdatedAt = now.Add(time.Hour)

		err = ts.SaveSecret(secret)
		require.NoError(t, err)

		retrieved, err := ts.GetSecret(secret.ID)
		require.NoError(t, err)
		assert.Equal(t, []byte("updated"), retrieved.EncryptedData)
		assert.Equal(t, int64(2), retrieved.Version)
	})

	t.Run("save secret with deleted_at", func(t *testing.T) {
		deletedAt := now.Add(-time.Hour)
		secret := &SecretData{
			ID:            uuid.New(),
			UserID:        userID,
			Type:          "text",
			Name:          "deleted-secret",
			EncryptedData: []byte("data"),
			Version:       1,
			CreatedAt:     now,
			UpdatedAt:     now,
			DeletedAt:     &deletedAt,
		}

		err := ts.SaveSecret(secret)
		require.NoError(t, err)

		// Deleted secrets should not be returned by GetSecret
		_, err = ts.GetSecret(secret.ID)
		require.Error(t, err)
	})
}

func TestStorage_DeleteSecret(t *testing.T) {
	ts := NewTestStorage(t)
	defer ts.Cleanup()

	userID := uuid.New()
	now := time.Now().Truncate(time.Second)

	secret := &SecretData{
		ID:            uuid.New(),
		UserID:        userID,
		Type:          "text",
		Name:          "to-delete",
		EncryptedData: []byte("data"),
		Version:       1,
		CreatedAt:     now,
		UpdatedAt:     now,
		IsSynced:      true,
	}

	err := ts.SaveSecret(secret)
	require.NoError(t, err)

	// Delete
	err = ts.DeleteSecret(secret.ID)
	require.NoError(t, err)

	// Should not be found
	_, err = ts.GetSecret(secret.ID)
	require.Error(t, err)
}

func TestStorage_UnsyncedSecrets(t *testing.T) {
	ts := NewTestStorage(t)
	defer ts.Cleanup()

	userID := uuid.New()
	now := time.Now().Truncate(time.Second)

	// Add synced secret
	syncedSecret := &SecretData{
		ID:            uuid.New(),
		UserID:        userID,
		Type:          "text",
		Name:          "synced",
		EncryptedData: []byte("data"),
		Version:       1,
		CreatedAt:     now,
		UpdatedAt:     now,
		IsSynced:      true,
	}
	err := ts.SaveSecret(syncedSecret)
	require.NoError(t, err)

	// Add unsynced secret
	unsyncedSecret := &SecretData{
		ID:            uuid.New(),
		UserID:        userID,
		Type:          "text",
		Name:          "unsynced",
		EncryptedData: []byte("data"),
		Version:       1,
		CreatedAt:     now,
		UpdatedAt:     now,
		IsSynced:      false,
	}
	err = ts.SaveSecret(unsyncedSecret)
	require.NoError(t, err)

	// Get unsynced
	unsynced, err := ts.GetUnsyncedSecrets()
	require.NoError(t, err)
	assert.Len(t, unsynced, 1)
	assert.Equal(t, unsyncedSecret.ID, unsynced[0].ID)
}

func TestStorage_MarkSynced(t *testing.T) {
	ts := NewTestStorage(t)
	defer ts.Cleanup()

	userID := uuid.New()
	now := time.Now().Truncate(time.Second)

	secret := &SecretData{
		ID:            uuid.New(),
		UserID:        userID,
		Type:          "text",
		Name:          "to-mark-synced",
		EncryptedData: []byte("data"),
		Version:       1,
		CreatedAt:     now,
		UpdatedAt:     now,
		IsSynced:      false,
	}

	err := ts.SaveSecret(secret)
	require.NoError(t, err)

	// Mark synced
	err = ts.MarkSynced(secret.ID)
	require.NoError(t, err)

	// Should not appear in unsynced
	unsynced, err := ts.GetUnsyncedSecrets()
	require.NoError(t, err)
	assert.Len(t, unsynced, 0)
}

func TestStorage_ClearAllSecrets(t *testing.T) {
	ts := NewTestStorage(t)
	defer ts.Cleanup()

	userID := uuid.New()
	now := time.Now().Truncate(time.Second)

	// Add some secrets
	for i := 0; i < 5; i++ {
		secret := &SecretData{
			ID:            uuid.New(),
			UserID:        userID,
			Type:          "text",
			Name:          "secret-" + string(rune('a'+i)),
			EncryptedData: []byte("data"),
			Version:       1,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		err := ts.SaveSecret(secret)
		require.NoError(t, err)
	}

	// Clear all
	err := ts.ClearAllSecrets()
	require.NoError(t, err)

	// Should be empty
	secrets, err := ts.GetSecrets()
	require.NoError(t, err)
	assert.Len(t, secrets, 0)
}

func TestStorage_SaveSecrets(t *testing.T) {
	ts := NewTestStorage(t)
	defer ts.Cleanup()

	userID := uuid.New()
	now := time.Now().Truncate(time.Second)

	secrets := []*SecretData{
		{
			ID:            uuid.New(),
			UserID:        userID,
			Type:          "text",
			Name:          "batch-1",
			EncryptedData: []byte("data1"),
			Version:       1,
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            uuid.New(),
			UserID:        userID,
			Type:          "login_password",
			Name:          "batch-2",
			EncryptedData: []byte("data2"),
			Version:       1,
			CreatedAt:     now,
			UpdatedAt:     now,
		},
	}

	err := ts.SaveSecrets(secrets)
	require.NoError(t, err)

	// Verify all saved
	retrieved, err := ts.GetSecrets()
	require.NoError(t, err)
	assert.Len(t, retrieved, 2)
}

func TestStorage_EncryptDecryptData(t *testing.T) {
	ts := NewTestStorage(t)
	defer ts.Cleanup()

	t.Run("encrypt and decrypt data", func(t *testing.T) {
		key, err := crypto.GenerateKey()
		require.NoError(t, err)

		ts.SetMasterKey(key)

		originalData := map[string]string{
			"username": "testuser",
			"password": "testpass",
		}

		encrypted, err := ts.EncryptData(originalData)
		require.NoError(t, err)
		assert.NotEmpty(t, encrypted)

		var decrypted map[string]string
		err = ts.DecryptData(encrypted, &decrypted)
		require.NoError(t, err)
		assert.Equal(t, originalData, decrypted)
	})

	t.Run("encrypt without master key fails", func(t *testing.T) {
		ts2 := NewTestStorage(t)
		defer ts2.Cleanup()

		_, err := ts2.EncryptData("test")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "master key not set")
	})

	t.Run("decrypt without master key fails", func(t *testing.T) {
		ts2 := NewTestStorage(t)
		defer ts2.Cleanup()

		var result string
		err := ts2.DecryptData([]byte("encrypted"), &result)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "master key not set")
	})
}

func TestStorage_Close(t *testing.T) {
	ts := NewTestStorage(t)

	err := ts.Close()
	require.NoError(t, err)

	// Cleanup the temp dir manually since Close was already called
	os.RemoveAll(ts.tmpDir)
}

func TestStorage_GetMasterKey_FileNotExists(t *testing.T) {
	ts := NewTestStorage(t)
	defer ts.Cleanup()

	// Try to get master key without saving one first
	_, err := ts.GetMasterKey("anypassword")
	require.Error(t, err)
}

func TestStorage_SaveSecrets_WithDeletedAt(t *testing.T) {
	ts := NewTestStorage(t)
	defer ts.Cleanup()

	userID := uuid.New()
	now := time.Now().Truncate(time.Second)
	deletedAt := now.Add(-time.Hour)

	secrets := []*SecretData{
		{
			ID:            uuid.New(),
			UserID:        userID,
			Type:          "text",
			Name:          "deleted-batch",
			EncryptedData: []byte("data"),
			Version:       1,
			CreatedAt:     now,
			UpdatedAt:     now,
			DeletedAt:     &deletedAt,
			IsSynced:      true,
		},
	}

	err := ts.SaveSecrets(secrets)
	require.NoError(t, err)

	// Verify - deleted secret should not appear in GetSecrets
	retrieved, err := ts.GetSecrets()
	require.NoError(t, err)
	assert.Len(t, retrieved, 0)
}

func TestStorage_GetSecretByName_NotFound(t *testing.T) {
	ts := NewTestStorage(t)
	defer ts.Cleanup()

	_, err := ts.GetSecretByName("nonexistent")
	require.Error(t, err)
}

func TestStorage_EncryptData_InvalidKey(t *testing.T) {
	ts := NewTestStorage(t)
	defer ts.Cleanup()

	// Set an invalid key (too short)
	ts.SetMasterKey([]byte("short"))

	_, err := ts.EncryptData("test data")
	require.Error(t, err)
}

func TestStorage_DecryptData_InvalidData(t *testing.T) {
	ts := NewTestStorage(t)
	defer ts.Cleanup()

	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	ts.SetMasterKey(key)

	// Try to decrypt invalid data
	var result string
	err = ts.DecryptData([]byte("invalid-encrypted-data"), &result)
	require.Error(t, err)
}

func TestNewStorage(t *testing.T) {
	// Create a temporary home directory
	tmpHome, err := os.MkdirTemp("", "gophkeeper-storage-new-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpHome)

	// Save and restore HOME
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", originalHome)

	storage, err := NewStorage()
	require.NoError(t, err)
	require.NotNil(t, storage)
	defer storage.Close()

	// Verify we can use the storage
	err = storage.SaveToken("test-token")
	require.NoError(t, err)

	token, err := storage.GetToken()
	require.NoError(t, err)
	assert.Equal(t, "test-token", token)
}

func TestStorage_GetToken_Error(t *testing.T) {
	ts := NewTestStorage(t)
	defer ts.Cleanup()

	// Make tokenFile point to a directory instead of file to cause read error
	err := os.MkdirAll(ts.tokenFile, 0700)
	require.NoError(t, err)

	_, err = ts.GetToken()
	require.Error(t, err)
}

func TestStorage_SaveMasterKey_Error(t *testing.T) {
	ts := NewTestStorage(t)
	defer ts.Cleanup()

	// Make keychainFile point to a directory to cause write error
	err := os.MkdirAll(ts.keychainFile, 0700)
	require.NoError(t, err)

	key := []byte("0123456789abcdef0123456789abcdef")
	err = ts.SaveMasterKey(key, "password")
	require.Error(t, err)
}

func TestStorage_GetMasterKey_DecodeError(t *testing.T) {
	ts := NewTestStorage(t)
	defer ts.Cleanup()

	// Write invalid base64 to keychain file
	err := os.WriteFile(ts.keychainFile, []byte("not-valid-base64!!!"), 0600)
	require.NoError(t, err)

	_, err = ts.GetMasterKey("password")
	require.Error(t, err)
}

