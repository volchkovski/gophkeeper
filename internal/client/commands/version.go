package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("GophKeeper Client\n")
		fmt.Printf("Version: %s\n", Version)
		fmt.Printf("Build Date: %s\n", BuildDate)
	},
}

