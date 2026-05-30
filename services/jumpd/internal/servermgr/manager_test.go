package servermgr

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sting8k/jump/services/jumpd/internal/db"
	"github.com/sting8k/jump/services/jumpd/internal/jwtauth"
	"golang.org/x/crypto/ssh"
)

func testDB(t *testing.T) *db.DB {
	t.Helper()
	dir := t.TempDir()
	os.Setenv("ENCRYPTION_KEY", "test-key-for-servermgr")
	t.Cleanup(func() { os.Unsetenv("ENCRYPTION_KEY") })
	database, err := db.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return database
}

func TestGenerateSSHKey(t *testing.T) {
	database := testDB(t)
	mgr := New(database)

	key, err := mgr.GenerateSSHKey("user-1", "test-key")
	if err != nil {
		t.Fatalf("GenerateSSHKey: %v", err)
	}
	if key.ID == "" {
		t.Error("expected non-empty key ID")
	}
	if key.UserID != "user-1" {
		t.Errorf("user_id = %q, want %q", key.UserID, "user-1")
	}
	if key.Name != "test-key" {
		t.Errorf("name = %q, want %q", key.Name, "test-key")
	}
	if !strings.HasPrefix(key.PublicKey, "ssh-ed25519 ") {
		t.Errorf("public key does not start with ssh-ed25519: %q", key.PublicKey)
	}
	if key.Fingerprint == "" {
		t.Error("expected non-empty fingerprint")
	}

	// Verify private key is valid and can be parsed.
	privBytes, err := database.GetSSHKeyPrivate("user-1", key.ID)
	if err != nil {
		t.Fatalf("GetSSHKeyPrivate: %v", err)
	}
	_, err = ssh.ParsePrivateKey(privBytes)
	if err != nil {
		t.Fatalf("parse stored private key: %v", err)
	}
}

func TestGenerateSSHKeyIsolation(t *testing.T) {
	database := testDB(t)
	mgr := New(database)

	_, err := mgr.GenerateSSHKey("user-1", "key-a")
	if err != nil {
		t.Fatalf("GenerateSSHKey user-1: %v", err)
	}
	_, err = mgr.GenerateSSHKey("user-2", "key-b")
	if err != nil {
		t.Fatalf("GenerateSSHKey user-2: %v", err)
	}

	keys1, _ := database.ListSSHKeys("user-1")
	keys2, _ := database.ListSSHKeys("user-2")
	if len(keys1) != 1 {
		t.Errorf("user-1 keys: got %d, want 1", len(keys1))
	}
	if len(keys2) != 1 {
		t.Errorf("user-2 keys: got %d, want 1", len(keys2))
	}
}

func TestAPIListSSHKeysEmpty(t *testing.T) {
	database := testDB(t)
	mgr := New(database)

	mux := http.NewServeMux()
	mgr.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/v1/datai/ssh-keys", nil)
	req = req.WithContext(context.WithValue(req.Context(), jwtauth.ExportedUserKey, &jwtauth.User{ID: "user-1"}))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var keys []db.SSHKey
	if err := json.NewDecoder(rec.Body).Decode(&keys); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if keys != nil && len(keys) != 0 {
		t.Errorf("expected empty list, got %d", len(keys))
	}
}

func TestAPICreateSSHKey(t *testing.T) {
	database := testDB(t)
	mgr := New(database)

	mux := http.NewServeMux()
	mgr.RegisterRoutes(mux)

	body := strings.NewReader(`{"name":"my-key"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/datai/ssh-keys", body)
	req = req.WithContext(context.WithValue(req.Context(), jwtauth.ExportedUserKey, &jwtauth.User{ID: "user-1"}))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	var key db.SSHKey
	if err := json.NewDecoder(rec.Body).Decode(&key); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if key.Name != "my-key" {
		t.Errorf("name = %q, want %q", key.Name, "my-key")
	}
}

func TestAPICreateServer(t *testing.T) {
	database := testDB(t)
	mgr := New(database)

	mux := http.NewServeMux()
	mgr.RegisterRoutes(mux)

	body := strings.NewReader(`{"name":"prod","host":"10.0.0.1","port":22,"username":"deploy"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/datai/servers", body)
	req = req.WithContext(context.WithValue(req.Context(), jwtauth.ExportedUserKey, &jwtauth.User{ID: "user-1"}))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	var srv db.Server
	if err := json.NewDecoder(rec.Body).Decode(&srv); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if srv.Name != "prod" {
		t.Errorf("name = %q, want %q", srv.Name, "prod")
	}
	if srv.Host != "10.0.0.1" {
		t.Errorf("host = %q, want %q", srv.Host, "10.0.0.1")
	}
}

func TestAPIListServers(t *testing.T) {
	database := testDB(t)
	mgr := New(database)

	_, _ = database.CreateServer("user-1", db.ServerInput{Name: "s1", Host: "h1", Username: "u1"})
	_, _ = database.CreateServer("user-1", db.ServerInput{Name: "s2", Host: "h2", Username: "u2"})
	_, _ = database.CreateServer("user-2", db.ServerInput{Name: "s3", Host: "h3", Username: "u3"})

	mux := http.NewServeMux()
	mgr.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/v1/datai/servers", nil)
	req = req.WithContext(context.WithValue(req.Context(), jwtauth.ExportedUserKey, &jwtauth.User{ID: "user-1"}))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var servers []db.Server
	if err := json.NewDecoder(rec.Body).Decode(&servers); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(servers) != 2 {
		t.Errorf("got %d servers, want 2 (user isolation)", len(servers))
	}
}

func TestAPIListTemplates(t *testing.T) {
	database := testDB(t)
	mgr := New(database)
	_ = database.SeedBuiltinTemplates()

	mux := http.NewServeMux()
	mgr.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/v1/datai/templates", nil)
	req = req.WithContext(context.WithValue(req.Context(), jwtauth.ExportedUserKey, &jwtauth.User{ID: "user-1"}))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var templates []db.PiTemplate
	if err := json.NewDecoder(rec.Body).Decode(&templates); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(templates) < 3 {
		t.Errorf("got %d templates, want at least 3 builtins", len(templates))
	}
}

func TestAPIUnauthorized(t *testing.T) {
	database := testDB(t)
	mgr := New(database)

	mux := http.NewServeMux()
	mgr.RegisterRoutes(mux)

	// No user in context
	req := httptest.NewRequest(http.MethodGet, "/v1/datai/ssh-keys", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}
