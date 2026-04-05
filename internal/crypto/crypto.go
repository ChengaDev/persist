package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/subtle"
	"errors"
	"fmt"
	"io"
	"os"
	"syscall"

	"golang.org/x/crypto/argon2"
	"golang.org/x/term"
)

const (
	// Argon2id parameters (OWASP recommended minimums).
	argonMemory      = 64 * 1024 // 64 MB
	argonIterations  = 3
	argonParallelism = 2
	argonKeyLen      = 32

	// CanaryPlaintext is the known value encrypted in the canary record to
	// validate the master password without storing the password itself.
	CanaryPlaintext = "psst:canary:valid:v1"
)

// GenerateSalt returns 16 cryptographically random bytes.
func GenerateSalt() ([]byte, error) {
	salt := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("generating salt: %w", err)
	}
	return salt, nil
}

// DeriveKey derives a 32-byte AES key from password+salt using Argon2id.
// The call is intentionally slow to resist brute-force attacks.
func DeriveKey(password, salt []byte) []byte {
	return argon2.IDKey(password, salt, argonIterations, argonMemory, argonParallelism, argonKeyLen)
}

// Encrypt encrypts plaintext with AES-256-GCM using the provided 32-byte key.
// Returns (ciphertext, nonce, error). The nonce is randomly generated and must
// be stored alongside the ciphertext for decryption.
func Encrypt(key, plaintext []byte) (ciphertext, nonce []byte, err error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, fmt.Errorf("creating cipher block: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, fmt.Errorf("creating GCM: %w", err)
	}
	nonce = make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, fmt.Errorf("generating nonce: %w", err)
	}
	ciphertext = gcm.Seal(nil, nonce, plaintext, nil)
	return ciphertext, nonce, nil
}

// Decrypt decrypts ciphertext with AES-256-GCM. GCM authentication ensures
// integrity; a wrong key or tampered ciphertext returns an error.
func Decrypt(key, nonce, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("creating cipher block: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("creating GCM: %w", err)
	}
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		// Deliberately opaque — do not leak why decryption failed.
		return nil, errors.New("decryption failed: wrong password or corrupted data")
	}
	return plaintext, nil
}

// VerifyCanary derives the key from password+salt and attempts to decrypt the
// canary ciphertext, comparing the result against CanaryPlaintext in constant
// time. Returns true only if the password is correct.
func VerifyCanary(password, salt, nonce, encryptedCanary []byte) bool {
	key := DeriveKey(password, salt)
	defer ZeroBytes(key)

	plaintext, err := Decrypt(key, nonce, encryptedCanary)
	if err != nil {
		return false
	}
	defer ZeroBytes(plaintext)

	expected := []byte(CanaryPlaintext)
	return subtle.ConstantTimeCompare(plaintext, expected) == 1
}

// PromptPassword writes prompt to stderr and reads a password with echo
// suppressed. Returns an error if the password is empty.
func PromptPassword(prompt string) ([]byte, error) {
	fmt.Fprint(os.Stderr, prompt)
	password, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Fprintln(os.Stderr) // restore newline after hidden input
	if err != nil {
		return nil, fmt.Errorf("reading password: %w", err)
	}
	if len(password) == 0 {
		return nil, errors.New("password cannot be empty")
	}
	return password, nil
}

// PromptPasswordConfirm prompts twice and returns an error unless both entries
// match (compared in constant time to avoid timing leaks).
func PromptPasswordConfirm(prompt, confirmPrompt string) ([]byte, error) {
	p1, err := PromptPassword(prompt)
	if err != nil {
		return nil, err
	}
	p2, err := PromptPassword(confirmPrompt)
	if err != nil {
		ZeroBytes(p1)
		return nil, err
	}
	defer ZeroBytes(p2)

	if subtle.ConstantTimeCompare(p1, p2) != 1 {
		ZeroBytes(p1)
		return nil, errors.New("passwords do not match")
	}
	return p1, nil
}

// ZeroBytes overwrites a byte slice with zeroes to limit the time sensitive
// material lives in process memory.
func ZeroBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
