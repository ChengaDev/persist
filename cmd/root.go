package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Version is set at build time via -ldflags "-X cmd.Version=x.y.z".
var Version = "dev"

var rootCmd = &cobra.Command{
	Use:     "psst",
	Short:   "Secure long-term clipboard manager",
	Version: Version,
	Long: `psst — persist secrets safely across sessions.

Save plain or AES-256-GCM encrypted key-value pairs and retrieve them
directly into your clipboard, never printing sensitive values to stdout.`,
}

// Execute is the entry point called from main.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(saveCmd)
	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(hintCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(lockCmd)
}
