package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/volchkovski/gophkeeper/internal/client/api"
	"github.com/volchkovski/gophkeeper/internal/client/config"
	"github.com/volchkovski/gophkeeper/internal/client/storage"
	"github.com/volchkovski/gophkeeper/internal/common/crypto"
	"golang.org/x/term"
)

var registerCmd = &cobra.Command{
	Use:   "register",
	Short: "Register a new user",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get username
		username, err := promptString("Username: ")
		if err != nil {
			return err
		}

		// Get password
		password, err := promptPassword("Password: ")
		if err != nil {
			return err
		}

		// Confirm password
		confirmPassword, err := promptPassword("Confirm password: ")
		if err != nil {
			return err
		}

		if password != confirmPassword {
			return fmt.Errorf("passwords do not match")
		}

		// Initialize storage for registration
		var localStore *storage.Storage
		localStore, err = storage.NewStorage()
		if err != nil {
			return fmt.Errorf("failed to initialize storage: %w", err)
		}
		defer localStore.Close()

		// Register with server
		result, err := apiClient.Register(username, password)
		if err != nil {
			return fmt.Errorf("registration failed: %w", err)
		}

		fmt.Printf("Successfully registered user: %s\n", result.Username)

		// Generate master key
		masterKey, err := crypto.GenerateKey()
		if err != nil {
			return fmt.Errorf("failed to generate master key: %w", err)
		}

		// Save master key encrypted with password
		if err := localStore.SaveMasterKey(masterKey, password); err != nil {
			return fmt.Errorf("failed to save master key: %w", err)
		}

		fmt.Println("Master key generated and saved.")
		fmt.Println("Please login to start using GophKeeper.")

		return nil
	},
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to GophKeeper",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get username
		username, err := promptString("Username: ")
		if err != nil {
			return err
		}

		// Get password
		password, err := promptPassword("Password: ")
		if err != nil {
			return err
		}

		// Login with server
		result, err := apiClient.Login(username, password)
		if err != nil {
			return fmt.Errorf("login failed: %w", err)
		}

		// Save token
		if err := store.SaveToken(result.AccessToken); err != nil {
			return fmt.Errorf("failed to save token: %w", err)
		}

		// Check if master key exists
		if !store.HasMasterKey() {
			// First login on this device - generate master key
			masterKey, err := crypto.GenerateKey()
			if err != nil {
				return fmt.Errorf("failed to generate master key: %w", err)
			}

			if err := store.SaveMasterKey(masterKey, password); err != nil {
				return fmt.Errorf("failed to save master key: %w", err)
			}

			fmt.Println("Master key generated and saved.")
		} else {
			// Load master key
			masterKey, err := store.GetMasterKey(password)
			if err != nil {
				return fmt.Errorf("failed to load master key: %w", err)
			}
			store.SetMasterKey(masterKey)
		}

		fmt.Printf("Successfully logged in as: %s\n", username)

		// Sync on login if enabled
		if cfg.SyncOnStart {
			fmt.Println("Syncing secrets...")
			if err := performSync(); err != nil {
				fmt.Printf("Warning: sync failed: %v\n", err)
			}
		}

		return nil
	},
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Logout from GophKeeper",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := store.DeleteToken(); err != nil {
			return fmt.Errorf("failed to delete token: %w", err)
		}

		fmt.Println("Successfully logged out.")
		return nil
	},
}

// promptString prompts for a string input.
func promptString(prompt string) (string, error) {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(input), nil
}

// promptPassword prompts for a password input (hidden).
func promptPassword(prompt string) (string, error) {
	fmt.Print(prompt)
	password, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		return "", err
	}
	return string(password), nil
}

// ensureLoggedIn checks if user is logged in and master key is loaded.
func ensureLoggedIn() error {
	token, err := store.GetToken()
	if err != nil {
		return fmt.Errorf("failed to check login status: %w", err)
	}

	if token == "" {
		return fmt.Errorf("not logged in. Please run 'gophkeeper login' first")
	}

	apiClient.SetToken(token)

	// Load master key if not already loaded
	if !store.HasMasterKey() {
		return fmt.Errorf("master key not found. Please run 'gophkeeper login' first")
	}

	return nil
}

// loadMasterKey loads the master key with password.
func loadMasterKey() error {
	if store.HasMasterKey() {
		password, err := promptPassword("Enter your password to unlock: ")
		if err != nil {
			return err
		}

		masterKey, err := store.GetMasterKey(password)
		if err != nil {
			return fmt.Errorf("failed to unlock: %w", err)
		}
		store.SetMasterKey(masterKey)
	}
	return nil
}

// performSync performs synchronization with the server.
func performSync() error {
	// Get unsynced secrets
	unsyncedSecrets, err := store.GetUnsyncedSecrets()
	if err != nil {
		return fmt.Errorf("failed to get unsynced secrets: %w", err)
	}

	// Prepare sync request
	syncSecrets := make([]api.SyncSecretRequest, len(unsyncedSecrets))
	for i, s := range unsyncedSecrets {
		syncSecrets[i] = api.SyncSecretRequest{
			ID:            s.ID.String(),
			Type:          s.Type.String(),
			Name:          s.Name,
			EncryptedData: s.EncryptedData,
			Metadata:      s.Metadata,
			Version:       s.Version,
			UpdatedAt:     s.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	// Sync with server
	result, err := apiClient.Sync(syncSecrets)
	if err != nil {
		return err
	}

	// Save updated secrets from server
	for _, s := range result.UpdatedSecrets {
		secret := apiResponseToStorage(s)
		secret.IsSynced = true
		if err := store.SaveSecret(secret); err != nil {
			return fmt.Errorf("failed to save secret: %w", err)
		}
	}

	// Mark local secrets as synced
	for _, s := range unsyncedSecrets {
		if err := store.MarkSynced(s.ID); err != nil {
			return fmt.Errorf("failed to mark secret as synced: %w", err)
		}
	}

	// Handle conflicts (for now, just report them)
	if len(result.Conflicts) > 0 {
		fmt.Printf("Warning: %d conflicts detected. Server version will be used.\n", len(result.Conflicts))
		for _, conflict := range result.Conflicts {
			fmt.Printf("  - %s (local v%d vs server v%d)\n",
				conflict.ClientVersion.Name,
				conflict.ClientVersion.Version,
				conflict.ServerVersion.Version,
			)
			// Use server version
			secret := apiResponseToStorage(conflict.ServerVersion)
			secret.IsSynced = true
			if err := store.SaveSecret(secret); err != nil {
				return fmt.Errorf("failed to save conflict resolution: %w", err)
			}
		}
	}

	return nil
}

// apiResponseToStorage converts API response to storage format.
func apiResponseToStorage(s api.SecretResponse) *storage.SecretData {
	secret := &storage.SecretData{
		Type:          storage.NewSecretType(s.Type),
		Name:          s.Name,
		EncryptedData: s.EncryptedData,
		Metadata:      s.Metadata,
		Version:       s.Version,
	}

	if id, err := parseUUID(s.ID); err == nil {
		secret.ID = id
	}

	if t, err := parseTime(s.CreatedAt); err == nil {
		secret.CreatedAt = t
	}
	if t, err := parseTime(s.UpdatedAt); err == nil {
		secret.UpdatedAt = t
	}

	return secret
}

// ensureConfigDir ensures the config directory exists.
func ensureConfigDir() error {
	return config.EnsureConfigDir()
}

