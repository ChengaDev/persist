package cmd

import (
	"fmt"

	"github.com/ChengaDev/persist/internal/storage"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:     "delete <key>",
	Aliases: []string{"rm"},
	Short:   "Delete an entry",
	Args:    cobra.ExactArgs(1),
	RunE:    runDelete,
}

func runDelete(_ *cobra.Command, args []string) error {
	key := args[0]

	store, err := storage.Open()
	if err != nil {
		return err
	}
	defer store.Close()

	if err := requireInitialized(store); err != nil {
		return err
	}

	if err := store.Delete(key); err != nil {
		return err
	}

	fmt.Printf("Entry %q deleted.\n", key)
	return nil
}
