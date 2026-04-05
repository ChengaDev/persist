package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ChengaDev/persist/internal/config"
	"github.com/ChengaDev/persist/internal/crypto"
	"github.com/ChengaDev/persist/internal/search"
	"github.com/ChengaDev/persist/internal/session"
	"github.com/ChengaDev/persist/internal/storage"
)

// requireInitialized returns a user-friendly error when psst init has not been run.
func requireInitialized(store *storage.Store) error {
	ok, err := store.IsInitialized()
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("psst is not initialized — run `psst init` first")
	}
	return nil
}

const maxPasswordAttempts = 3

// promptAndVerify returns the master password, either from the session cache
// or by prompting the user (with up to maxPasswordAttempts tries).
// The caller is responsible for zeroing the returned bytes with crypto.ZeroBytes.
func promptAndVerify(store *storage.Store) ([]byte, error) {
	canary, err := store.GetCanary()
	if err != nil {
		return nil, fmt.Errorf("reading canary record: %w", err)
	}

	cfg, _ := config.Load() // failure here falls back to defaults (no session)

	// Check session cache first.
	if cfg.SessionTimeout > 0 {
		if cached, err := session.FileGet(); err == nil && cached != nil {
			if crypto.VerifyCanary(cached, canary.Salt, canary.Nonce, canary.Value) {
				return cached, nil
			}
			// Cached password is stale or wrong — clear it and fall through to prompt.
			_ = session.FileClear()
			crypto.ZeroBytes(cached)
		}
	}

	// Prompt with retry loop.
	for attempt := 1; attempt <= maxPasswordAttempts; attempt++ {
		password, err := crypto.PromptPassword("Master password: ")
		if err != nil {
			return nil, err
		}

		if crypto.VerifyCanary(password, canary.Salt, canary.Nonce, canary.Value) {
			// Cache for future commands within the session window.
			if cfg.SessionTimeout > 0 {
				ttl := time.Duration(cfg.SessionTimeout) * time.Minute
				_ = session.FileSet(password, ttl) // best-effort; ignore error
			}
			return password, nil
		}

		crypto.ZeroBytes(password)
		remaining := maxPasswordAttempts - attempt
		if remaining > 0 {
			fmt.Fprintf(os.Stderr, "Incorrect password. %d %s remaining.\n",
				remaining, pluralize(remaining, "attempt", "attempts"))
		}
	}

	return nil, fmt.Errorf("incorrect master password (all %d attempts failed)", maxPasswordAttempts)
}

// withSuggestions wraps a not-found error with fuzzy key suggestions.
func withSuggestions(err error, key string, store *storage.Store) error {
	entries, listErr := store.List()
	if listErr != nil || len(entries) == 0 {
		return err
	}
	keys := make([]string, len(entries))
	for i, e := range entries {
		keys[i] = e.Key
	}
	suggestions := search.Suggest(key, keys, 3)
	if len(suggestions) == 0 {
		return err
	}
	return fmt.Errorf("%w\n\nDid you mean?\n  %s", err, strings.Join(suggestions, "\n  "))
}

func pluralize(n int, singular, plural string) string {
	if n == 1 {
		return singular
	}
	return plural
}
