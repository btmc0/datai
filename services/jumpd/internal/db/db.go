package db

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// DB wraps a sql.DB connected to the datai SQLite database.
type DB struct {
	db *sql.DB
}

// Open opens (or creates) the SQLite database at path, enables WAL mode
// and foreign keys, and runs schema migrations.
func Open(path string) (*DB, error) {
	if path == "" {
		path = os.Getenv("DATAI_DB_PATH")
	}
	if path == "" {
		path = "/data/datai.db"
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("db: mkdir %s: %w", filepath.Dir(path), err)
	}
	sqlDB, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("db: open %s: %w", path, err)
	}
	for _, pragma := range []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA foreign_keys=ON",
		"PRAGMA busy_timeout=5000",
	} {
		if _, err := sqlDB.Exec(pragma); err != nil {
			sqlDB.Close()
			return nil, fmt.Errorf("db: %s: %w", pragma, err)
		}
	}
	if _, err := sqlDB.Exec(schema); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("db: migrate: %w", err)
	}
	return &DB{db: sqlDB}, nil
}

// Close closes the underlying database connection.
func (d *DB) Close() error {
	return d.db.Close()
}

// newID generates a random hex ID (16 bytes = 32 hex chars).
func newID() string {
	b := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		panic("db: crypto/rand unavailable: " + err.Error())
	}
	return hex.EncodeToString(b)
}

// encryptionKey derives a 32-byte AES key from the ENCRYPTION_KEY env var.
func encryptionKey() ([]byte, error) {
	raw := os.Getenv("ENCRYPTION_KEY")
	if raw == "" {
		return nil, fmt.Errorf("db: ENCRYPTION_KEY env not set")
	}
	h := sha256.Sum256([]byte(raw))
	return h[:], nil
}

// encrypt encrypts plaintext with AES-256-GCM using the ENCRYPTION_KEY.
func encrypt(plaintext []byte) ([]byte, error) {
	key, err := encryptionKey()
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("db: aes cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("db: gcm: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("db: nonce: %w", err)
	}
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// decrypt decrypts ciphertext produced by encrypt.
func decrypt(ciphertext []byte) ([]byte, error) {
	key, err := encryptionKey()
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("db: aes cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("db: gcm: %w", err)
	}
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("db: ciphertext too short")
	}
	return gcm.Open(nil, ciphertext[:nonceSize], ciphertext[nonceSize:], nil)
}
