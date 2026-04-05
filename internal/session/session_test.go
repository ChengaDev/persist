package session_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/ChengaDev/persist/internal/session"
)

// setHome redirects os.UserHomeDir() to a temp directory for both Unix and Windows.
func setHome(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir) // Windows
}

func TestFileSetAndGet(t *testing.T) {
	setHome(t)

	password := []byte("super-secret-password")
	if err := session.FileSet(password, 5*time.Minute); err != nil {
		t.Fatal(err)
	}

	got, err := session.FileGet()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, password) {
		t.Fatalf("got %q, want %q", got, password)
	}
}

func TestFileGetReturnsNilWhenNoSession(t *testing.T) {
	setHome(t)

	got, err := session.FileGet()
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatalf("expected nil when no session exists, got %q", got)
	}
}

func TestFileGetReturnsNilAfterExpiry(t *testing.T) {
	setHome(t)

	_ = session.FileSet([]byte("password"), -1*time.Second) // already expired

	got, err := session.FileGet()
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatal("expected nil for expired session")
	}
}

func TestFileClear(t *testing.T) {
	setHome(t)

	_ = session.FileSet([]byte("password"), 5*time.Minute)
	if err := session.FileClear(); err != nil {
		t.Fatal(err)
	}

	got, _ := session.FileGet()
	if got != nil {
		t.Fatal("expected nil after Clear()")
	}
}

func TestFileClearWhenNoSession(t *testing.T) {
	setHome(t)

	// Should not error when no session file exists.
	if err := session.FileClear(); err != nil {
		t.Fatalf("Clear with no session should not error: %v", err)
	}
}
