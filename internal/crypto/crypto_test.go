package crypto_test

import (
	"bytes"
	"testing"

	"github.com/ChengaDev/persist/internal/crypto"
)

func TestGenerateSalt(t *testing.T) {
	s1, err := crypto.GenerateSalt()
	if err != nil {
		t.Fatal(err)
	}
	if len(s1) != 16 {
		t.Fatalf("expected 16 bytes, got %d", len(s1))
	}
	s2, err := crypto.GenerateSalt()
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(s1, s2) {
		t.Fatal("two generated salts should not be identical")
	}
}

func TestDeriveKeyDeterministic(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Argon2id test in short mode")
	}
	password := []byte("test-password")
	salt := []byte("1234567890123456")

	k1 := crypto.DeriveKey(password, salt)
	k2 := crypto.DeriveKey(password, salt)

	if len(k1) != 32 {
		t.Fatalf("expected 32-byte key, got %d", len(k1))
	}
	if !bytes.Equal(k1, k2) {
		t.Fatal("same inputs must produce the same key")
	}
}

func TestDeriveKeyDifferentSalts(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Argon2id test in short mode")
	}
	password := []byte("test-password")
	k1 := crypto.DeriveKey(password, []byte("salt____________"))
	k2 := crypto.DeriveKey(password, []byte("different_salt__"))
	if bytes.Equal(k1, k2) {
		t.Fatal("different salts must produce different keys")
	}
}

func TestEncryptDecryptRoundtrip(t *testing.T) {
	key := bytes.Repeat([]byte{0x42}, 32)
	plaintext := []byte("hello, psst!")

	ciphertext, nonce, err := crypto.Encrypt(key, plaintext)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(ciphertext, plaintext) {
		t.Fatal("ciphertext must differ from plaintext")
	}

	got, err := crypto.Decrypt(key, nonce, ciphertext)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, plaintext) {
		t.Fatalf("roundtrip failed: got %q, want %q", got, plaintext)
	}
}

func TestEncryptProducesUniqueNonces(t *testing.T) {
	key := bytes.Repeat([]byte{0x01}, 32)
	plaintext := []byte("same plaintext")

	_, n1, _ := crypto.Encrypt(key, plaintext)
	_, n2, _ := crypto.Encrypt(key, plaintext)

	if bytes.Equal(n1, n2) {
		t.Fatal("two encryptions must produce different nonces")
	}
}

func TestDecryptWrongKey(t *testing.T) {
	key := bytes.Repeat([]byte{0x01}, 32)
	ciphertext, nonce, _ := crypto.Encrypt(key, []byte("secret"))

	wrongKey := bytes.Repeat([]byte{0xFF}, 32)
	if _, err := crypto.Decrypt(wrongKey, nonce, ciphertext); err == nil {
		t.Fatal("expected decryption to fail with wrong key")
	}
}

func TestDecryptTamperedCiphertext(t *testing.T) {
	key := bytes.Repeat([]byte{0x01}, 32)
	ciphertext, nonce, _ := crypto.Encrypt(key, []byte("secret"))

	ciphertext[0] ^= 0xFF
	if _, err := crypto.Decrypt(key, nonce, ciphertext); err == nil {
		t.Fatal("expected decryption to fail with tampered ciphertext")
	}
}

func TestVerifyCanaryCorrectPassword(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Argon2id test in short mode")
	}
	password := []byte("my-master-password")
	salt, _ := crypto.GenerateSalt()
	key := crypto.DeriveKey(password, salt)

	ct, nonce, err := crypto.Encrypt(key, []byte(crypto.CanaryPlaintext))
	if err != nil {
		t.Fatal(err)
	}

	if !crypto.VerifyCanary(password, salt, nonce, ct) {
		t.Fatal("canary verification should succeed with correct password")
	}
}

func TestVerifyCanaryWrongPassword(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Argon2id test in short mode")
	}
	password := []byte("my-master-password")
	salt, _ := crypto.GenerateSalt()
	key := crypto.DeriveKey(password, salt)

	ct, nonce, _ := crypto.Encrypt(key, []byte(crypto.CanaryPlaintext))

	if crypto.VerifyCanary([]byte("wrong-password"), salt, nonce, ct) {
		t.Fatal("canary verification should fail with wrong password")
	}
}

func TestZeroBytes(t *testing.T) {
	b := []byte{1, 2, 3, 4, 5}
	crypto.ZeroBytes(b)
	for i, v := range b {
		if v != 0 {
			t.Fatalf("byte at index %d not zeroed (got %d)", i, v)
		}
	}
}
