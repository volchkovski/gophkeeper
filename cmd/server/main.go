// Package main is the entry point for the GophKeeper server.
package main

import (
	"fmt"
	"os"

	"github.com/volchkovski/gophkeeper/internal/server/app"
	"github.com/volchkovski/gophkeeper/internal/server/config"
)

// Version information (set at build time).
var (
	Version   = "dev"
	BuildDate = "unknown"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Create and initialize application
	application, err := app.New(cfg, app.BuildInfo{
		Version:   Version,
		BuildDate: BuildDate,
	})
	if err != nil {
		return err
	}
	defer application.Close()

	// Run the server
	return application.Run()
}
