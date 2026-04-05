package config_test

import (
	"testing"

	"github.com/ChengaDev/persist/internal/config"
)

// setHome redirects os.UserHomeDir() to a temp directory for both Unix and Windows.
func setHome(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir) // Windows
}

func TestLoadDefaultsWhenNoFile(t *testing.T) {
	setHome(t)

	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.SessionTimeout != 0 {
		t.Fatalf("expected SessionTimeout=0, got %d", cfg.SessionTimeout)
	}
	if cfg.ClipboardClearAfter != 0 {
		t.Fatalf("expected ClipboardClearAfter=0, got %d", cfg.ClipboardClearAfter)
	}
	if cfg.DefaultSecure {
		t.Fatal("expected DefaultSecure=false")
	}
}

func TestSaveAndLoad(t *testing.T) {
	setHome(t)

	want := config.Config{
		SessionTimeout:      15,
		ClipboardClearAfter: 45,
		DefaultSecure:       true,
	}

	if err := config.Save(want); err != nil {
		t.Fatal(err)
	}

	got, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.SessionTimeout != want.SessionTimeout {
		t.Fatalf("SessionTimeout: got %d, want %d", got.SessionTimeout, want.SessionTimeout)
	}
	if got.ClipboardClearAfter != want.ClipboardClearAfter {
		t.Fatalf("ClipboardClearAfter: got %d, want %d", got.ClipboardClearAfter, want.ClipboardClearAfter)
	}
	if got.DefaultSecure != want.DefaultSecure {
		t.Fatalf("DefaultSecure: got %v, want %v", got.DefaultSecure, want.DefaultSecure)
	}
}

func TestSaveOverwritesPreviousConfig(t *testing.T) {
	setHome(t)

	_ = config.Save(config.Config{SessionTimeout: 5})
	_ = config.Save(config.Config{SessionTimeout: 20})

	cfg, _ := config.Load()
	if cfg.SessionTimeout != 20 {
		t.Fatalf("expected 20, got %d", cfg.SessionTimeout)
	}
}
