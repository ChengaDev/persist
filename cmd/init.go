package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/ChengaDev/persist/internal/crypto"
	"github.com/ChengaDev/persist/internal/storage"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize psst with a master password",
	Long: `Creates the local database and encrypts a canary record with the master
password. The password itself is never stored — the canary is used to
validate it in future sessions.`,
	Args: cobra.NoArgs,
	RunE: runInit,
}

func runInit(_ *cobra.Command, _ []string) error {
	store, err := storage.Open()
	if err != nil {
		return err
	}
	defer store.Close()

	already, err := store.IsInitialized()
	if err != nil {
		return err
	}
	if already {
		path, _ := storage.DBPath()
		fmt.Fprintf(os.Stderr, "psst is already initialized (%s).\n", path)
		fmt.Fprintln(os.Stderr, "To start over, delete that file and run `psst init` again.")
		return nil
	}

	password, err := crypto.PromptPasswordConfirm("Master password: ", "Confirm master password: ")
	if err != nil {
		return err
	}
	defer crypto.ZeroBytes(password)

	salt, err := crypto.GenerateSalt()
	if err != nil {
		return err
	}

	key := crypto.DeriveKey(password, salt)
	defer crypto.ZeroBytes(key)

	ciphertext, nonce, err := crypto.Encrypt(key, []byte(crypto.CanaryPlaintext))
	if err != nil {
		return err
	}

	if err := store.SaveCanary(ciphertext, salt, nonce); err != nil {
		return err
	}

	fmt.Fprint(os.Stderr, "Password hint (optional, stored in plain text — press Enter to skip): ")
	hint, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	hint = strings.TrimSpace(hint)
	if hint != "" {
		if err := store.SaveHint(hint); err != nil {
			return err
		}
	}

	fmt.Println("psst initialized. Your master password is not stored anywhere.")
	return nil
}
