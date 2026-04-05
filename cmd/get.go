package cmd

import (
	"fmt"
	"time"

	"github.com/ChengaDev/persist/internal/config"
	"github.com/ChengaDev/persist/internal/crypto"
	"github.com/ChengaDev/persist/internal/storage"
	"github.com/atotto/clipboard"
	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Retrieve a value and copy it to the clipboard",
	Long: `Retrieves the value for <key> and injects it into the system clipboard.

Sensitive (--secure) values are never printed to stdout — they are
decrypted in memory and written directly to the clipboard, then the
plaintext bytes are zeroed before the process exits.`,
	Args: cobra.ExactArgs(1),
	RunE: runGet,
}

func runGet(_ *cobra.Command, args []string) error {
	key := args[0]

	store, err := storage.Open()
	if err != nil {
		return err
	}
	defer store.Close()

	if err := requireInitialized(store); err != nil {
		return err
	}

	entry, err := store.Get(key)
	if err != nil {
		return withSuggestions(err, key, store)
	}

	var value []byte

	if entry.IsSecure {
		password, err := promptAndVerify(store)
		if err != nil {
			return err
		}
		defer crypto.ZeroBytes(password)

		encKey := crypto.DeriveKey(password, entry.Salt)
		defer crypto.ZeroBytes(encKey)

		plaintext, err := crypto.Decrypt(encKey, entry.Nonce, entry.Value)
		if err != nil {
			return err
		}
		defer crypto.ZeroBytes(plaintext)
		value = plaintext
	} else {
		value = entry.Value
	}

	// string(value) creates an immutable Go string that cannot be zeroed.
	// This is an unavoidable limitation of the clipboard API; the string
	// will persist in memory until the GC collects it. The plaintext []byte
	// is still zeroed via defer above to minimise the exposure window.
	if err := clipboard.WriteAll(string(value)); err != nil {
		return fmt.Errorf("writing to clipboard: %w", err)
	}

	cfg, _ := config.Load()
	if cfg.ClipboardClearAfter > 0 {
		fmt.Printf("Value for %q copied to clipboard. Clearing in %d seconds...\n", key, cfg.ClipboardClearAfter)
		time.Sleep(time.Duration(cfg.ClipboardClearAfter) * time.Second)
		_ = clipboard.WriteAll("")
		fmt.Println("Clipboard cleared.")
	} else {
		fmt.Printf("Value for %q copied to clipboard.\n", key)
	}

	return nil
}
