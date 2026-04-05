package cmd

import (
	"fmt"

	"github.com/ChengaDev/persist/internal/storage"
	"github.com/spf13/cobra"
)

var hintCmd = &cobra.Command{
	Use:   "hint",
	Short: "Show the master password hint",
	Long: `Prints the plain-text hint you set during psst init.
The hint is stored unencrypted — it is only a memory aid, not the password.`,
	Args: cobra.NoArgs,
	RunE: runHint,
}

func runHint(_ *cobra.Command, _ []string) error {
	store, err := storage.Open()
	if err != nil {
		return err
	}
	defer store.Close()

	if err := requireInitialized(store); err != nil {
		return err
	}

	hint, err := store.GetHint()
	if err != nil {
		return err
	}
	if hint == "" {
		fmt.Println("No hint set. Re-run `psst init` after deleting the database to add one.")
		return nil
	}
	fmt.Println("Hint:", hint)
	return nil
}
