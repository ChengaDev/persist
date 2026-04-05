package cmd

import (
	"fmt"
	"os"

	"github.com/ChengaDev/persist/internal/config"
	"github.com/ChengaDev/persist/internal/crypto"
	"github.com/ChengaDev/persist/internal/storage"
	"github.com/atotto/clipboard"
	"github.com/spf13/cobra"
)

var (
	saveValue    string
	saveSecure   bool
	saveFromClip bool
	saveForce    bool
)

var saveCmd = &cobra.Command{
	Use:   "save <key>",
	Short: "Save a key-value pair",
	Long: `Saves a key-value pair to the local database.

Value sources (pick one):
  --value <val>   Inline value (plain entries only — never use with --secure)
  --from-clip     Read value from the system clipboard
  (neither)       For --secure entries, psst prompts with hidden input

Flags:
  --secure        Encrypt the value with AES-256-GCM + Argon2id key derivation
  --force         Overwrite the entry if it already exists`,
	Args: cobra.ExactArgs(1),
	RunE: runSave,
}

func init() {
	saveCmd.Flags().StringVarP(&saveValue, "value", "v", "", "Value to store (plain entries only)")
	saveCmd.Flags().BoolVarP(&saveSecure, "secure", "s", false, "Encrypt the value")
	saveCmd.Flags().BoolVarP(&saveFromClip, "from-clip", "c", false, "Read value from clipboard")
	saveCmd.Flags().BoolVarP(&saveForce, "force", "f", false, "Overwrite existing entry")

	// --value + --from-clip are ambiguous; handled in runSave with a clear message.
	saveCmd.MarkFlagsMutuallyExclusive("value", "from-clip")
}

func runSave(cmd *cobra.Command, args []string) error {
	key := args[0]

	store, err := storage.Open()
	if err != nil {
		return err
	}
	defer store.Close()

	if err := requireInitialized(store); err != nil {
		return err
	}

	// Apply default_secure config unless the user explicitly passed a flag.
	if !saveSecure && !cmd.Flags().Changed("secure") {
		cfg, _ := config.Load()
		saveSecure = cfg.DefaultSecure
	}

	// --- validate flags ------------------------------------------------------

	if saveSecure && saveValue != "" {
		return fmt.Errorf(
			"--value cannot be used with --secure: passing secrets as CLI flags leaks them into shell history.\n" +
				"Use --from-clip (copy the secret first) or omit --value to be prompted with hidden input.",
		)
	}

	// --- resolve value -------------------------------------------------------

	var value []byte

	switch {
	case saveFromClip:
		text, err := clipboard.ReadAll()
		if err != nil {
			return fmt.Errorf("reading clipboard: %w", err)
		}
		if text == "" {
			return fmt.Errorf("clipboard is empty")
		}
		value = []byte(text)

	case saveSecure:
		// Prompt with hidden input — the only safe way to accept a secret
		// without it appearing in the process argument list or shell history.
		v, err := crypto.PromptPassword("Secret value (hidden): ")
		if err != nil {
			return err
		}
		value = v

	default:
		if saveValue == "" {
			return fmt.Errorf("provide a value via --value <val> or --from-clip")
		}
		value = []byte(saveValue)
	}

	// --- persist -------------------------------------------------------------

	if saveSecure {
		defer crypto.ZeroBytes(value)

		password, err := promptAndVerify(store)
		if err != nil {
			return err
		}
		defer crypto.ZeroBytes(password)

		salt, err := crypto.GenerateSalt()
		if err != nil {
			return err
		}

		encKey := crypto.DeriveKey(password, salt)
		defer crypto.ZeroBytes(encKey)

		ciphertext, nonce, err := crypto.Encrypt(encKey, value)
		if err != nil {
			return err
		}

		if err := store.SaveSecure(key, ciphertext, salt, nonce, saveForce); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Secure entry %q saved.\n", key)
	} else {
		if err := store.Save(key, value, saveForce); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Entry %q saved.\n", key)
	}

	return nil
}
