// Package sshpty provides SSH-based remote PTY sessions. It connects to
// a remote server via SSH key authentication, allocates a pseudo-terminal,
// and exposes Read/Write for bidirectional I/O.
package sshpty

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

const keepAliveInterval = 30 * time.Second

// SSHSession wraps an SSH connection with an allocated PTY.
type SSHSession struct {
	client  *ssh.Client
	session *ssh.Session
	stdin   io.WriteCloser
	stdout  io.Reader

	closeOnce sync.Once
	done      chan struct{}
}

// Dial connects to a remote host via SSH using private key authentication.
// The privateKey should be a PEM-encoded private key (unencrypted or
// decrypted before calling).
func Dial(host string, port int, user string, privateKey []byte) (*SSHSession, error) {
	signer, err := ssh.ParsePrivateKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("sshpty: parse private key: %w", err)
	}

	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("sshpty: dial %s: %w", addr, err)
	}

	session, err := client.NewSession()
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("sshpty: new session: %w", err)
	}

	stdin, err := session.StdinPipe()
	if err != nil {
		session.Close()
		client.Close()
		return nil, fmt.Errorf("sshpty: stdin pipe: %w", err)
	}

	stdout, err := session.StdoutPipe()
	if err != nil {
		session.Close()
		client.Close()
		return nil, fmt.Errorf("sshpty: stdout pipe: %w", err)
	}

	// Merge stderr into stdout so the terminal sees everything.
	session.Stderr = session.Stdout

	s := &SSHSession{
		client:  client,
		session: session,
		stdin:   stdin,
		stdout:  stdout,
		done:    make(chan struct{}),
	}

	go s.keepAlive()

	return s, nil
}

// RequestPTY allocates a pseudo-terminal on the remote side.
func (s *SSHSession) RequestPTY(term string, rows, cols int) error {
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}
	return s.session.RequestPty(term, rows, cols, modes)
}

// Start runs a specific command on the remote server (e.g. "pi", "bash").
func (s *SSHSession) Start(cmd string) error {
	return s.session.Start(cmd)
}

// Shell starts the user's default login shell on the remote server.
func (s *SSHSession) Shell() error {
	return s.session.Shell()
}

// Resize sends a window-change request to update the remote PTY dimensions.
func (s *SSHSession) Resize(rows, cols int) error {
	return s.session.WindowChange(rows, cols)
}

// Read reads from the remote PTY stdout.
func (s *SSHSession) Read(p []byte) (int, error) {
	return s.stdout.Read(p)
}

// Write writes to the remote PTY stdin.
func (s *SSHSession) Write(p []byte) (int, error) {
	return s.stdin.Write(p)
}

// Close tears down the SSH session and connection.
func (s *SSHSession) Close() error {
	var firstErr error
	s.closeOnce.Do(func() {
		close(s.done)
		if err := s.stdin.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
		if err := s.session.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
		if err := s.client.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	})
	return firstErr
}

// Wait blocks until the remote command finishes.
func (s *SSHSession) Wait() error {
	return s.session.Wait()
}

// keepAlive sends periodic keep-alive requests to prevent idle disconnects.
func (s *SSHSession) keepAlive() {
	ticker := time.NewTicker(keepAliveInterval)
	defer ticker.Stop()
	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			// SendRequest on the underlying connection; ignore errors
			// since a failed keepalive will surface as a read/write error.
			_, _, _ = s.client.SendRequest("keepalive@openssh.com", true, nil)
		}
	}
}
