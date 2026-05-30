package db

import (
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"strings"
	"time"
)

// SSHKey represents a stored SSH key pair.
type SSHKey struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	Name        string    `json:"name"`
	PublicKey   string    `json:"public_key"`
	Fingerprint string    `json:"fingerprint"`
	CreatedAt   time.Time `json:"created_at"`
}

// CreateSSHKey stores a new SSH key pair. The private key is encrypted at rest.
func (d *DB) CreateSSHKey(userID, name string, privateKey, publicKey []byte) (*SSHKey, error) {
	enc, err := encrypt(privateKey)
	if err != nil {
		return nil, fmt.Errorf("db: encrypt key: %w", err)
	}
	id := newID()
	fp := fingerprintMD5(publicKey)
	now := time.Now().UTC()
	_, err = d.db.Exec(
		`INSERT INTO ssh_keys (id, user_id, name, private_key, public_key, fingerprint, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id, userID, name, enc, string(publicKey), fp, now,
	)
	if err != nil {
		return nil, fmt.Errorf("db: insert ssh_key: %w", err)
	}
	return &SSHKey{
		ID:          id,
		UserID:      userID,
		Name:        name,
		PublicKey:   string(publicKey),
		Fingerprint: fp,
		CreatedAt:   now,
	}, nil
}

// ListSSHKeys returns all SSH keys for a user (without private key material).
func (d *DB) ListSSHKeys(userID string) ([]SSHKey, error) {
	rows, err := d.db.Query(
		`SELECT id, user_id, name, public_key, fingerprint, created_at
		 FROM ssh_keys WHERE user_id = ? ORDER BY created_at DESC`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("db: list ssh_keys: %w", err)
	}
	defer rows.Close()
	var keys []SSHKey
	for rows.Next() {
		var k SSHKey
		if err := rows.Scan(&k.ID, &k.UserID, &k.Name, &k.PublicKey, &k.Fingerprint, &k.CreatedAt); err != nil {
			return nil, fmt.Errorf("db: scan ssh_key: %w", err)
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

// GetSSHKey returns a single SSH key by ID, scoped to user.
func (d *DB) GetSSHKey(userID, id string) (*SSHKey, error) {
	var k SSHKey
	err := d.db.QueryRow(
		`SELECT id, user_id, name, public_key, fingerprint, created_at
		 FROM ssh_keys WHERE id = ? AND user_id = ?`, id, userID,
	).Scan(&k.ID, &k.UserID, &k.Name, &k.PublicKey, &k.Fingerprint, &k.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("db: get ssh_key %s: %w", id, err)
	}
	return &k, nil
}

// GetSSHKeyPrivate returns the decrypted private key for a key ID scoped to user.
func (d *DB) GetSSHKeyPrivate(userID, id string) ([]byte, error) {
	var enc []byte
	err := d.db.QueryRow(
		`SELECT private_key FROM ssh_keys WHERE id = ? AND user_id = ?`, id, userID,
	).Scan(&enc)
	if err != nil {
		return nil, fmt.Errorf("db: get ssh_key private %s: %w", id, err)
	}
	return decrypt(enc)
}

// DeleteSSHKey removes an SSH key by ID, scoped to user.
func (d *DB) DeleteSSHKey(userID, id string) error {
	res, err := d.db.Exec(`DELETE FROM ssh_keys WHERE id = ? AND user_id = ?`, id, userID)
	if err != nil {
		return fmt.Errorf("db: delete ssh_key %s: %w", id, err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("db: ssh_key %s not found", id)
	}
	return nil
}

// fingerprintMD5 computes the MD5 fingerprint of an SSH public key,
// returning the colon-separated hex form (e.g. "ab:cd:ef:...").
func fingerprintMD5(pubKey []byte) string {
	parts := strings.Fields(string(pubKey))
	if len(parts) < 2 {
		return ""
	}
	raw, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return ""
	}
	h := md5.Sum(raw)
	segs := make([]string, len(h))
	for i, b := range h {
		segs[i] = fmt.Sprintf("%02x", b)
	}
	return strings.Join(segs, ":")
}
