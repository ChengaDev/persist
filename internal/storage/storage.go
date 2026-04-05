package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const (
	// CanaryKey is the reserved key for the master-password validation record.
	CanaryKey = "__psst_canary__"
	// HintKey is the reserved key for the optional plain-text password hint.
	HintKey = "__psst_hint__"

	dbDir  = ".persist"
	dbFile = "data.db"

	timeLayout = "2006-01-02 15:04:05"
)

// Entry represents one row in the entries table.
type Entry struct {
	ID        int64
	Key       string
	Value     []byte
	IsSecure  bool
	Salt      []byte
	Nonce     []byte
	CreatedAt time.Time
}

// Store wraps an open SQLite database.
type Store struct {
	db *sql.DB
}

// DBPath returns the canonical path to the database file (~/.persist/data.db).
func DBPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home directory: %w", err)
	}
	return filepath.Join(home, dbDir, dbFile), nil
}

// Open opens (or creates) the SQLite database at the default path, applies
// migrations, and returns a ready-to-use Store.
func Open() (*Store, error) {
	path, err := DBPath()
	if err != nil {
		return nil, err
	}
	return OpenAt(path)
}

// OpenAt opens (or creates) a SQLite database at the given path. Intended for
// use in tests where a temporary directory is preferred over ~/.persist.
func OpenAt(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, fmt.Errorf("creating data directory: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("connecting to database: %w", err)
	}

	// Enforce 0600 on the database file regardless of umask.
	if err := os.Chmod(path, 0600); err != nil {
		db.Close()
		return nil, fmt.Errorf("setting database permissions: %w", err)
	}

	// Reject an existing file that has group/other read or write bits set,
	// e.g. one that was created externally with loose permissions.
	if err := requireSafePermissions(path); err != nil {
		db.Close()
		return nil, err
	}

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("running migrations: %w", err)
	}
	return s, nil
}

// requireSafePermissions returns an error if path has any group or other
// read/write/execute bits set. On Windows this check is skipped because
// Windows uses ACLs rather than Unix permission bits.
func requireSafePermissions(path string) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	fi, err := os.Stat(path)
	if err != nil {
		return err
	}
	if fi.Mode()&0077 != 0 {
		return fmt.Errorf(
			"database file %s has unsafe permissions (%o); expected 0600 — fix with: chmod 600 %s",
			path, fi.Mode().Perm(), path,
		)
	}
	return nil
}

// Close releases the database connection.
func (s *Store) Close() error { return s.db.Close() }

// IsInitialized returns true when the canary record exists, indicating that
// psst init has been completed.
func (s *Store) IsInitialized() (bool, error) {
	var n int
	err := s.db.QueryRow("SELECT COUNT(*) FROM entries WHERE key = ?", CanaryKey).Scan(&n)
	return n > 0, err
}

// SaveCanary stores the encrypted canary record, replacing any prior entry.
func (s *Store) SaveCanary(value, salt, nonce []byte) error {
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO entries (key, value, is_secure, salt, nonce)
		 VALUES (?, ?, 1, ?, ?)`,
		CanaryKey, value, salt, nonce,
	)
	return err
}

// GetCanary retrieves the canary record.
func (s *Store) GetCanary() (*Entry, error) {
	return s.getByKey(CanaryKey)
}

// SaveHint stores a plain-text password hint, replacing any prior hint.
func (s *Store) SaveHint(hint string) error {
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO entries (key, value, is_secure) VALUES (?, ?, 0)`,
		HintKey, []byte(hint),
	)
	return err
}

// GetHint returns the stored hint text, or ("", nil) if none was set.
func (s *Store) GetHint() (string, error) {
	var value []byte
	err := s.db.QueryRow("SELECT value FROM entries WHERE key = ?", HintKey).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return string(value), nil
}

// Save writes a plain-text entry. Returns an error if the key already exists
// and force is false.
func (s *Store) Save(key string, value []byte, force bool) error {
	var err error
	if force {
		_, err = s.db.Exec(
			`INSERT OR REPLACE INTO entries (key, value, is_secure) VALUES (?, ?, 0)`,
			key, value,
		)
	} else {
		_, err = s.db.Exec(
			`INSERT INTO entries (key, value, is_secure) VALUES (?, ?, 0)`,
			key, value,
		)
	}
	return interpretInsertErr(key, err)
}

// SaveSecure writes an encrypted entry. Returns an error if the key already
// exists and force is false.
func (s *Store) SaveSecure(key string, value, salt, nonce []byte, force bool) error {
	var err error
	if force {
		_, err = s.db.Exec(
			`INSERT OR REPLACE INTO entries (key, value, is_secure, salt, nonce)
			 VALUES (?, ?, 1, ?, ?)`,
			key, value, salt, nonce,
		)
	} else {
		_, err = s.db.Exec(
			`INSERT INTO entries (key, value, is_secure, salt, nonce)
			 VALUES (?, ?, 1, ?, ?)`,
			key, value, salt, nonce,
		)
	}
	return interpretInsertErr(key, err)
}

// Get retrieves an entry by key, excluding reserved keys.
func (s *Store) Get(key string) (*Entry, error) {
	if key == CanaryKey || key == HintKey {
		return nil, fmt.Errorf("key %q not found", key)
	}
	return s.getByKey(key)
}

// Delete removes an entry. Returns an error if the key does not exist or is
// a reserved key.
func (s *Store) Delete(key string) error {
	if key == CanaryKey || key == HintKey {
		return fmt.Errorf("key %q is reserved and cannot be deleted", key)
	}
	result, err := s.db.Exec("DELETE FROM entries WHERE key = ?", key)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("key %q not found", key)
	}
	return nil
}

// List returns all user entries in reverse-chronological order, never
// including reserved keys.
func (s *Store) List() ([]Entry, error) {
	rows, err := s.db.Query(
		`SELECT id, key, value, is_secure, salt, nonce, created_at
		 FROM entries
		 WHERE key != ? AND key != ?
		 ORDER BY created_at DESC`,
		CanaryKey, HintKey,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []Entry
	for rows.Next() {
		e, err := scanEntry(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// --- internal helpers -------------------------------------------------------

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS entries (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			key        TEXT    UNIQUE NOT NULL,
			value      BLOB    NOT NULL,
			is_secure  INTEGER NOT NULL DEFAULT 0,
			salt       BLOB,
			nonce      BLOB,
			created_at TEXT    DEFAULT (strftime('%Y-%m-%d %H:%M:%S', 'now'))
		)
	`)
	return err
}

func (s *Store) getByKey(key string) (*Entry, error) {
	row := s.db.QueryRow(
		`SELECT id, key, value, is_secure, salt, nonce, created_at
		 FROM entries WHERE key = ?`,
		key,
	)
	e, err := scanEntry(row)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("key %q not found", key)
	}
	if err != nil {
		return nil, err
	}
	return &e, nil
}

// rowScanner abstracts over *sql.Row and *sql.Rows so scanEntry handles both.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanEntry(row rowScanner) (Entry, error) {
	var e Entry
	var isSecureInt int
	var salt, nonce []byte
	var createdAtStr string

	err := row.Scan(&e.ID, &e.Key, &e.Value, &isSecureInt, &salt, &nonce, &createdAtStr)
	if err != nil {
		return Entry{}, err
	}

	e.IsSecure = isSecureInt != 0
	e.Salt = salt
	e.Nonce = nonce

	if t, parseErr := time.Parse(timeLayout, createdAtStr); parseErr == nil {
		e.CreatedAt = t
	}
	return e, nil
}

func interpretInsertErr(key string, err error) error {
	if err == nil {
		return nil
	}
	if strings.Contains(err.Error(), "UNIQUE constraint failed") {
		return fmt.Errorf("key %q already exists; use --force to overwrite", key)
	}
	return err
}
