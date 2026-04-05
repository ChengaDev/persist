package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const configFile = "config.json"

// Config holds all user-configurable settings.
type Config struct {
	// SessionTimeout is how many minutes to cache the master password after
	// the first successful unlock. 0 disables the session entirely.
	SessionTimeout int `json:"session_timeout"`

	// ClipboardClearAfter is how many seconds to wait before wiping the
	// clipboard after a `psst get`. 0 disables auto-clear.
	ClipboardClearAfter int `json:"clipboard_clear_after"`

	// DefaultSecure makes --secure the implicit default on `psst save`.
	DefaultSecure bool `json:"default_secure"`
}

func defaults() Config {
	return Config{
		SessionTimeout:      0,
		ClipboardClearAfter: 0,
		DefaultSecure:       false,
	}
}

// Path returns the path to the config file.
func Path() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home directory: %w", err)
	}
	return filepath.Join(home, ".persist", configFile), nil
}

// Load reads the config file, returning defaults if it does not exist.
func Load() (Config, error) {
	path, err := Path()
	if err != nil {
		return defaults(), err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return defaults(), nil
	}
	if err != nil {
		return defaults(), fmt.Errorf("reading config: %w", err)
	}
	cfg := defaults()
	if err := json.Unmarshal(data, &cfg); err != nil {
		return defaults(), fmt.Errorf("parsing config: %w", err)
	}
	return cfg, nil
}

// Save writes cfg to the config file (chmod 600).
func Save(cfg Config) error {
	path, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}
