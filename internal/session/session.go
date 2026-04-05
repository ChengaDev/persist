// Package session handles short-lived caching of the master password so that
// repeated psst commands within a session window skip the password prompt.
//
// The password is written to ~/.persist/.session (chmod 600) as a JSON file
// containing a base64-encoded password field and an expiry timestamp.
// The file is deleted automatically when the session expires.
//
// Security note: the password is briefly held in memory as a []byte during
// read/write. All []byte values are zeroed after use. Go strings created
// internally by encoding/json are immutable and cannot be zeroed — this is a
// known Go limitation; the window is minimised by keeping strings short-lived.
package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ChengaDev/persist/internal/crypto"
)

// fileSession is the on-disk representation. Using []byte for Password causes
// encoding/json to base64-encode it automatically, matching the prior string
// format on disk while allowing us to zero the slice after use.
type fileSession struct {
	Password  []byte    `json:"password"`
	ExpiresAt time.Time `json:"expires_at"`
}

func sessionFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".persist", ".session"), nil
}

// FileGet returns the cached password from the session file, or nil if none
// exists or the session has expired (expired file is removed automatically).
func FileGet() ([]byte, error) {
	path, err := sessionFilePath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer crypto.ZeroBytes(data)

	var s fileSession
	if err := json.Unmarshal(data, &s); err != nil {
		_ = os.Remove(path)
		return nil, nil
	}
	defer crypto.ZeroBytes(s.Password)

	if time.Now().After(s.ExpiresAt) {
		_ = os.Remove(path)
		return nil, nil
	}

	// Return a copy so the caller owns the bytes independently of s.Password.
	result := make([]byte, len(s.Password))
	copy(result, s.Password)
	return result, nil
}

// FileSet atomically writes the password to the session file with the given TTL.
func FileSet(password []byte, ttl time.Duration) error {
	path, err := sessionFilePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("creating session directory: %w", err)
	}

	s := fileSession{
		Password:  password,
		ExpiresAt: time.Now().Add(ttl),
	}
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	defer crypto.ZeroBytes(data)

	// Atomic write: write to a temp file then rename so readers never see
	// a partial file. Clean up the temp file if rename fails.
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// FileClear removes the session file.
func FileClear() error {
	path, err := sessionFilePath()
	if err != nil {
		return err
	}
	err = os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
