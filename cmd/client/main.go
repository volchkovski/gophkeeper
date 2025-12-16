// Package main is the entry point for the GophKeeper client.
package main

import (
	"github.com/volchkovski/gophkeeper/internal/client/commands"
)

// Version information (set at build time).
var (
	Version   = "dev"
	BuildDate = "unknown"
)

func main() {
	// Set version info
	commands.Version = Version
	commands.BuildDate = BuildDate

	// Execute CLI
	commands.Execute()
}
