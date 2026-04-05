package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/ChengaDev/persist/internal/search"
	"github.com/ChengaDev/persist/internal/storage"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list [filter]",
	Short: "List all saved keys",
	Long: `Displays a table of all saved keys, their type (Plain/Secure), and creation date.

An optional filter argument performs a case-insensitive substring search:

  psst list github     # shows all keys containing "github"`,
	Args: cobra.MaximumNArgs(1),
	RunE: runList,
}

func runList(_ *cobra.Command, args []string) error {
	store, err := storage.Open()
	if err != nil {
		return err
	}
	defer store.Close()

	if err := requireInitialized(store); err != nil {
		return err
	}

	entries, err := store.List()
	if err != nil {
		return err
	}

	// Apply substring filter when provided.
	if len(args) == 1 && args[0] != "" {
		term := args[0]
		keys := make([]string, len(entries))
		for i, e := range entries {
			keys[i] = e.Key
		}
		matched := search.Filter(term, keys)
		matchSet := make(map[string]bool, len(matched))
		for _, k := range matched {
			matchSet[k] = true
		}
		filtered := entries[:0]
		for _, e := range entries {
			if matchSet[e.Key] {
				filtered = append(filtered, e)
			}
		}
		entries = filtered

		if len(entries) == 0 {
			fmt.Fprintf(os.Stderr, "No keys matching %q.\n", term)
			return nil
		}
	}

	if len(entries) == 0 {
		fmt.Println("No entries found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "KEY\tTYPE\tCREATED")
	fmt.Fprintln(w, "---\t----\t-------")
	for _, e := range entries {
		typ := "Plain"
		if e.IsSecure {
			typ = "Secure"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", e.Key, typ, e.CreatedAt.Format("2006-01-02 15:04:05"))
	}
	return w.Flush()
}
