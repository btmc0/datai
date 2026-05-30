package db

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// DataiPeer represents a registered peer (laptop/device) in the datai system.
type DataiPeer struct {
	ID            string  `json:"id"`
	UserID        string  `json:"user_id"`
	Name          string  `json:"name"`
	TailscaleIP   string  `json:"tailscale_ip"`
	TailscaleFQDN string  `json:"tailscale_fqdn,omitempty"`
	Port          int     `json:"port"`
	Status        string  `json:"status"`
	LastSeen      *string `json:"last_seen,omitempty"`
	CreatedAt     string  `json:"created_at"`
}

// PeerInput is the input for creating a peer.
type PeerInput struct {
	Name          string `json:"name"`
	TailscaleIP   string `json:"tailscale_ip"`
	TailscaleFQDN string `json:"tailscale_fqdn,omitempty"`
	Port          int    `json:"port,omitempty"`
}

// ListPeers returns all peers for a user.
func (d *DB) ListPeers(userID string) ([]DataiPeer, error) {
	rows, err := d.db.Query(
		`SELECT id, user_id, name, tailscale_ip, tailscale_fqdn, port, status, last_seen, created_at
		 FROM datai_peers WHERE user_id = ? ORDER BY created_at`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var peers []DataiPeer
	for rows.Next() {
		var p DataiPeer
		if err := rows.Scan(&p.ID, &p.UserID, &p.Name, &p.TailscaleIP, &p.TailscaleFQDN, &p.Port, &p.Status, &p.LastSeen, &p.CreatedAt); err != nil {
			return nil, err
		}
		peers = append(peers, p)
	}
	return peers, rows.Err()
}

// ListAllPeers returns all peers across all users (for startup loading).
func (d *DB) ListAllPeers() ([]DataiPeer, error) {
	rows, err := d.db.Query(
		`SELECT id, user_id, name, tailscale_ip, tailscale_fqdn, port, status, last_seen, created_at
		 FROM datai_peers ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var peers []DataiPeer
	for rows.Next() {
		var p DataiPeer
		if err := rows.Scan(&p.ID, &p.UserID, &p.Name, &p.TailscaleIP, &p.TailscaleFQDN, &p.Port, &p.Status, &p.LastSeen, &p.CreatedAt); err != nil {
			return nil, err
		}
		peers = append(peers, p)
	}
	return peers, rows.Err()
}

// CreatePeer creates a new peer.
func (d *DB) CreatePeer(userID string, input PeerInput) (*DataiPeer, error) {
	id := uuid.New().String()
	port := input.Port
	if port == 0 {
		port = 8790
	}
	_, err := d.db.Exec(
		`INSERT INTO datai_peers (id, user_id, name, tailscale_ip, tailscale_fqdn, port)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		id, userID, input.Name, input.TailscaleIP, input.TailscaleFQDN, port)
	if err != nil {
		return nil, err
	}
	p := &DataiPeer{
		ID:            id,
		UserID:        userID,
		Name:          input.Name,
		TailscaleIP:   input.TailscaleIP,
		TailscaleFQDN: input.TailscaleFQDN,
		Port:          port,
		Status:        "unknown",
		CreatedAt:     time.Now().UTC().Format(time.RFC3339),
	}
	return p, nil
}

// GetPeer returns a peer by ID for a user.
func (d *DB) GetPeer(userID, peerID string) (*DataiPeer, error) {
	var p DataiPeer
	err := d.db.QueryRow(
		`SELECT id, user_id, name, tailscale_ip, tailscale_fqdn, port, status, last_seen, created_at
		 FROM datai_peers WHERE id = ? AND user_id = ?`, peerID, userID).
		Scan(&p.ID, &p.UserID, &p.Name, &p.TailscaleIP, &p.TailscaleFQDN, &p.Port, &p.Status, &p.LastSeen, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// DeletePeer deletes a peer by ID for a user.
func (d *DB) DeletePeer(userID, peerID string) error {
	res, err := d.db.Exec(`DELETE FROM datai_peers WHERE id = ? AND user_id = ?`, peerID, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("db: peer %s not found", peerID)
	}
	return nil
}

// UpdatePeerStatus updates the status and last_seen of a peer.
func (d *DB) UpdatePeerStatus(peerID, status string) error {
	_, err := d.db.Exec(
		`UPDATE datai_peers SET status = ?, last_seen = CURRENT_TIMESTAMP WHERE id = ?`,
		status, peerID)
	return err
}
