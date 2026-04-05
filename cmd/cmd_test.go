package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/ChengaDev/persist/internal/crypto"
	"github.com/ChengaDev/persist/internal/storage"
	"github.com/atotto/clipboard"
)

// setHome redirects os.UserHomeDir() to a temp directory for both Unix and Windows.
func setHome(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
}

// setupInitializedStore sets HOME to a temp dir and seeds the database with a
// canary record so commands that call requireInitialized pass.
func setupInitializedStore(t *testing.T, password []byte) {
	t.Helper()
	setHome(t)

	store, err := storage.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	salt, err := crypto.GenerateSalt()
	if err != nil {
		t.Fatal(err)
	}
	key := crypto.DeriveKey(password, salt)
	defer crypto.ZeroBytes(key)

	ct, nonce, err := crypto.Encrypt(key, []byte(crypto.CanaryPlaintext))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveCanary(ct, salt, nonce); err != nil {
		t.Fatal(err)
	}
}

// captureStdout runs fn and returns everything written to os.Stdout during the call.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

// resetSaveFlags resets package-level save flag vars to their defaults between tests.
func resetSaveFlags() {
	saveValue = ""
	saveSecure = false
	saveFromClip = false
	saveForce = false
}

// --- pluralize ---------------------------------------------------------------

func TestPluralize(t *testing.T) {
	if got := pluralize(1, "attempt", "attempts"); got != "attempt" {
		t.Fatalf("got %q, want %q", got, "attempt")
	}
	if got := pluralize(0, "attempt", "attempts"); got != "attempts" {
		t.Fatalf("got %q, want %q", got, "attempts")
	}
	if got := pluralize(3, "attempt", "attempts"); got != "attempts" {
		t.Fatalf("got %q, want %q", got, "attempts")
	}
}

// --- requireInitialized ------------------------------------------------------

func TestRequireInitialized_NotInitialized(t *testing.T) {
	setHome(t)

	store, err := storage.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	if err := requireInitialized(store); err == nil {
		t.Fatal("expected error for uninitialized store, got nil")
	}
}

func TestRequireInitialized_Initialized(t *testing.T) {
	setupInitializedStore(t, []byte("password"))

	store, err := storage.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	if err := requireInitialized(store); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- withSuggestions ---------------------------------------------------------

func TestWithSuggestions_NoKeys(t *testing.T) {
	setupInitializedStore(t, []byte("password"))

	store, err := storage.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	base := fmt.Errorf("key %q not found", "missing")
	wrapped := withSuggestions(base, "missing", store)
	// No entries → original error unchanged.
	if wrapped != base {
		t.Fatalf("expected original error, got %v", wrapped)
	}
}

func TestWithSuggestions_WithMatchingKeys(t *testing.T) {
	setupInitializedStore(t, []byte("password"))

	store, err := storage.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	_ = store.Save("github-token", []byte("val1"), false)
	_ = store.Save("github-ssh", []byte("val2"), false)

	// "github" is a substring of both keys — guaranteed to produce suggestions.
	base := fmt.Errorf("key %q not found", "github")
	wrapped := withSuggestions(base, "github", store)
	if !strings.Contains(wrapped.Error(), "Did you mean?") {
		t.Fatalf("expected suggestions in error, got: %v", wrapped)
	}
}

// --- save --------------------------------------------------------------------

func TestRunSave_PlainValue(t *testing.T) {
	setupInitializedStore(t, []byte("password"))
	resetSaveFlags()
	saveValue = "hello world"

	if err := runSave(saveCmd, []string{"greeting"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	store, _ := storage.Open()
	defer store.Close()

	entry, err := store.Get("greeting")
	if err != nil {
		t.Fatalf("entry not found: %v", err)
	}
	if string(entry.Value) != "hello world" {
		t.Fatalf("got %q, want %q", entry.Value, "hello world")
	}
}

func TestRunSave_Force(t *testing.T) {
	setupInitializedStore(t, []byte("password"))
	resetSaveFlags()
	saveValue = "v1"

	if err := runSave(saveCmd, []string{"key"}); err != nil {
		t.Fatal(err)
	}

	resetSaveFlags()
	saveValue = "v2"
	saveForce = true

	if err := runSave(saveCmd, []string{"key"}); err != nil {
		t.Fatalf("force overwrite failed: %v", err)
	}

	store, _ := storage.Open()
	defer store.Close()
	entry, _ := store.Get("key")
	if string(entry.Value) != "v2" {
		t.Fatalf("got %q, want %q", entry.Value, "v2")
	}
}

func TestRunSave_DuplicateKeyErrors(t *testing.T) {
	setupInitializedStore(t, []byte("password"))
	resetSaveFlags()
	saveValue = "v1"

	_ = runSave(saveCmd, []string{"key"})

	resetSaveFlags()
	saveValue = "v2"

	if err := runSave(saveCmd, []string{"key"}); err == nil {
		t.Fatal("expected duplicate-key error, got nil")
	}
}

func TestRunSave_MissingValueErrors(t *testing.T) {
	setupInitializedStore(t, []byte("password"))
	resetSaveFlags()
	// saveValue is empty, saveFromClip is false, saveSecure is false

	if err := runSave(saveCmd, []string{"key"}); err == nil {
		t.Fatal("expected error for missing value, got nil")
	}
}

func TestRunSave_SecureWithValueFlagErrors(t *testing.T) {
	setupInitializedStore(t, []byte("password"))
	resetSaveFlags()
	saveSecure = true
	saveValue = "oops"

	if err := runSave(saveCmd, []string{"key"}); err == nil {
		t.Fatal("expected error when --secure and --value combined, got nil")
	}
}

// --- list --------------------------------------------------------------------

func TestRunList_Empty(t *testing.T) {
	setupInitializedStore(t, []byte("password"))

	out := captureStdout(t, func() {
		if err := runList(listCmd, nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "No entries found") {
		t.Fatalf("expected empty-list message, got: %q", out)
	}
}

func TestRunList_WithEntries(t *testing.T) {
	setupInitializedStore(t, []byte("password"))
	resetSaveFlags()
	saveValue = "tok"
	_ = runSave(saveCmd, []string{"github-token"})

	out := captureStdout(t, func() {
		if err := runList(listCmd, nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "github-token") {
		t.Fatalf("expected key in list output, got: %q", out)
	}
}

func TestRunList_Filter(t *testing.T) {
	setupInitializedStore(t, []byte("password"))
	resetSaveFlags()
	saveValue = "val"
	_ = runSave(saveCmd, []string{"github-token"})
	resetSaveFlags()
	saveValue = "val"
	_ = runSave(saveCmd, []string{"aws-key"})

	out := captureStdout(t, func() {
		if err := runList(listCmd, []string{"github"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "github-token") {
		t.Fatalf("expected github-token in filtered output, got: %q", out)
	}
	if strings.Contains(out, "aws-key") {
		t.Fatalf("aws-key should be filtered out, got: %q", out)
	}
}

// --- delete ------------------------------------------------------------------

func TestRunDelete_Existing(t *testing.T) {
	setupInitializedStore(t, []byte("password"))
	resetSaveFlags()
	saveValue = "val"
	_ = runSave(saveCmd, []string{"mykey"})

	out := captureStdout(t, func() {
		if err := runDelete(deleteCmd, []string{"mykey"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "deleted") {
		t.Fatalf("expected deleted message, got: %q", out)
	}

	store, _ := storage.Open()
	defer store.Close()
	if _, err := store.Get("mykey"); err == nil {
		t.Fatal("entry still exists after delete")
	}
}

func TestRunDelete_NotFound(t *testing.T) {
	setupInitializedStore(t, []byte("password"))

	if err := runDelete(deleteCmd, []string{"nonexistent"}); err == nil {
		t.Fatal("expected error deleting nonexistent key, got nil")
	}
}

// --- hint --------------------------------------------------------------------

func TestRunHint_NoHint(t *testing.T) {
	setupInitializedStore(t, []byte("password"))

	out := captureStdout(t, func() {
		if err := runHint(hintCmd, nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "No hint set") {
		t.Fatalf("expected no-hint message, got: %q", out)
	}
}

func TestRunHint_WithHint(t *testing.T) {
	setupInitializedStore(t, []byte("password"))

	store, _ := storage.Open()
	_ = store.SaveHint("my secret hint")
	store.Close()

	out := captureStdout(t, func() {
		if err := runHint(hintCmd, nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "my secret hint") {
		t.Fatalf("expected hint in output, got: %q", out)
	}
}

// --- lock --------------------------------------------------------------------

func TestRunLock(t *testing.T) {
	setHome(t)

	out := captureStdout(t, func() {
		if err := runLock(lockCmd, nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "cleared") {
		t.Fatalf("expected cleared message, got: %q", out)
	}
}

// --- get (plain) -------------------------------------------------------------

func TestRunGet_PlainEntry(t *testing.T) {
	if err := clipboard.WriteAll(""); err != nil {
		t.Skipf("clipboard not available: %v", err)
	}
	setupInitializedStore(t, []byte("password"))
	resetSaveFlags()
	saveValue = "clipboard-value"
	_ = runSave(saveCmd, []string{"mykey"})

	out := captureStdout(t, func() {
		if err := runGet(getCmd, []string{"mykey"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "copied to clipboard") {
		t.Fatalf("expected clipboard message, got: %q", out)
	}
}

func TestRunGet_NotFound(t *testing.T) {
	setupInitializedStore(t, []byte("password"))

	if err := runGet(getCmd, []string{"nonexistent"}); err == nil {
		t.Fatal("expected error for missing key, got nil")
	}
}
