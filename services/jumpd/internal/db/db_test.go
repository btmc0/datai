package db

import (
	"os"
	"path/filepath"
	"testing"
)

func testDB(t *testing.T) *DB {
	t.Helper()
	t.Setenv("ENCRYPTION_KEY", "test-secret-key-for-unit-tests")
	path := filepath.Join(t.TempDir(), "test.db")
	d, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { d.Close() })
	return d
}

func TestOpenCreatesDB(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "dir", "test.db")
	t.Setenv("ENCRYPTION_KEY", "test-key")
	d, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer d.Close()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("db file not created: %v", err)
	}
}

func TestSSHKeyCRUD(t *testing.T) {
	d := testDB(t)
	userID := "user-1"

	// Create
	pub := []byte("ssh-rsa AAAAB3NzaC1yc2EAAAADAQAB test@host")
	priv := []byte("-----BEGIN OPENSSH PRIVATE KEY-----\nfake-key-data\n-----END OPENSSH PRIVATE KEY-----")
	key, err := d.CreateSSHKey(userID, "my-key", priv, pub)
	if err != nil {
		t.Fatalf("CreateSSHKey: %v", err)
	}
	if key.ID == "" {
		t.Fatal("expected non-empty ID")
	}
	if key.Name != "my-key" {
		t.Fatalf("name = %q, want %q", key.Name, "my-key")
	}

	// List
	keys, err := d.ListSSHKeys(userID)
	if err != nil {
		t.Fatalf("ListSSHKeys: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("len = %d, want 1", len(keys))
	}
	if keys[0].ID != key.ID {
		t.Fatalf("ID mismatch")
	}

	// Get
	got, err := d.GetSSHKey(userID, key.ID)
	if err != nil {
		t.Fatalf("GetSSHKey: %v", err)
	}
	if got.PublicKey != string(pub) {
		t.Fatalf("public key mismatch")
	}

	// Decrypt private key roundtrip
	decrypted, err := d.GetSSHKeyPrivate(userID, key.ID)
	if err != nil {
		t.Fatalf("GetSSHKeyPrivate: %v", err)
	}
	if string(decrypted) != string(priv) {
		t.Fatalf("private key roundtrip failed:\n  got:  %q\n  want: %q", decrypted, priv)
	}

	// Isolation: other user can't see key
	keys2, err := d.ListSSHKeys("user-2")
	if err != nil {
		t.Fatalf("ListSSHKeys user-2: %v", err)
	}
	if len(keys2) != 0 {
		t.Fatalf("user-2 should see 0 keys, got %d", len(keys2))
	}

	// Delete
	if err := d.DeleteSSHKey(userID, key.ID); err != nil {
		t.Fatalf("DeleteSSHKey: %v", err)
	}
	keys, _ = d.ListSSHKeys(userID)
	if len(keys) != 0 {
		t.Fatalf("after delete: len = %d, want 0", len(keys))
	}
}

func TestServerCRUD(t *testing.T) {
	d := testDB(t)
	userID := "user-1"

	// Create
	srv, err := d.CreateServer(userID, ServerInput{
		Name:     "prod-1",
		Host:     "10.0.0.1",
		Port:     22,
		Username: "deploy",
	})
	if err != nil {
		t.Fatalf("CreateServer: %v", err)
	}
	if srv.ID == "" {
		t.Fatal("expected non-empty ID")
	}

	// List
	servers, err := d.ListServers(userID)
	if err != nil {
		t.Fatalf("ListServers: %v", err)
	}
	if len(servers) != 1 {
		t.Fatalf("len = %d, want 1", len(servers))
	}

	// Get
	got, err := d.GetServer(userID, srv.ID)
	if err != nil {
		t.Fatalf("GetServer: %v", err)
	}
	if got.Host != "10.0.0.1" {
		t.Fatalf("host = %q, want %q", got.Host, "10.0.0.1")
	}

	// Update
	if err := d.UpdateServer(userID, srv.ID, ServerInput{
		Name:     "prod-1-updated",
		Host:     "10.0.0.2",
		Port:     2222,
		Username: "admin",
	}); err != nil {
		t.Fatalf("UpdateServer: %v", err)
	}
	got, _ = d.GetServer(userID, srv.ID)
	if got.Host != "10.0.0.2" || got.Port != 2222 {
		t.Fatalf("update failed: host=%q port=%d", got.Host, got.Port)
	}

	// Pi status
	if err := d.UpdateServerPiStatus(userID, srv.ID, true, "1.2.3"); err != nil {
		t.Fatalf("UpdateServerPiStatus: %v", err)
	}
	got, _ = d.GetServer(userID, srv.ID)
	if !got.PiInstalled || got.PiVersion != "1.2.3" {
		t.Fatalf("pi status: installed=%v version=%q", got.PiInstalled, got.PiVersion)
	}

	// Delete
	if err := d.DeleteServer(userID, srv.ID); err != nil {
		t.Fatalf("DeleteServer: %v", err)
	}
	servers, _ = d.ListServers(userID)
	if len(servers) != 0 {
		t.Fatalf("after delete: len = %d, want 0", len(servers))
	}
}

func TestPiConfigCRUD(t *testing.T) {
	d := testDB(t)
	userID := "user-1"

	srv, err := d.CreateServer(userID, ServerInput{
		Name: "test", Host: "1.2.3.4", Username: "root",
	})
	if err != nil {
		t.Fatalf("CreateServer: %v", err)
	}

	// Save config
	cfg, err := d.SavePiConfig(srv.ID, PiConfigInput{
		ConfigType: "system_prompt",
		Name:       "default",
		Content:    "You are a helpful assistant.",
		RemotePath: "~/.config/pi/system-prompt.md",
	})
	if err != nil {
		t.Fatalf("SavePiConfig: %v", err)
	}
	if cfg.ID == "" {
		t.Fatal("expected non-empty ID")
	}

	// List
	configs, err := d.ListPiConfigs(srv.ID)
	if err != nil {
		t.Fatalf("ListPiConfigs: %v", err)
	}
	if len(configs) != 1 {
		t.Fatalf("len = %d, want 1", len(configs))
	}
	if configs[0].Content != "You are a helpful assistant." {
		t.Fatalf("content mismatch")
	}

	// Update
	if err := d.UpdatePiConfig(cfg.ID, PiConfigInput{
		ConfigType: "system_prompt",
		Name:       "default",
		Content:    "Updated prompt.",
		RemotePath: "~/.config/pi/system-prompt.md",
	}); err != nil {
		t.Fatalf("UpdatePiConfig: %v", err)
	}
	configs, _ = d.ListPiConfigs(srv.ID)
	if configs[0].Content != "Updated prompt." {
		t.Fatalf("update content mismatch: %q", configs[0].Content)
	}

	// Mark synced
	if err := d.MarkSynced(cfg.ID); err != nil {
		t.Fatalf("MarkSynced: %v", err)
	}
	configs, _ = d.ListPiConfigs(srv.ID)
	if configs[0].SyncedAt == nil {
		t.Fatal("synced_at should be set")
	}

	// Delete
	if err := d.DeletePiConfig(cfg.ID); err != nil {
		t.Fatalf("DeletePiConfig: %v", err)
	}
	configs, _ = d.ListPiConfigs(srv.ID)
	if len(configs) != 0 {
		t.Fatalf("after delete: len = %d, want 0", len(configs))
	}
}

func TestPiTemplateSeedAndList(t *testing.T) {
	d := testDB(t)

	if err := d.SeedBuiltinTemplates(); err != nil {
		t.Fatalf("SeedBuiltinTemplates: %v", err)
	}

	// Seed again is idempotent
	if err := d.SeedBuiltinTemplates(); err != nil {
		t.Fatalf("SeedBuiltinTemplates (2nd): %v", err)
	}

	// List as user should see builtins
	templates, err := d.ListTemplates("any-user")
	if err != nil {
		t.Fatalf("ListTemplates: %v", err)
	}
	if len(templates) != 3 {
		t.Fatalf("len = %d, want 3 builtins", len(templates))
	}
	for _, tmpl := range templates {
		if !tmpl.IsBuiltin {
			t.Fatalf("expected builtin, got custom: %s", tmpl.Name)
		}
	}

	// Create custom template
	custom, err := d.CreateTemplate("user-1", TemplateInput{
		Name:       "My Custom",
		Description: "Custom template",
		ConfigData: `{"system_prompt":"custom","skills":[],"settings":{}}`,
	})
	if err != nil {
		t.Fatalf("CreateTemplate: %v", err)
	}

	// user-1 sees 3 builtins + 1 custom
	templates, _ = d.ListTemplates("user-1")
	if len(templates) != 4 {
		t.Fatalf("user-1: len = %d, want 4", len(templates))
	}

	// user-2 sees only 3 builtins
	templates, _ = d.ListTemplates("user-2")
	if len(templates) != 3 {
		t.Fatalf("user-2: len = %d, want 3", len(templates))
	}

	// Delete custom
	if err := d.DeleteTemplate("user-1", custom.ID); err != nil {
		t.Fatalf("DeleteTemplate: %v", err)
	}
	templates, _ = d.ListTemplates("user-1")
	if len(templates) != 3 {
		t.Fatalf("after delete: len = %d, want 3", len(templates))
	}

	// Cannot delete builtin
	if err := d.DeleteTemplate("user-1", "builtin-coding-assistant"); err == nil {
		t.Fatal("expected error deleting builtin template")
	}
}

func TestEncryptionKeyMissing(t *testing.T) {
	t.Setenv("ENCRYPTION_KEY", "")
	path := filepath.Join(t.TempDir(), "test.db")
	d, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer d.Close()

	_, err = d.CreateSSHKey("user-1", "test", []byte("private"), []byte("ssh-rsa AAAA test"))
	if err == nil {
		t.Fatal("expected error when ENCRYPTION_KEY not set")
	}
}

func TestConversationCRUD(t *testing.T) {
	d := testDB(t)
	userID := "user-1"

	// Create server (needed for session references)
	srv, err := d.CreateServer(userID, ServerInput{
		Name: "test", Host: "1.2.3.4", Username: "root",
	})
	if err != nil {
		t.Fatalf("CreateServer: %v", err)
	}

	// Create conversation
	conv, err := d.CreateConversation(userID, ConversationInput{Name: "workspace-1"})
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}
	if conv.ID == "" {
		t.Fatal("expected non-empty ID")
	}
	if conv.Name != "workspace-1" {
		t.Fatalf("name = %q, want %q", conv.Name, "workspace-1")
	}

	// List
	convs, err := d.ListConversations(userID)
	if err != nil {
		t.Fatalf("ListConversations: %v", err)
	}
	if len(convs) != 1 {
		t.Fatalf("len = %d, want 1", len(convs))
	}

	// Get
	got, err := d.GetConversation(userID, conv.ID)
	if err != nil {
		t.Fatalf("GetConversation: %v", err)
	}
	if got.Name != "workspace-1" {
		t.Fatalf("name = %q, want %q", got.Name, "workspace-1")
	}
	if len(got.Sessions) != 0 {
		t.Fatalf("sessions = %d, want 0", len(got.Sessions))
	}

	// Add sessions
	if err := d.AddConversationSession(conv.ID, ConversationSessionInput{
		SessionID: "sess-1", ServerID: srv.ID, Position: 0, WidthPercent: 50.0,
	}); err != nil {
		t.Fatalf("AddConversationSession: %v", err)
	}
	if err := d.AddConversationSession(conv.ID, ConversationSessionInput{
		SessionID: "sess-2", ServerID: srv.ID, Position: 1, WidthPercent: 50.0,
	}); err != nil {
		t.Fatalf("AddConversationSession (2): %v", err)
	}

	got, _ = d.GetConversation(userID, conv.ID)
	if len(got.Sessions) != 2 {
		t.Fatalf("sessions = %d, want 2", len(got.Sessions))
	}
	if got.Sessions[0].SessionID != "sess-1" || got.Sessions[1].SessionID != "sess-2" {
		t.Fatalf("session order wrong: %v", got.Sessions)
	}

	// Update layout
	if err := d.UpdateConversationSession(conv.ID, "sess-1", ConversationSessionUpdate{
		Position: 0, WidthPercent: 70.0,
	}); err != nil {
		t.Fatalf("UpdateConversationSession: %v", err)
	}
	got, _ = d.GetConversation(userID, conv.ID)
	if got.Sessions[0].WidthPercent != 70.0 {
		t.Fatalf("width = %f, want 70.0", got.Sessions[0].WidthPercent)
	}

	// Remove session
	if err := d.RemoveConversationSession(conv.ID, "sess-1"); err != nil {
		t.Fatalf("RemoveConversationSession: %v", err)
	}
	got, _ = d.GetConversation(userID, conv.ID)
	if len(got.Sessions) != 1 {
		t.Fatalf("after remove: sessions = %d, want 1", len(got.Sessions))
	}

	// Update conversation name
	if err := d.UpdateConversation(userID, conv.ID, ConversationInput{Name: "renamed"}); err != nil {
		t.Fatalf("UpdateConversation: %v", err)
	}
	got, _ = d.GetConversation(userID, conv.ID)
	if got.Name != "renamed" {
		t.Fatalf("name = %q, want %q", got.Name, "renamed")
	}

	// User isolation
	convs2, err := d.ListConversations("user-2")
	if err != nil {
		t.Fatalf("ListConversations user-2: %v", err)
	}
	if len(convs2) != 0 {
		t.Fatalf("user-2 should see 0 conversations, got %d", len(convs2))
	}

	// Delete (cascade should remove sessions)
	if err := d.DeleteConversation(userID, conv.ID); err != nil {
		t.Fatalf("DeleteConversation: %v", err)
	}
	convs, _ = d.ListConversations(userID)
	if len(convs) != 0 {
		t.Fatalf("after delete: len = %d, want 0", len(convs))
	}
}

func TestSessionLogCRUD(t *testing.T) {
	d := testDB(t)
	userID := "user-1"

	// Create a server for FK reference
	srv, err := d.CreateServer(userID, ServerInput{Name: "test", Host: "1.2.3.4", Username: "root"})
	if err != nil {
		t.Fatalf("CreateServer: %v", err)
	}

	// Append logs
	if err := d.AppendLog(userID, "sess-1", srv.ID, "raw", "hello world", ""); err != nil {
		t.Fatalf("AppendLog: %v", err)
	}
	if err := d.AppendLog(userID, "sess-1", srv.ID, "structured", `{"type":"tool_call"}`, `{"tool":"bash"}`); err != nil {
		t.Fatalf("AppendLog structured: %v", err)
	}
	if err := d.AppendLog(userID, "sess-1", "", "event", "session started", ""); err != nil {
		t.Fatalf("AppendLog event: %v", err)
	}

	// Count
	count, err := d.GetSessionLogCount(userID, "sess-1")
	if err != nil {
		t.Fatalf("GetSessionLogCount: %v", err)
	}
	if count != 3 {
		t.Fatalf("count = %d, want 3", count)
	}

	// Get all
	logs, err := d.GetSessionLogs(userID, "sess-1", "", 100, 0)
	if err != nil {
		t.Fatalf("GetSessionLogs: %v", err)
	}
	if len(logs) != 3 {
		t.Fatalf("len = %d, want 3", len(logs))
	}
	if logs[0].Content != "hello world" || logs[0].LogType != "raw" {
		t.Fatalf("first log: content=%q type=%q", logs[0].Content, logs[0].LogType)
	}

	// Filter by type
	logs, err = d.GetSessionLogs(userID, "sess-1", "structured", 100, 0)
	if err != nil {
		t.Fatalf("GetSessionLogs structured: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("structured len = %d, want 1", len(logs))
	}
	if logs[0].Metadata != `{"tool":"bash"}` {
		t.Fatalf("metadata = %q", logs[0].Metadata)
	}

	// Pagination
	logs, err = d.GetSessionLogs(userID, "sess-1", "", 2, 0)
	if err != nil {
		t.Fatalf("GetSessionLogs paginated: %v", err)
	}
	if len(logs) != 2 {
		t.Fatalf("paginated len = %d, want 2", len(logs))
	}
	logs, err = d.GetSessionLogs(userID, "sess-1", "", 2, 2)
	if err != nil {
		t.Fatalf("GetSessionLogs offset: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("offset len = %d, want 1", len(logs))
	}

	// User isolation
	logs, err = d.GetSessionLogs("user-2", "sess-1", "", 100, 0)
	if err != nil {
		t.Fatalf("GetSessionLogs user-2: %v", err)
	}
	if len(logs) != 0 {
		t.Fatalf("user-2 should see 0 logs, got %d", len(logs))
	}

	// Delete
	if err := d.DeleteSessionLogs(userID, "sess-1"); err != nil {
		t.Fatalf("DeleteSessionLogs: %v", err)
	}
	count, _ = d.GetSessionLogCount(userID, "sess-1")
	if count != 0 {
		t.Fatalf("after delete: count = %d, want 0", count)
	}
}

func TestPeerCRUD(t *testing.T) {
	d := testDB(t)
	userID := "user-1"

	// Create
	p, err := d.CreatePeer(userID, PeerInput{
		Name:          "laptop-1",
		TailscaleIP:   "100.64.0.1",
		TailscaleFQDN: "laptop-1.tail.ts.net",
	})
	if err != nil {
		t.Fatalf("CreatePeer: %v", err)
	}
	if p.Port != 8790 {
		t.Fatalf("default port = %d, want 8790", p.Port)
	}
	if p.Status != "unknown" {
		t.Fatalf("default status = %q, want unknown", p.Status)
	}

	// List
	peers, err := d.ListPeers(userID)
	if err != nil {
		t.Fatalf("ListPeers: %v", err)
	}
	if len(peers) != 1 {
		t.Fatalf("len = %d, want 1", len(peers))
	}
	if peers[0].Name != "laptop-1" {
		t.Fatalf("name = %q, want laptop-1", peers[0].Name)
	}

	// Get
	got, err := d.GetPeer(userID, p.ID)
	if err != nil {
		t.Fatalf("GetPeer: %v", err)
	}
	if got.TailscaleIP != "100.64.0.1" {
		t.Fatalf("ip = %q, want 100.64.0.1", got.TailscaleIP)
	}

	// UpdateStatus
	if err := d.UpdatePeerStatus(p.ID, "online"); err != nil {
		t.Fatalf("UpdatePeerStatus: %v", err)
	}
	got, _ = d.GetPeer(userID, p.ID)
	if got.Status != "online" {
		t.Fatalf("status = %q, want online", got.Status)
	}
	if got.LastSeen == nil {
		t.Fatal("last_seen should be set after UpdatePeerStatus")
	}

	// ListAllPeers
	all, err := d.ListAllPeers()
	if err != nil {
		t.Fatalf("ListAllPeers: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("all len = %d, want 1", len(all))
	}

	// Delete
	if err := d.DeletePeer(userID, p.ID); err != nil {
		t.Fatalf("DeletePeer: %v", err)
	}
	peers, _ = d.ListPeers(userID)
	if len(peers) != 0 {
		t.Fatalf("after delete: len = %d, want 0", len(peers))
	}

	// Delete not found
	if err := d.DeletePeer(userID, "nonexistent"); err == nil {
		t.Fatal("expected error deleting nonexistent peer")
	}

	// User isolation
	p2, _ := d.CreatePeer("user-2", PeerInput{Name: "other", TailscaleIP: "100.64.0.2"})
	if _, err := d.GetPeer(userID, p2.ID); err == nil {
		t.Fatal("user-1 should not see user-2 peer")
	}
}

func TestDefaultPort(t *testing.T) {
	d := testDB(t)
	srv, err := d.CreateServer("user-1", ServerInput{
		Name: "test", Host: "1.2.3.4", Username: "root", Port: 0,
	})
	if err != nil {
		t.Fatalf("CreateServer: %v", err)
	}
	if srv.Port != 22 {
		t.Fatalf("port = %d, want 22", srv.Port)
	}
}
