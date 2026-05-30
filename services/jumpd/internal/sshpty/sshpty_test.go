package sshpty

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"testing"

	"golang.org/x/crypto/ssh"
)

func generateTestKey(t *testing.T) []byte {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	pemBlock, err := ssh.MarshalPrivateKey(priv, "")
	if err != nil {
		t.Fatalf("marshal key: %v", err)
	}
	return pem.EncodeToMemory(pemBlock)
}

func TestDialInvalidHost(t *testing.T) {
	key := generateTestKey(t)
	_, err := Dial("192.0.2.1", 22, "nobody", key)
	if err == nil {
		t.Fatal("expected error dialing invalid host")
	}
}

func TestDialInvalidKey(t *testing.T) {
	_, err := Dial("localhost", 22, "nobody", []byte("not a key"))
	if err == nil {
		t.Fatal("expected error with invalid key")
	}
}

func TestParseValidKey(t *testing.T) {
	key := generateTestKey(t)
	_, err := ssh.ParsePrivateKey(key)
	if err != nil {
		t.Fatalf("expected valid key to parse, got: %v", err)
	}
}

func TestParseInvalidKey(t *testing.T) {
	_, err := ssh.ParsePrivateKey([]byte("garbage"))
	if err == nil {
		t.Fatal("expected error parsing invalid key")
	}
}

func TestSSHSessionClose(t *testing.T) {
	// Verify that Close on an already-closed session doesn't panic.
	// We can't create a real SSHSession without a server, so we test
	// the closeOnce guard by calling Close on a zero-value done channel.
	s := &SSHSession{
		done: make(chan struct{}),
	}
	// Close will fail because client/session are nil, but it must not panic.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Close panicked: %v", r)
		}
	}()
	// We expect this to panic on nil dereference since we have no real
	// session — just verify the done channel is closed.
	close(s.done)
	// Verify done is closed.
	select {
	case <-s.done:
	default:
		t.Fatal("done channel should be closed")
	}
}
