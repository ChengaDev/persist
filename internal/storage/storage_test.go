package storage_test

import (
	"path/filepath"
	"testing"

	"github.com/ChengaDev/persist/internal/storage"
)

func newStore(t *testing.T) *storage.Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	store, err := storage.OpenAt(path)
	if err != nil {
		t.Fatalf("opening store: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

// --- IsInitialized / Canary -------------------------------------------------

func TestNotInitializedByDefault(t *testing.T) {
	store := newStore(t)
	ok, err := store.IsInitialized()
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("fresh store should not be initialized")
	}
}

func TestSaveAndVerifyCanary(t *testing.T) {
	store := newStore(t)

	value := []byte("encrypted-canary")
	salt := []byte("saltsaltsaltsalt")
	nonce := []byte("nonce_nonce_")

	if err := store.SaveCanary(value, salt, nonce); err != nil {
		t.Fatal(err)
	}

	ok, err := store.IsInitialized()
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("store should be initialized after SaveCanary")
	}

	canary, err := store.GetCanary()
	if err != nil {
		t.Fatal(err)
	}
	if string(canary.Value) != string(value) {
		t.Fatalf("canary value mismatch: got %q, want %q", canary.Value, value)
	}
	if string(canary.Salt) != string(salt) {
		t.Fatalf("canary salt mismatch")
	}
}

// --- Hint -------------------------------------------------------------------

func TestHintRoundtrip(t *testing.T) {
	store := newStore(t)

	hint, err := store.GetHint()
	if err != nil {
		t.Fatal(err)
	}
	if hint != "" {
		t.Fatal("hint should be empty initially")
	}

	if err := store.SaveHint("my dog's name"); err != nil {
		t.Fatal(err)
	}

	hint, err = store.GetHint()
	if err != nil {
		t.Fatal(err)
	}
	if hint != "my dog's name" {
		t.Fatalf("got %q, want %q", hint, "my dog's name")
	}
}

func TestHintOverwrite(t *testing.T) {
	store := newStore(t)
	_ = store.SaveHint("first")
	_ = store.SaveHint("second")

	hint, _ := store.GetHint()
	if hint != "second" {
		t.Fatalf("expected hint to be overwritten, got %q", hint)
	}
}

// --- Plain entries ----------------------------------------------------------

func TestSavePlainAndGet(t *testing.T) {
	store := newStore(t)

	if err := store.Save("mykey", []byte("myvalue"), false); err != nil {
		t.Fatal(err)
	}

	entry, err := store.Get("mykey")
	if err != nil {
		t.Fatal(err)
	}
	if string(entry.Value) != "myvalue" {
		t.Fatalf("got %q, want %q", entry.Value, "myvalue")
	}
	if entry.IsSecure {
		t.Fatal("plain entry should not be marked secure")
	}
}

func TestSaveDuplicateReturnsError(t *testing.T) {
	store := newStore(t)
	_ = store.Save("key", []byte("v1"), false)

	err := store.Save("key", []byte("v2"), false)
	if err == nil {
		t.Fatal("expected error when saving duplicate key without --force")
	}
}

func TestSaveForceOverwrites(t *testing.T) {
	store := newStore(t)
	_ = store.Save("key", []byte("original"), false)
	_ = store.Save("key", []byte("updated"), true)

	entry, err := store.Get("key")
	if err != nil {
		t.Fatal(err)
	}
	if string(entry.Value) != "updated" {
		t.Fatalf("expected force-overwrite, got %q", entry.Value)
	}
}

// --- Secure entries ---------------------------------------------------------

func TestSaveSecureAndGet(t *testing.T) {
	store := newStore(t)

	value := []byte("encrypted-blob")
	salt := []byte("saltsaltsaltsalt")
	nonce := []byte("nonce_nonce_")

	if err := store.SaveSecure("seckey", value, salt, nonce, false); err != nil {
		t.Fatal(err)
	}

	entry, err := store.Get("seckey")
	if err != nil {
		t.Fatal(err)
	}
	if !entry.IsSecure {
		t.Fatal("entry should be marked secure")
	}
	if string(entry.Value) != string(value) {
		t.Fatalf("value mismatch: got %q, want %q", entry.Value, value)
	}
	if string(entry.Salt) != string(salt) {
		t.Fatal("salt mismatch")
	}
}

func TestSaveSecureDuplicateReturnsError(t *testing.T) {
	store := newStore(t)
	_ = store.SaveSecure("k", []byte("v"), []byte("saltsaltsaltsalt"), []byte("nonce_nonce_"), false)

	err := store.SaveSecure("k", []byte("v2"), []byte("saltsaltsaltsalt"), []byte("nonce_nonce_"), false)
	if err == nil {
		t.Fatal("expected error on duplicate secure entry without --force")
	}
}

// --- Get reserved keys ------------------------------------------------------

func TestGetReservedKeyReturnsError(t *testing.T) {
	store := newStore(t)
	if _, err := store.Get(storage.CanaryKey); err == nil {
		t.Fatal("expected error when getting canary key directly")
	}
	if _, err := store.Get(storage.HintKey); err == nil {
		t.Fatal("expected error when getting hint key directly")
	}
}

// --- Delete -----------------------------------------------------------------

func TestDeleteEntry(t *testing.T) {
	store := newStore(t)
	_ = store.Save("todelete", []byte("value"), false)

	if err := store.Delete("todelete"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get("todelete"); err == nil {
		t.Fatal("entry should be gone after delete")
	}
}

func TestDeleteNonExistentReturnsError(t *testing.T) {
	store := newStore(t)
	if err := store.Delete("ghost"); err == nil {
		t.Fatal("expected error when deleting non-existent key")
	}
}

func TestDeleteReservedKeyReturnsError(t *testing.T) {
	store := newStore(t)
	if err := store.Delete(storage.CanaryKey); err == nil {
		t.Fatal("expected error when deleting canary key")
	}
}

// --- List -------------------------------------------------------------------

func TestListExcludesReservedKeys(t *testing.T) {
	store := newStore(t)
	_ = store.SaveCanary([]byte("c"), []byte("saltsaltsaltsalt"), []byte("nonce_nonce_"))
	_ = store.SaveHint("hint")
	_ = store.Save("user-key", []byte("value"), false)

	entries, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Key != "user-key" {
		t.Fatalf("unexpected key in list: %q", entries[0].Key)
	}
}

func TestListEmpty(t *testing.T) {
	store := newStore(t)
	entries, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected empty list, got %d entries", len(entries))
	}
}
