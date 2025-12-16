package commands

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/volchkovski/gophkeeper/internal/client/config"
	"github.com/volchkovski/gophkeeper/internal/client/storage"
	"github.com/volchkovski/gophkeeper/internal/common/models"
)

// Secret types
const (
	TypeLoginPassword = "login_password"
	TypeText          = "text"
	TypeBinary        = "binary"
	TypeCard          = "card"
)

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new secret",
	Long:  "Add a new secret (password, text, binary, or card)",
}

var addPasswordCmd = &cobra.Command{
	Use:   "password",
	Short: "Add a login/password pair",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := ensureLoggedIn(); err != nil {
			return err
		}
		if err := loadMasterKey(); err != nil {
			return err
		}

		// Get secret name
		name, err := promptString("Name (e.g., 'Gmail', 'GitHub'): ")
		if err != nil {
			return err
		}

		// Get login
		login, err := promptString("Login/Username: ")
		if err != nil {
			return err
		}

		// Get password
		password, err := promptPassword("Password: ")
		if err != nil {
			return err
		}

		// Optional website
		website, _ := promptString("Website (optional): ")

		// Optional notes
		notes, _ := promptString("Notes (optional): ")

		// Create data structures
		data := models.LoginPassword{
			Login:    login,
			Password: password,
		}

		metadata := models.Metadata{
			Website: website,
			Notes:   notes,
		}

		// Encrypt and save
		return saveSecret(name, TypeLoginPassword, data, metadata)
	},
}

var addTextCmd = &cobra.Command{
	Use:   "text",
	Short: "Add arbitrary text data",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := ensureLoggedIn(); err != nil {
			return err
		}
		if err := loadMasterKey(); err != nil {
			return err
		}

		// Get secret name
		name, err := promptString("Name: ")
		if err != nil {
			return err
		}

		// Get text content
		fmt.Println("Enter text content (press Ctrl+D when done):")
		content, err := readMultilineInput()
		if err != nil {
			return err
		}

		// Optional notes
		notes, _ := promptString("Notes (optional): ")

		data := models.TextData{
			Content: content,
		}

		metadata := models.Metadata{
			Notes: notes,
		}

		return saveSecret(name, TypeText, data, metadata)
	},
}

var addBinaryCmd = &cobra.Command{
	Use:   "binary [file]",
	Short: "Add a binary file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := ensureLoggedIn(); err != nil {
			return err
		}
		if err := loadMasterKey(); err != nil {
			return err
		}

		filePath := args[0]

		// Read file
		fileData, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}

		// Get secret name
		name, err := promptString("Name (press Enter to use filename): ")
		if err != nil {
			return err
		}
		if name == "" {
			name = filePath
		}

		// Optional notes
		notes, _ := promptString("Notes (optional): ")

		data := models.BinaryData{
			Data:     fileData,
			Filename: filePath,
		}

		metadata := models.Metadata{
			Notes: notes,
		}

		return saveSecret(name, TypeBinary, data, metadata)
	},
}

var addCardCmd = &cobra.Command{
	Use:   "card",
	Short: "Add credit/debit card information",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := ensureLoggedIn(); err != nil {
			return err
		}
		if err := loadMasterKey(); err != nil {
			return err
		}

		// Get secret name
		name, err := promptString("Card name (e.g., 'Visa Personal'): ")
		if err != nil {
			return err
		}

		// Get card details
		number, err := promptString("Card number: ")
		if err != nil {
			return err
		}

		holder, err := promptString("Card holder name: ")
		if err != nil {
			return err
		}

		expiry, err := promptString("Expiry date (MM/YY): ")
		if err != nil {
			return err
		}

		cvv, err := promptPassword("CVV: ")
		if err != nil {
			return err
		}

		// Optional notes
		notes, _ := promptString("Notes (optional): ")

		data := models.CardData{
			Number:     number,
			Holder:     holder,
			ExpiryDate: expiry,
			CVV:        cvv,
		}

		metadata := models.Metadata{
			Notes: notes,
		}

		return saveSecret(name, TypeCard, data, metadata)
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all secrets",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := ensureLoggedIn(); err != nil {
			return err
		}

		secrets, err := store.GetSecrets()
		if err != nil {
			return fmt.Errorf("failed to get secrets: %w", err)
		}

		if len(secrets) == 0 {
			fmt.Println("No secrets found.")
			return nil
		}

		fmt.Printf("%-36s %-15s %-30s %s\n", "ID", "TYPE", "NAME", "SYNCED")
		fmt.Println(repeatString("-", 90))

		for _, secret := range secrets {
			syncStatus := "✗"
			if secret.IsSynced {
				syncStatus = "✓"
			}
			fmt.Printf("%-36s %-15s %-30s %s\n",
				secret.ID.String()[:8]+"...",
				secret.Type,
				truncateString(secret.Name, 30),
				syncStatus,
			)
		}

		fmt.Printf("\nTotal: %d secrets\n", len(secrets))
		return nil
	},
}

var getCmd = &cobra.Command{
	Use:   "get <name>",
	Short: "Get a secret by name",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := ensureLoggedIn(); err != nil {
			return err
		}
		if err := loadMasterKey(); err != nil {
			return err
		}

		name := args[0]

		secret, err := store.GetSecretByName(name)
		if err != nil {
			return fmt.Errorf("secret not found: %s", name)
		}

		// Decrypt and display
		return displaySecret(secret)
	},
}

var editCmd = &cobra.Command{
	Use:   "edit <name>",
	Short: "Edit a secret",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := ensureLoggedIn(); err != nil {
			return err
		}
		if err := loadMasterKey(); err != nil {
			return err
		}

		name := args[0]

		secret, err := store.GetSecretByName(name)
		if err != nil {
			return fmt.Errorf("secret not found: %s", name)
		}

		// Edit based on type
		switch secret.Type {
		case TypeLoginPassword:
			return editLoginPassword(secret)
		case TypeText:
			return editText(secret)
		case TypeCard:
			return editCard(secret)
		default:
			return fmt.Errorf("unsupported secret type for editing: %s", secret.Type)
		}
	},
}

var deleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a secret",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := ensureLoggedIn(); err != nil {
			return err
		}

		name := args[0]

		secret, err := store.GetSecretByName(name)
		if err != nil {
			return fmt.Errorf("secret not found: %s", name)
		}

		// Confirm deletion
		confirm, err := promptString(fmt.Sprintf("Are you sure you want to delete '%s'? (y/N): ", name))
		if err != nil {
			return err
		}

		if confirm != "y" && confirm != "Y" {
			fmt.Println("Deletion cancelled.")
			return nil
		}

		if err := store.DeleteSecret(secret.ID); err != nil {
			return fmt.Errorf("failed to delete secret: %w", err)
		}

		fmt.Printf("Secret '%s' deleted.\n", name)
		return nil
	},
}

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Synchronize secrets with server",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := ensureLoggedIn(); err != nil {
			return err
		}

		fmt.Println("Syncing secrets...")
		if err := performSync(); err != nil {
			return fmt.Errorf("sync failed: %w", err)
		}

		fmt.Println("Sync completed successfully.")
		return nil
	},
}

var exportCmd = &cobra.Command{
	Use:   "export <name> <file>",
	Short: "Export a secret to a file",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := ensureLoggedIn(); err != nil {
			return err
		}
		if err := loadMasterKey(); err != nil {
			return err
		}

		name := args[0]
		filePath := args[1]

		secret, err := store.GetSecretByName(name)
		if err != nil {
			return fmt.Errorf("secret not found: %s", name)
		}

		// Only binary secrets can be exported as files
		if secret.Type != TypeBinary {
			return fmt.Errorf("only binary secrets can be exported to files")
		}

		// Decrypt data
		var data models.BinaryData
		if err := store.DecryptData(secret.EncryptedData, &data); err != nil {
			return fmt.Errorf("failed to decrypt secret: %w", err)
		}

		// Write to file
		if err := os.WriteFile(filePath, data.Data, 0600); err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}

		fmt.Printf("Secret exported to: %s\n", filePath)
		return nil
	},
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Show or modify configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfg == nil {
			var err error
			cfg, err = config.Load()
			if err != nil {
				return err
			}
		}

		fmt.Printf("Server URL: %s\n", cfg.ServerURL)
		fmt.Printf("Timeout: %s\n", cfg.Timeout)
		fmt.Printf("Auto Sync: %v\n", cfg.AutoSync)
		fmt.Printf("Sync on Start: %v\n", cfg.SyncOnStart)

		return nil
	},
}

func init() {
	addCmd.AddCommand(addPasswordCmd)
	addCmd.AddCommand(addTextCmd)
	addCmd.AddCommand(addBinaryCmd)
	addCmd.AddCommand(addCardCmd)
}

// saveSecret encrypts and saves a secret.
func saveSecret(name, secretType string, data, metadata interface{}) error {
	// Encrypt data
	encryptedData, err := store.EncryptData(data)
	if err != nil {
		return fmt.Errorf("failed to encrypt data: %w", err)
	}

	// Marshal metadata
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Create secret
	secret := &storage.SecretData{
		ID:            uuid.New(),
		Type:          secretType,
		Name:          name,
		EncryptedData: encryptedData,
		Metadata:      string(metadataJSON),
		Version:       1,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		IsSynced:      false,
	}

	// Save to local storage
	if err := store.SaveSecret(secret); err != nil {
		return fmt.Errorf("failed to save secret: %w", err)
	}

	fmt.Printf("Secret '%s' saved successfully.\n", name)

	// Auto sync if enabled
	if cfg.AutoSync {
		fmt.Println("Syncing...")
		if err := performSync(); err != nil {
			fmt.Printf("Warning: sync failed: %v\n", err)
		}
	}

	return nil
}

// displaySecret decrypts and displays a secret.
func displaySecret(secret *storage.SecretData) error {
	fmt.Printf("\n=== %s ===\n", secret.Name)
	fmt.Printf("Type: %s\n", secret.Type)
	fmt.Printf("Version: %d\n", secret.Version)
	fmt.Printf("Created: %s\n", secret.CreatedAt.Format(time.RFC3339))
	fmt.Printf("Updated: %s\n", secret.UpdatedAt.Format(time.RFC3339))
	fmt.Println()

	switch secret.Type {
	case TypeLoginPassword:
		var data models.LoginPassword
		if err := store.DecryptData(secret.EncryptedData, &data); err != nil {
			return fmt.Errorf("failed to decrypt: %w", err)
		}
		fmt.Printf("Login: %s\n", data.Login)
		fmt.Printf("Password: %s\n", data.Password)

	case TypeText:
		var data models.TextData
		if err := store.DecryptData(secret.EncryptedData, &data); err != nil {
			return fmt.Errorf("failed to decrypt: %w", err)
		}
		fmt.Printf("Content:\n%s\n", data.Content)

	case TypeBinary:
		var data models.BinaryData
		if err := store.DecryptData(secret.EncryptedData, &data); err != nil {
			return fmt.Errorf("failed to decrypt: %w", err)
		}
		fmt.Printf("Filename: %s\n", data.Filename)
		fmt.Printf("Size: %d bytes\n", len(data.Data))
		fmt.Println("(Use 'gophkeeper export' to save to file)")

	case TypeCard:
		var data models.CardData
		if err := store.DecryptData(secret.EncryptedData, &data); err != nil {
			return fmt.Errorf("failed to decrypt: %w", err)
		}
		fmt.Printf("Card Number: %s\n", maskCardNumber(data.Number))
		fmt.Printf("Holder: %s\n", data.Holder)
		fmt.Printf("Expiry: %s\n", data.ExpiryDate)
		fmt.Printf("CVV: ***\n")

		// Ask if user wants to see full details
		show, _ := promptString("Show full card number and CVV? (y/N): ")
		if show == "y" || show == "Y" {
			fmt.Printf("\nCard Number: %s\n", data.Number)
			fmt.Printf("CVV: %s\n", data.CVV)
		}
	}

	// Show metadata
	if secret.Metadata != "" {
		var metadata models.Metadata
		if err := json.Unmarshal([]byte(secret.Metadata), &metadata); err == nil {
			if metadata.Website != "" {
				fmt.Printf("\nWebsite: %s\n", metadata.Website)
			}
			if metadata.Notes != "" {
				fmt.Printf("Notes: %s\n", metadata.Notes)
			}
		}
	}

	return nil
}

// editLoginPassword edits a login/password secret.
func editLoginPassword(secret *storage.SecretData) error {
	var data models.LoginPassword
	if err := store.DecryptData(secret.EncryptedData, &data); err != nil {
		return fmt.Errorf("failed to decrypt: %w", err)
	}

	fmt.Printf("Current login: %s\n", data.Login)
	newLogin, err := promptString("New login (press Enter to keep): ")
	if err != nil {
		return err
	}
	if newLogin != "" {
		data.Login = newLogin
	}

	fmt.Println("Current password: ********")
	newPassword, err := promptPassword("New password (press Enter to keep): ")
	if err != nil {
		return err
	}
	if newPassword != "" {
		data.Password = newPassword
	}

	return updateSecret(secret, data)
}

// editText edits a text secret.
func editText(secret *storage.SecretData) error {
	var data models.TextData
	if err := store.DecryptData(secret.EncryptedData, &data); err != nil {
		return fmt.Errorf("failed to decrypt: %w", err)
	}

	fmt.Printf("Current content:\n%s\n", data.Content)
	fmt.Println("\nEnter new content (press Ctrl+D when done, or press Enter twice to keep current):")
	newContent, err := readMultilineInput()
	if err != nil {
		return err
	}
	if newContent != "" {
		data.Content = newContent
	}

	return updateSecret(secret, data)
}

// editCard edits a card secret.
func editCard(secret *storage.SecretData) error {
	var data models.CardData
	if err := store.DecryptData(secret.EncryptedData, &data); err != nil {
		return fmt.Errorf("failed to decrypt: %w", err)
	}

	fmt.Printf("Current card number: %s\n", maskCardNumber(data.Number))
	newNumber, err := promptString("New card number (press Enter to keep): ")
	if err != nil {
		return err
	}
	if newNumber != "" {
		data.Number = newNumber
	}

	fmt.Printf("Current holder: %s\n", data.Holder)
	newHolder, err := promptString("New holder (press Enter to keep): ")
	if err != nil {
		return err
	}
	if newHolder != "" {
		data.Holder = newHolder
	}

	fmt.Printf("Current expiry: %s\n", data.ExpiryDate)
	newExpiry, err := promptString("New expiry (press Enter to keep): ")
	if err != nil {
		return err
	}
	if newExpiry != "" {
		data.ExpiryDate = newExpiry
	}

	newCVV, err := promptPassword("New CVV (press Enter to keep): ")
	if err != nil {
		return err
	}
	if newCVV != "" {
		data.CVV = newCVV
	}

	return updateSecret(secret, data)
}

// updateSecret updates a secret with new data.
func updateSecret(secret *storage.SecretData, data interface{}) error {
	encryptedData, err := store.EncryptData(data)
	if err != nil {
		return fmt.Errorf("failed to encrypt data: %w", err)
	}

	secret.EncryptedData = encryptedData
	secret.Version++
	secret.UpdatedAt = time.Now()
	secret.IsSynced = false

	if err := store.SaveSecret(secret); err != nil {
		return fmt.Errorf("failed to save secret: %w", err)
	}

	fmt.Printf("Secret '%s' updated successfully.\n", secret.Name)

	if cfg.AutoSync {
		fmt.Println("Syncing...")
		if err := performSync(); err != nil {
			fmt.Printf("Warning: sync failed: %v\n", err)
		}
	}

	return nil
}

// Helper functions

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func repeatString(s string, count int) string {
	result := ""
	for i := 0; i < count; i++ {
		result += s
	}
	return result
}

func maskCardNumber(number string) string {
	if len(number) < 4 {
		return "****"
	}
	return repeatString("*", len(number)-4) + number[len(number)-4:]
}

func readMultilineInput() (string, error) {
	var lines []string
	scanner := bufio.NewScanner(os.Stdin)
	emptyCount := 0
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			emptyCount++
			if emptyCount >= 2 {
				break
			}
		} else {
			emptyCount = 0
		}
		lines = append(lines, line)
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	// Remove trailing empty lines
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return strings.Join(lines, "\n"), nil
}

func parseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}

func parseTime(s string) (time.Time, error) {
	return time.Parse("2006-01-02T15:04:05Z07:00", s)
}

