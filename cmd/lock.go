package cmd

import (
	"fmt"

	"github.com/ChengaDev/persist/internal/session"
	"github.com/spf13/cobra"
)

var lockCmd = &cobra.Command{
	Use:   "lock",
	Short: "Expire the current session immediately",
	Long:  `Deletes the session file so the next psst get or psst save --secure requires the master password again.`,
	Args:  cobra.NoArgs,
	RunE:  runLock,
}

func runLock(_ *cobra.Command, _ []string) error {
	if err := session.FileClear(); err != nil {
		return err
	}
	fmt.Println("Session cleared. Master password required for next operation.")
	return nil
}
