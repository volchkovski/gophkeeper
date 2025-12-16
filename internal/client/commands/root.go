// Package commands provides CLI commands for the GophKeeper client.
package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/volchkovski/gophkeeper/internal/client/api"
	"github.com/volchkovski/gophkeeper/internal/client/config"
	"github.com/volchkovski/gophkeeper/internal/client/storage"
)

var (
	// Version information (set at build time)
	Version   = "dev"
	BuildDate = "unknown"

	// Global variables
	cfg       *config.Config
	apiClient *api.Client
	store     *storage.Storage
)

// rootCmd represents the base command.
var rootCmd = &cobra.Command{
	Use:   "gophkeeper",
	Short: "GophKeeper - secure password manager",
	Long: `GophKeeper is a client-server password manager that securely stores
your passwords, text data, binary files, and credit card information.

All data is encrypted client-side before being sent to the server,
ensuring that only you can access your sensitive information.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip initialization for certain commands
		if cmd.Name() == "version" || cmd.Name() == "help" {
			return nil
		}

		var err error

		// Load configuration
		cfg, err = config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Initialize API client
		apiClient = api.NewClient(cfg.ServerURL, cfg.Timeout)

		// Initialize storage (skip for register command)
		if cmd.Name() != "register" {
			store, err = storage.NewStorage()
			if err != nil {
				return fmt.Errorf("failed to initialize storage: %w", err)
			}

			// Load token if exists
			token, err := store.GetToken()
			if err != nil {
				return fmt.Errorf("failed to load token: %w", err)
			}
			if token != "" {
				apiClient.SetToken(token)
			}
		}

		return nil
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		if store != nil {
			_ = store.Close()
		}
	},
}

// Execute executes the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(registerCmd)
	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(logoutCmd)
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(editCmd)
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(syncCmd)
	rootCmd.AddCommand(exportCmd)
	rootCmd.AddCommand(configCmd)
}

