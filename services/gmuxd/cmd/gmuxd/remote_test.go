package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gmuxapp/gmux/services/gmuxd/internal/config"
)

func TestEnableTailscaleConfig_NewFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "host.toml")

	if err := enableTailscaleConfig(cfgPath); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, "[remote]") {
		t.Errorf("missing [remote] section in:\n%s", content)
	}
	if !strings.Contains(content, `mode = "tsnet"`) {
		t.Errorf("missing remote mode in:\n%s", content)
	}
	if !strings.Contains(content, "host-toml") {
		t.Errorf("new file should contain reference link:\n%s", content)
	}
}

func TestEnableTailscaleConfig_ExistingFileNoSection(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "host.toml")
	os.WriteFile(cfgPath, []byte("port = 9999\n"), 0o644)

	if err := enableTailscaleConfig(cfgPath); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(cfgPath)
	content := string(data)
	if !strings.Contains(content, "port = 9999") {
		t.Errorf("original content lost:\n%s", content)
	}
	if !strings.Contains(content, "[remote]") {
		t.Errorf("missing [remote] section:\n%s", content)
	}
	if !strings.Contains(content, `mode = "tsnet"`) {
		t.Errorf("missing remote mode:\n%s", content)
	}
	// Should not prepend the header comment to an existing user file.
	if strings.HasPrefix(content, "# gmux") {
		t.Errorf("should not add header comment to existing file:\n%s", content)
	}
}

func TestEnableTailscaleConfig_ExistingSectionDisabled(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "host.toml")
	os.WriteFile(cfgPath, []byte("[tailscale]\nenabled = false\nhostname = \"mybox\"\n"), 0o644)

	if err := enableTailscaleConfig(cfgPath); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(cfgPath)
	content := string(data)
	if !strings.Contains(content, "[remote]") {
		t.Errorf("missing [remote] section:\n%s", content)
	}
	if !strings.Contains(content, `mode = "tsnet"`) {
		t.Errorf("missing remote mode:\n%s", content)
	}
	if !strings.Contains(content, "hostname = \"mybox\"") {
		t.Errorf("hostname lost:\n%s", content)
	}
}

func TestEnableTailscaleConfig_ExistingSectionNoEnabled(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "host.toml")
	os.WriteFile(cfgPath, []byte("[tailscale]\nhostname = \"mybox\"\n\n[discovery]\ntailscale = true\n"), 0o644)

	if err := enableTailscaleConfig(cfgPath); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(cfgPath)
	content := string(data)
	if !strings.Contains(content, `mode = "tsnet"`) {
		t.Errorf("remote mode not added:\n%s", content)
	}
	if !strings.Contains(content, "hostname = \"mybox\"") {
		t.Errorf("hostname lost:\n%s", content)
	}
	if !strings.Contains(content, "[discovery]") {
		t.Errorf("discovery section lost:\n%s", content)
	}
}

func TestEnableTailscaleConfig_PreservesComments(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "host.toml")
	initial := `# My gmux config
port = 9999

[tailscale]
# Keep this hostname!
hostname = "mybox"
enabled = false  # was disabled

[discovery]
# auto-discover containers
devcontainers = true
`
	os.WriteFile(cfgPath, []byte(initial), 0o644)

	if err := enableTailscaleConfig(cfgPath); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(cfgPath)
	content := string(data)

	// Comments must survive.
	for _, want := range []string{
		"# My gmux config",
		"# Keep this hostname!",
		"# auto-discover containers",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("comment lost %q in:\n%s", want, content)
		}
	}

	// Values must survive.
	if !strings.Contains(content, "port = 9999") {
		t.Errorf("port lost:\n%s", content)
	}
	if !strings.Contains(content, `hostname = "mybox"`) {
		t.Errorf("hostname lost:\n%s", content)
	}

	if !strings.Contains(content, `mode = "tsnet"`) {
		t.Errorf("remote mode not added:\n%s", content)
	}
}

func TestEnableTailscaleConfig_AlreadyEnabled(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "host.toml")
	initial := "[remote]\nmode = \"tsnet\"\n\n[tailscale]\nhostname = \"mybox\"\n"
	os.WriteFile(cfgPath, []byte(initial), 0o644)

	if err := enableTailscaleConfig(cfgPath); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(cfgPath)
	if string(data) != initial {
		t.Errorf("file was modified when already enabled:\n%s", data)
	}
}

// Verify that enableTailscaleConfig produces valid TOML that config.Load
// accepts with tailscale actually enabled. Tests the full round-trip.
func TestEnableTailscaleConfig_ProducesValidConfig(t *testing.T) {
	cases := []struct {
		name    string
		initial string
	}{
		{"empty file", ""},
		{"port only", "port = 9999\n"},
		{"section disabled", "[tailscale]\nenabled = false\nhostname = \"mybox\"\n"},
		{"section no enabled", "[tailscale]\nhostname = \"mybox\"\n"},
		{"no trailing newline", "[tailscale]\nhostname = \"mybox\""},
		{"section header only no newline", "[tailscale]"},
		{"remote section no mode", "[remote]\n"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			cfgDir := filepath.Join(dir, "gmux")
			os.MkdirAll(cfgDir, 0o755)
			cfgPath := filepath.Join(cfgDir, "host.toml")
			if tt.initial != "" {
				os.WriteFile(cfgPath, []byte(tt.initial), 0o644)
			}

			if err := enableTailscaleConfig(cfgPath); err != nil {
				t.Fatal(err)
			}

			t.Setenv("XDG_CONFIG_HOME", dir)
			cfg, err := config.Load()
			if err != nil {
				data, _ := os.ReadFile(cfgPath)
				t.Fatalf("config.Load failed: %v\nfile contents:\n%s", err, data)
			}
			if cfg.Remote.Mode != "tsnet" {
				data, _ := os.ReadFile(cfgPath)
				t.Errorf("remote.mode = %q, want tsnet\nfile contents:\n%s", cfg.Remote.Mode, data)
			}
			if !cfg.Tailscale.Enabled {
				data, _ := os.ReadFile(cfgPath)
				t.Errorf("tailscale should be enabled by remote.mode=tsnet\nfile contents:\n%s", data)
			}
		})
	}
}

func TestDisplayStatus_NeedsLogin(t *testing.T) {
	var stdout bytes.Buffer
	h := &tailscaleHealth{
		Listen: "127.0.0.1:8790",
		TS: &tsHealth{
			AuthURL: "https://login.tailscale.com/a/abc123",
		},
	}
	code := displayStatus(h, &stdout)
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	out := stdout.String()
	if !strings.Contains(out, "https://login.tailscale.com/a/abc123") {
		t.Errorf("missing auth URL in:\n%s", out)
	}
	if !strings.Contains(out, "gmuxd tsnet") {
		t.Errorf("should tell user to run gmuxd tsnet again:\n%s", out)
	}
	// Should NOT mention HTTPS or MagicDNS problems.
	if strings.Contains(out, "HTTPS") || strings.Contains(out, "MagicDNS") {
		t.Errorf("should not mention HTTPS/MagicDNS before login:\n%s", out)
	}
}

func TestDisplayStatus_Connected(t *testing.T) {
	var stdout bytes.Buffer
	h := &tailscaleHealth{
		Listen: "127.0.0.1:8790",
		TS: &tsHealth{
			FQDN:      "gmux.tailnet.ts.net",
			Connected: true,
			HTTPS:     true,
			MagicDNS:  true,
		},
	}
	code := displayStatus(h, &stdout)
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	out := stdout.String()
	if !strings.Contains(out, "https://gmux.tailnet.ts.net") {
		t.Errorf("missing FQDN in:\n%s", out)
	}
	if !strings.Contains(out, "Remote access is active") {
		t.Errorf("missing active message:\n%s", out)
	}
}

func TestDisplayStatus_ConnectedMissingHTTPS(t *testing.T) {
	var stdout bytes.Buffer
	h := &tailscaleHealth{
		Listen: "127.0.0.1:8790",
		TS: &tsHealth{
			FQDN:      "gmux.tailnet.ts.net",
			Connected: true,
			HTTPS:     false,
			MagicDNS:  true,
		},
	}
	code := displayStatus(h, &stdout)
	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	out := stdout.String()
	if !strings.Contains(out, "HTTPS is not enabled") {
		t.Errorf("should warn about HTTPS:\n%s", out)
	}
	if !strings.Contains(out, "login.tailscale.com/admin/dns") {
		t.Errorf("should link to admin console:\n%s", out)
	}
}

func TestDisplayStatus_NotConnected(t *testing.T) {
	var stdout bytes.Buffer
	h := &tailscaleHealth{
		Listen: "127.0.0.1:8790",
		TS: &tsHealth{
			Connected: false,
		},
	}
	code := displayStatus(h, &stdout)
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	out := stdout.String()
	if !strings.Contains(out, "still connecting") {
		t.Errorf("should say still connecting:\n%s", out)
	}
	// Should NOT mention HTTPS or MagicDNS problems.
	if strings.Contains(out, "HTTPS") || strings.Contains(out, "MagicDNS") {
		t.Errorf("should not mention HTTPS/MagicDNS when not connected:\n%s", out)
	}
}

func TestRunTsnetRelayConfiguredDoesNotStartTailscaleSetup(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	cfgDir := filepath.Join(dir, "gmux")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "host.toml"), []byte(`
[remote]
mode = "relay"

[relay]
url = "wss://relay.example.com/_gmux/agent"
token = "secret"
`), 0o644); err != nil {
		t.Fatal(err)
	}
	stdin := strings.NewReader("y\n")
	var stdout, stderr bytes.Buffer
	code := runTsnet(stdin, &stdout, &stderr)
	if code != 0 {
		t.Errorf("exit code = %d, want 0; stderr=%s", code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "Relay access is configured") {
		t.Errorf("missing relay configured message:\n%s", out)
	}
	if strings.Contains(out, "Enable remote access?") {
		t.Errorf("should not prompt for Tailscale setup when relay is configured:\n%s", out)
	}
}

func TestRunRelayConfigured(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	cfgDir := filepath.Join(dir, "gmux")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "host.toml"), []byte(`
[remote]
mode = "relay"

[relay]
url = "wss://relay.example.com/_gmux/agent"
token = "secret"
`), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := runRelay(&stdout, &stderr)
	if code != 0 {
		t.Errorf("exit code = %d, want 0; stderr=%s", code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "Relay access is configured") {
		t.Errorf("missing relay configured message:\n%s", out)
	}
	if !strings.Contains(out, "wss://relay.example.com/_gmux/agent") {
		t.Errorf("missing relay URL:\n%s", out)
	}
}

func TestRemoteSetup_UserDeclinesNoConfigChange(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	stdin := strings.NewReader("n\n")
	var stdout, stderr bytes.Buffer
	code := remoteSetup(defaultConfig(), stdin, &stdout, &stderr)
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}

	// Config file should not have been created.
	cfgPath := filepath.Join(dir, "gmux", "host.toml")
	if _, err := os.Stat(cfgPath); !os.IsNotExist(err) {
		t.Errorf("config file should not exist after declining, err=%v", err)
	}
}

func TestRemoteSetup_ShowsExplanation(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	stdin := strings.NewReader("n\n")
	var stdout, stderr bytes.Buffer
	remoteSetup(defaultConfig(), stdin, &stdout, &stderr)

	out := stdout.String()
	if !strings.Contains(out, "Tailscale") {
		t.Errorf("should mention Tailscale:\n%s", out)
	}
	if !strings.Contains(out, remoteDocsURL) {
		t.Errorf("should link to docs:\n%s", out)
	}
	if !strings.Contains(out, "[y/N]") {
		t.Errorf("should show confirmation prompt:\n%s", out)
	}
}

func defaultConfig() config.Config {
	return config.Config{Port: 8790}
}
