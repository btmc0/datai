package servermgr

import (
	"fmt"
	"net"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// PiStatus describes the Pi agent installation state on a remote server.
type PiStatus struct {
	Installed bool   `json:"installed"`
	Version   string `json:"version"`
	Path      string `json:"path"`
}

// CheckPi connects to a server via SSH and checks whether Pi is installed.
func (m *Manager) CheckPi(userID, serverID string) (*PiStatus, error) {
	client, srv, err := m.dialSSH(userID, serverID)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	status := &PiStatus{}

	// Check if pi binary exists.
	pathOut, err := runCommand(client, "which pi 2>/dev/null || true")
	if err != nil {
		return nil, fmt.Errorf("servermgr: check pi path: %w", err)
	}
	path := strings.TrimSpace(pathOut)
	if path == "" {
		// Update DB to reflect Pi is not installed.
		_ = m.db.UpdateServerPiStatus(userID, srv.ID, false, "")
		return status, nil
	}
	status.Installed = true
	status.Path = path

	// Get version.
	verOut, err := runCommand(client, "pi --version 2>/dev/null || true")
	if err == nil {
		status.Version = strings.TrimSpace(verOut)
	}

	_ = m.db.UpdateServerPiStatus(userID, srv.ID, true, status.Version)
	return status, nil
}

// InstallPi connects to a server via SSH and installs Pi using the
// official install script. It updates the DB with the installed version.
func (m *Manager) InstallPi(userID, serverID string) error {
	client, srv, err := m.dialSSH(userID, serverID)
	if err != nil {
		return err
	}
	defer client.Close()

	m.SSE.Broadcast(userID, SSEEvent{
		Type: "pi-install",
		Data: map[string]string{"server_id": serverID, "status": "downloading"},
	})

	_, err = runCommand(client, "curl -fsSL https://pi.dev/install | bash")
	if err != nil {
		m.SSE.Broadcast(userID, SSEEvent{
			Type: "pi-install",
			Data: map[string]string{"server_id": serverID, "status": "failed", "error": err.Error()},
		})
		return fmt.Errorf("servermgr: install pi: %w", err)
	}

	// Verify installation and capture version.
	verOut, _ := runCommand(client, "pi --version 2>/dev/null || true")
	version := strings.TrimSpace(verOut)

	m.SSE.Broadcast(userID, SSEEvent{
		Type: "pi-install",
		Data: map[string]string{"server_id": serverID, "status": "installed", "version": version},
	})

	return m.db.UpdateServerPiStatus(userID, srv.ID, true, version)
}

// dialSSH connects to a server's SSH using the server's assigned key.
// Returns a raw ssh.Client for running commands.
func (m *Manager) dialSSH(userID, serverID string) (*ssh.Client, *serverRef, error) {
	srv, err := m.db.GetServer(userID, serverID)
	if err != nil {
		return nil, nil, fmt.Errorf("servermgr: get server: %w", err)
	}
	if srv.SSHKeyID == "" {
		return nil, nil, fmt.Errorf("servermgr: server %s has no SSH key assigned", serverID)
	}
	privKey, err := m.db.GetSSHKeyPrivate(userID, srv.SSHKeyID)
	if err != nil {
		return nil, nil, fmt.Errorf("servermgr: get ssh key: %w", err)
	}
	signer, err := ssh.ParsePrivateKey(privKey)
	if err != nil {
		return nil, nil, fmt.Errorf("servermgr: parse key: %w", err)
	}
	addr := net.JoinHostPort(srv.Host, fmt.Sprintf("%d", srv.Port))
	config := &ssh.ClientConfig{
		User: srv.Username,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, nil, fmt.Errorf("servermgr: dial %s: %w", addr, err)
	}
	ref := &serverRef{ID: srv.ID, Host: srv.Host, Port: srv.Port, Username: srv.Username}
	return client, ref, nil
}

// serverRef is a lightweight reference to a server for internal use.
type serverRef struct {
	ID       string
	Host     string
	Port     int
	Username string
}

// runCommand executes a command on an SSH client and returns the combined output.
func runCommand(client *ssh.Client, cmd string) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("new session: %w", err)
	}
	defer session.Close()
	out, err := session.CombinedOutput(cmd)
	return string(out), err
}
