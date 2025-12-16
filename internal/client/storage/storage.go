// Package storage provides local storage functionality for the GophKeeper client.
package storage

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3" // SQLite driver
	"github.com/volchkovski/gophkeeper/internal/client/config"
	"github.com/volchkovski/gophkeeper/internal/common/crypto"
)

// SecretData represents a secret stored locally.
type SecretData struct {
	ID            uuid.UUID `json:"id"`
	UserID        uuid.UUID `json:"user_id"`
	Type          string    `json:"type"`
	Name          string    `json:"name"`
	EncryptedData []byte    `json:"encrypted_data"`
	Metadata      string    `json:"metadata"`
	Version       int64     `json:"version"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	DeletedAt     *time.Time `json:"deleted_at,omitempty"`
	IsSynced      bool      `json:"is_synced"`
}

// Storage provides local data storage.
type Storage struct {
	db          *sql.DB
	masterKey   []byte
	tokenFile   string
	keychainFile string
}

// NewStorage creates a new Storage instance.
func NewStorage() (*Storage, error) {
	storagePath, err := config.GetStoragePath()
	if err != nil {
		return nil, err
	}

	configDir, err := config.GetConfigDir()
	if err != nil {
		return nil, err
	}

	keychainPath, err := config.GetKeychainPath()
	if err != nil {
		return nil, err
	}

	// Ensure config directory exists
	if err := config.EnsureConfigDir(); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite3", storagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	s := &Storage{
		db:           db,
		tokenFile:    configDir + "/token",
		keychainFile: keychainPath,
	}

	if err := s.initDB(); err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return s, nil
}

// initDB creates necessary tables.
func (s *Storage) initDB() error {
	schema := `
	CREATE TABLE IF NOT EXISTS secrets (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		type TEXT NOT NULL,
		name TEXT NOT NULL,
		encrypted_data BLOB NOT NULL,
		metadata TEXT,
		version INTEGER NOT NULL DEFAULT 1,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		deleted_at DATETIME,
		is_synced INTEGER NOT NULL DEFAULT 0
	);
	CREATE INDEX IF NOT EXISTS idx_secrets_user_id ON secrets(user_id);
	CREATE INDEX IF NOT EXISTS idx_secrets_name ON secrets(name);
	CREATE INDEX IF NOT EXISTS idx_secrets_is_synced ON secrets(is_synced);
	`

	_, err := s.db.Exec(schema)
	return err
}

// Close closes the storage.
func (s *Storage) Close() error {
	return s.db.Close()
}

// SaveToken saves the authentication token.
func (s *Storage) SaveToken(token string) error {
	return os.WriteFile(s.tokenFile, []byte(token), 0600)
}

// GetToken retrieves the authentication token.
func (s *Storage) GetToken() (string, error) {
	data, err := os.ReadFile(s.tokenFile)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(data), nil
}

// DeleteToken removes the authentication token.
func (s *Storage) DeleteToken() error {
	err := os.Remove(s.tokenFile)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// SaveMasterKey saves the master key encrypted with the user's password.
func (s *Storage) SaveMasterKey(key []byte, password string) error {
	encryptedKey, err := crypto.EncryptWithPassword(key, password)
	if err != nil {
		return fmt.Errorf("failed to encrypt master key: %w", err)
	}

	encoded := base64.StdEncoding.EncodeToString(encryptedKey)
	return os.WriteFile(s.keychainFile, []byte(encoded), 0600)
}

// GetMasterKey retrieves and decrypts the master key.
func (s *Storage) GetMasterKey(password string) ([]byte, error) {
	encoded, err := os.ReadFile(s.keychainFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read keychain: %w", err)
	}

	encryptedKey, err := base64.StdEncoding.DecodeString(string(encoded))
	if err != nil {
		return nil, fmt.Errorf("failed to decode keychain: %w", err)
	}

	key, err := crypto.DecryptWithPassword(encryptedKey, password)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt master key: %w", err)
	}

	return key, nil
}

// HasMasterKey checks if a master key exists.
func (s *Storage) HasMasterKey() bool {
	_, err := os.Stat(s.keychainFile)
	return err == nil
}

// SetMasterKey sets the master key for encrypting data.
func (s *Storage) SetMasterKey(key []byte) {
	s.masterKey = key
}

// SaveSecret saves a secret to local storage.
func (s *Storage) SaveSecret(secret *SecretData) error {
	query := `
		INSERT OR REPLACE INTO secrets 
		(id, user_id, type, name, encrypted_data, metadata, version, created_at, updated_at, deleted_at, is_synced)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	var deletedAt interface{}
	if secret.DeletedAt != nil {
		deletedAt = secret.DeletedAt
	}

	_, err := s.db.Exec(query,
		secret.ID.String(),
		secret.UserID.String(),
		secret.Type,
		secret.Name,
		secret.EncryptedData,
		secret.Metadata,
		secret.Version,
		secret.CreatedAt,
		secret.UpdatedAt,
		deletedAt,
		secret.IsSynced,
	)
	return err
}

// GetSecret retrieves a secret by ID.
func (s *Storage) GetSecret(id uuid.UUID) (*SecretData, error) {
	query := `
		SELECT id, user_id, type, name, encrypted_data, metadata, version, created_at, updated_at, deleted_at, is_synced
		FROM secrets
		WHERE id = ? AND deleted_at IS NULL
	`

	return s.scanSecret(s.db.QueryRow(query, id.String()))
}

// GetSecretByName retrieves a secret by name.
func (s *Storage) GetSecretByName(name string) (*SecretData, error) {
	query := `
		SELECT id, user_id, type, name, encrypted_data, metadata, version, created_at, updated_at, deleted_at, is_synced
		FROM secrets
		WHERE name = ? AND deleted_at IS NULL
	`

	return s.scanSecret(s.db.QueryRow(query, name))
}

// GetSecrets retrieves all secrets.
func (s *Storage) GetSecrets() ([]*SecretData, error) {
	query := `
		SELECT id, user_id, type, name, encrypted_data, metadata, version, created_at, updated_at, deleted_at, is_synced
		FROM secrets
		WHERE deleted_at IS NULL
		ORDER BY name
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var secrets []*SecretData
	for rows.Next() {
		secret, err := s.scanSecretRow(rows)
		if err != nil {
			return nil, err
		}
		secrets = append(secrets, secret)
	}

	return secrets, rows.Err()
}

// GetUnsyncedSecrets retrieves secrets that need to be synced.
func (s *Storage) GetUnsyncedSecrets() ([]*SecretData, error) {
	query := `
		SELECT id, user_id, type, name, encrypted_data, metadata, version, created_at, updated_at, deleted_at, is_synced
		FROM secrets
		WHERE is_synced = 0
		ORDER BY updated_at
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var secrets []*SecretData
	for rows.Next() {
		secret, err := s.scanSecretRow(rows)
		if err != nil {
			return nil, err
		}
		secrets = append(secrets, secret)
	}

	return secrets, rows.Err()
}

// MarkSynced marks a secret as synced.
func (s *Storage) MarkSynced(id uuid.UUID) error {
	query := `UPDATE secrets SET is_synced = 1 WHERE id = ?`
	_, err := s.db.Exec(query, id.String())
	return err
}

// DeleteSecret performs soft delete on a secret.
func (s *Storage) DeleteSecret(id uuid.UUID) error {
	query := `UPDATE secrets SET deleted_at = ?, is_synced = 0 WHERE id = ?`
	_, err := s.db.Exec(query, time.Now(), id.String())
	return err
}

// ClearAllSecrets removes all secrets from local storage.
func (s *Storage) ClearAllSecrets() error {
	_, err := s.db.Exec("DELETE FROM secrets")
	return err
}

// SaveSecrets saves multiple secrets to local storage.
func (s *Storage) SaveSecrets(secrets []*SecretData) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT OR REPLACE INTO secrets 
		(id, user_id, type, name, encrypted_data, metadata, version, created_at, updated_at, deleted_at, is_synced)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, secret := range secrets {
		var deletedAt interface{}
		if secret.DeletedAt != nil {
			deletedAt = secret.DeletedAt
		}

		_, err := stmt.Exec(
			secret.ID.String(),
			secret.UserID.String(),
			secret.Type,
			secret.Name,
			secret.EncryptedData,
			secret.Metadata,
			secret.Version,
			secret.CreatedAt,
			secret.UpdatedAt,
			deletedAt,
			secret.IsSynced,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// scanSecret scans a single secret from a row.
func (s *Storage) scanSecret(row *sql.Row) (*SecretData, error) {
	var secret SecretData
	var id, userID string
	var deletedAt sql.NullTime
	var isSynced int

	err := row.Scan(
		&id,
		&userID,
		&secret.Type,
		&secret.Name,
		&secret.EncryptedData,
		&secret.Metadata,
		&secret.Version,
		&secret.CreatedAt,
		&secret.UpdatedAt,
		&deletedAt,
		&isSynced,
	)
	if err != nil {
		return nil, err
	}

	secret.ID, _ = uuid.Parse(id)
	secret.UserID, _ = uuid.Parse(userID)
	if deletedAt.Valid {
		secret.DeletedAt = &deletedAt.Time
	}
	secret.IsSynced = isSynced == 1

	return &secret, nil
}

// scanSecretRow scans a secret from rows.
func (s *Storage) scanSecretRow(rows *sql.Rows) (*SecretData, error) {
	var secret SecretData
	var id, userID string
	var deletedAt sql.NullTime
	var isSynced int

	err := rows.Scan(
		&id,
		&userID,
		&secret.Type,
		&secret.Name,
		&secret.EncryptedData,
		&secret.Metadata,
		&secret.Version,
		&secret.CreatedAt,
		&secret.UpdatedAt,
		&deletedAt,
		&isSynced,
	)
	if err != nil {
		return nil, err
	}

	secret.ID, _ = uuid.Parse(id)
	secret.UserID, _ = uuid.Parse(userID)
	if deletedAt.Valid {
		secret.DeletedAt = &deletedAt.Time
	}
	secret.IsSynced = isSynced == 1

	return &secret, nil
}

// EncryptData encrypts data with the master key.
func (s *Storage) EncryptData(data interface{}) ([]byte, error) {
	if s.masterKey == nil {
		return nil, fmt.Errorf("master key not set")
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %w", err)
	}

	encryptor, err := crypto.NewEncryptor(s.masterKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create encryptor: %w", err)
	}

	return encryptor.Encrypt(jsonData)
}

// DecryptData decrypts data with the master key.
func (s *Storage) DecryptData(encryptedData []byte, target interface{}) error {
	if s.masterKey == nil {
		return fmt.Errorf("master key not set")
	}

	encryptor, err := crypto.NewEncryptor(s.masterKey)
	if err != nil {
		return fmt.Errorf("failed to create encryptor: %w", err)
	}

	decrypted, err := encryptor.Decrypt(encryptedData)
	if err != nil {
		return fmt.Errorf("failed to decrypt data: %w", err)
	}

	return json.Unmarshal(decrypted, target)
}

