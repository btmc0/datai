// Package servermgr provides server and Pi agent management for datai.
// It handles SSH key generation, server health checks, Pi installation,
// and configuration syncing to remote servers.
package servermgr

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"

	"github.com/sting8k/jump/services/jumpd/internal/config"
	"github.com/sting8k/jump/services/jumpd/internal/db"
	"github.com/sting8k/jump/services/jumpd/internal/peering"
	"github.com/sting8k/jump/services/jumpd/internal/sshpty"
	"golang.org/x/crypto/ssh"
)

// Manager coordinates server management, SSH keys, and Pi configuration.
type Manager struct {
	db          *db.DB
	SSE         *SSEHub
	PeerManager *peering.Manager // set by main.go after construction
}

// New creates a Manager backed by the given database.
func New(database *db.DB) *Manager {
	return &Manager{db: database, SSE: NewSSEHub()}
}

// TestConnection verifies SSH connectivity to a server by dialing and
// immediately closing the connection.
func (m *Manager) TestConnection(userID, serverID string) error {
	srv, err := m.db.GetServer(userID, serverID)
	if err != nil {
		return fmt.Errorf("servermgr: get server: %w", err)
	}
	privKey, err := m.db.GetSSHKeyPrivate(userID, srv.SSHKeyID)
	if err != nil {
		return fmt.Errorf("servermgr: get ssh key: %w", err)
	}
	sess, err := sshpty.Dial(srv.Host, srv.Port, srv.Username, privKey)
	if err != nil {
		return fmt.Errorf("servermgr: connect: %w", err)
	}
	return sess.Close()
}

// GenerateSSHKey creates a new Ed25519 SSH key pair and stores it in the DB.
func (m *Manager) GenerateSSHKey(userID, name string) (*db.SSHKey, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("servermgr: generate key: %w", err)
	}

	privPEM, err := ssh.MarshalPrivateKey(priv, "")
	if err != nil {
		return nil, fmt.Errorf("servermgr: marshal private key: %w", err)
	}
	privBytes := pem.EncodeToMemory(privPEM)

	sshPub, err := ssh.NewPublicKey(pub)
	if err != nil {
		return nil, fmt.Errorf("servermgr: marshal public key: %w", err)
	}
	pubBytes := ssh.MarshalAuthorizedKey(sshPub)

	return m.db.CreateSSHKey(userID, name, privBytes, pubBytes)
}

// LoadPeers reads all peers from DB and registers them with PeerManager.
func (m *Manager) LoadPeers() error {
	if m.PeerManager == nil {
		return nil
	}
	peers, err := m.db.ListAllPeers()
	if err != nil {
		return err
	}
	for _, p := range peers {
		m.PeerManager.AddPeer(config.PeerConfig{
			Name: p.Name,
			URL:  fmt.Sprintf("http://%s:%d", p.TailscaleIP, p.Port),
		})
	}
	return nil
}

// dialServer is a helper that looks up a server and dials it via SSH.
func (m *Manager) dialServer(userID, serverID string) (*sshpty.SSHSession, *db.Server, error) {
	srv, err := m.db.GetServer(userID, serverID)
	if err != nil {
		return nil, nil, fmt.Errorf("servermgr: get server: %w", err)
	}
	privKey, err := m.db.GetSSHKeyPrivate(userID, srv.SSHKeyID)
	if err != nil {
		return nil, nil, fmt.Errorf("servermgr: get ssh key: %w", err)
	}
	sess, err := sshpty.Dial(srv.Host, srv.Port, srv.Username, privKey)
	if err != nil {
		return nil, nil, err
	}
	return sess, srv, nil
}
