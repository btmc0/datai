package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/toml"

	"github.com/sting8k/jump/packages/paths"
	"github.com/sting8k/jump/services/jumpd/internal/config"
	"github.com/sting8k/jump/services/jumpd/internal/unixipc"
)

const remoteDocsURL = "https://github.com/sting8k/jump/blob/dev/docs/product/remote-access.md"

func runTsnet(stdin io.Reader, stdout, stderr io.Writer) int {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(stderr, "jumpd tsnet: %v\n", err)
		return 1
	}

	if cfg.Remote.Mode == "relay" || (cfg.Relay.Enabled && !cfg.Tailscale.Enabled) {
		fmt.Fprintln(stdout, "Relay access is configured.")
		fmt.Fprintln(stdout, "Use `jumpd relay` to inspect relay configuration.")
		fmt.Fprintf(stdout, "Learn more: %s\n", remoteDocsURL)
		return 0
	}

	if !cfg.Tailscale.Enabled {
		return remoteSetup(cfg, stdin, stdout, stderr)
	}
	return remoteStatus(stdout, stderr)
}

func runRelay(stdin io.Reader, stdout, stderr io.Writer) int {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(stderr, "jumpd relay: %v\n", err)
		return 1
	}

	if !cfg.Relay.Enabled {
		return relaySetup(stdin, stdout, stderr)
	}

	fmt.Fprintln(stdout, "Relay access is configured.")
	if cfg.Remote.PublicURL != "" {
		fmt.Fprintf(stdout, "public url: %s\n", cfg.Remote.PublicURL)
	}
	fmt.Fprintf(stdout, "relay url: %s\n", cfg.Relay.URL)
	return 0
}

func relaySetup(stdin io.Reader, stdout, stderr io.Writer) int {
	fmt.Fprintln(stdout, "Relay access lets this jumpd connect outbound to jump-relayd")
	fmt.Fprintln(stdout, "so browsers can reach it through a public HTTPS/WSS endpoint.")
	fmt.Fprintln(stdout)
	fmt.Fprintf(stdout, "  Learn more: %s\n", remoteDocsURL)
	fmt.Fprintln(stdout)

	reader := bufio.NewReader(stdin)
	fmt.Fprintf(stdout, "Enable relay access? [y/N] ")
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	if answer != "y" && answer != "yes" {
		return 0
	}

	fmt.Fprintf(stdout, "Relay agent URL (wss://.../_jump/agent): ")
	relayURL, _ := reader.ReadString('\n')
	relayURL = strings.TrimSpace(relayURL)

	fmt.Fprintf(stdout, "Relay token: ")
	token, _ := reader.ReadString('\n')
	token = strings.TrimSpace(token)

	fmt.Fprintf(stdout, "Public browser URL (optional): ")
	publicURL, _ := reader.ReadString('\n')
	publicURL = strings.TrimSpace(publicURL)

	cfgPath := config.Path()
	if err := enableRelayConfig(cfgPath, relayURL, token, publicURL); err != nil {
		fmt.Fprintf(stderr, "jumpd relay: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Enabled relay in %s\n", cfgPath)

	fmt.Fprintln(stdout, "Restarting daemon...")
	if code := startBackground(stdout, stderr); code != 0 {
		return code
	}
	fmt.Fprintln(stdout, "Relay access is configured.")
	return 0
}

// remoteSetup explains remote access, asks for confirmation, enables it,
// restarts the daemon, and polls until tailscale reaches a known state.
func remoteSetup(cfg config.Config, stdin io.Reader, stdout, stderr io.Writer) int {
	fmt.Fprintln(stdout, "Remote access lets you connect to this machine's terminal sessions")
	fmt.Fprintln(stdout, "from anywhere using your browser. It works through Tailscale, which")
	fmt.Fprintln(stdout, "creates a private encrypted network between your devices.")
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "You'll need a Tailscale account (free for personal use).")
	fmt.Fprintln(stdout)
	fmt.Fprintf(stdout, "  Learn more: %s\n", remoteDocsURL)
	fmt.Fprintln(stdout)

	fmt.Fprintf(stdout, "Enable remote access? [y/N] ")
	reader := bufio.NewReader(stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	if answer != "y" && answer != "yes" {
		return 0
	}
	fmt.Fprintln(stdout)

	// Enable tailscale in the config file.
	cfgPath := config.Path()
	if err := enableTailscaleConfig(cfgPath); err != nil {
		fmt.Fprintf(stderr, "jumpd tsnet: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Enabled tailscale in %s\n", cfgPath)

	// Restart the daemon so it picks up the new config.
	fmt.Fprintln(stdout, "Restarting daemon...")
	if code := startBackground(stdout, stderr); code != 0 {
		return code
	}

	fmt.Fprintln(stdout)
	return remotePoll(stdout, stderr)
}

// enableTailscaleConfig ensures remote.mode = "tsnet" in the config file.
// Creates the file if it doesn't exist, or appends the section if missing.
//
// Uses the TOML library to parse the file and understand the current state,
// then makes the minimal edit needed. This preserves comments, formatting,
// and all other user content.
func enableTailscaleConfig(cfgPath string) error {
	dir := filepath.Dir(cfgPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("cannot create %s: %w", dir, err)
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("cannot read %s: %w", cfgPath, err)
	}

	// Parse with the TOML library to understand the current state.
	var parsed struct {
		Remote struct {
			Mode string `toml:"mode"`
		} `toml:"remote"`
	}
	md, parseErr := toml.Decode(string(data), &parsed)
	if parseErr != nil {
		return fmt.Errorf("cannot parse %s: %w", cfgPath, parseErr)
	}

	mode := strings.TrimSpace(parsed.Remote.Mode)
	if mode == "tsnet" {
		return nil // already enabled
	}
	if mode != "" {
		return fmt.Errorf("remote.mode is %q; disable it before enabling Tailscale remote access", mode)
	}

	content := string(data)
	// Normalize: ensure trailing newline so regex patterns reliably
	// match line endings (e.g. section header at end of file).
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}

	switch {
	case !md.IsDefined("remote"):
		// No [remote] section: append it.
		if content == "" {
			// New file: add a reference comment.
			content = "# jump daemon configuration\n# Reference: https://github.com/sting8k/jump/blob/dev/apps/website/src/content/docs/reference/host-toml.md\n\n"
		} else {
			// content already ends with \n from normalization above.
			content += "\n"
		}
		content += "[remote]\nmode = \"tsnet\"\n"

	case !md.IsDefined("remote", "mode"):
		// Section exists but no mode key: insert after the header.
		content = insertAfterSection(content, "remote", "mode = \"tsnet\"")
	}

	return os.WriteFile(cfgPath, []byte(content), 0o644)
}

func enableRelayConfig(cfgPath, relayURL, token, publicURL string) error {
	relayURL = strings.TrimSpace(relayURL)
	token = strings.TrimSpace(token)
	publicURL = strings.TrimSpace(publicURL)
	if err := validateRelaySetupInputs(relayURL, token, publicURL); err != nil {
		return err
	}

	dir := filepath.Dir(cfgPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("cannot create %s: %w", dir, err)
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("cannot read %s: %w", cfgPath, err)
	}

	var parsed struct {
		Remote struct {
			Mode string `toml:"mode"`
		} `toml:"remote"`
	}
	md, parseErr := toml.Decode(string(data), &parsed)
	if parseErr != nil {
		return fmt.Errorf("cannot parse %s: %w", cfgPath, parseErr)
	}

	mode := strings.TrimSpace(parsed.Remote.Mode)
	if mode == "tsnet" {
		return fmt.Errorf("remote.mode is %q; disable it before enabling relay access", mode)
	}

	content := string(data)
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}

	remoteModeLine := `mode = "relay"`
	publicURLLine := "public_url = " + strconv.Quote(publicURL)
	switch {
	case !md.IsDefined("remote"):
		if content == "" {
			content = "# jump daemon configuration\n# Reference: https://github.com/sting8k/jump/blob/dev/apps/website/src/content/docs/reference/host-toml.md\n\n"
		} else {
			content += "\n"
		}
		content += "[remote]\n" + remoteModeLine + "\n"
		if publicURL != "" {
			content += publicURLLine + "\n"
		}
	case !md.IsDefined("remote", "mode"):
		content = insertAfterSection(content, "remote", remoteModeLine)
		if publicURL != "" {
			content = insertAfterSection(content, "remote", publicURLLine)
		}
	default:
		if mode == "relay" && publicURL != "" {
			content = setKeyInSection(content, md, "remote", "public_url", publicURLLine)
		}
	}

	relayURLLine := "url = " + strconv.Quote(relayURL)
	tokenLine := "token = " + strconv.Quote(token)
	if !md.IsDefined("relay") {
		if content != "" && !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		content += "\n[relay]\n" + relayURLLine + "\n" + tokenLine + "\n"
	} else {
		content = setKeyInSection(content, md, "relay", "token", tokenLine)
		content = setKeyInSection(content, md, "relay", "url", relayURLLine)
	}

	return os.WriteFile(cfgPath, []byte(content), 0o644)
}

func validateRelaySetupInputs(relayURL, token, publicURL string) error {
	if relayURL == "" {
		return fmt.Errorf("relay url is required")
	}
	u, err := url.Parse(relayURL)
	if err != nil {
		return fmt.Errorf("relay url %q is invalid: %w", relayURL, err)
	}
	if u.Scheme != "ws" && u.Scheme != "wss" {
		return fmt.Errorf("relay url %q must use ws or wss scheme", relayURL)
	}
	if u.Host == "" {
		return fmt.Errorf("relay url %q has no host", relayURL)
	}
	if token == "" {
		return fmt.Errorf("relay token is required")
	}
	if publicURL == "" {
		return nil
	}
	public, err := url.Parse(publicURL)
	if err != nil {
		return fmt.Errorf("public url %q is invalid: %w", publicURL, err)
	}
	if public.Scheme != "http" && public.Scheme != "https" {
		return fmt.Errorf("public url %q must use http or https scheme", publicURL)
	}
	if public.Host == "" {
		return fmt.Errorf("public url %q has no host", publicURL)
	}
	return nil
}

func setKeyInSection(content string, md toml.MetaData, section, key, line string) string {
	if md.IsDefined(section, key) {
		return replaceKeyInSection(content, section, key, line)
	}
	return insertAfterSection(content, section, line)
}

// insertAfterSection inserts a line immediately after the [section] header.
func insertAfterSection(content, section, line string) string {
	re := regexp.MustCompile(`(?m)^\[` + regexp.QuoteMeta(section) + `\][ \t]*\r?\n`)
	loc := re.FindStringIndex(content)
	if loc == nil {
		return content // shouldn't happen, caller checked IsDefined
	}
	return content[:loc[1]] + line + "\n" + content[loc[1]:]
}

// replaceKeyInSection replaces a key = value line within a TOML section.
// Matches the first line starting with the key name (ignoring leading
// whitespace) between the section header and the next section header.
func replaceKeyInSection(content, section, key, replacement string) string {
	headerRe := regexp.MustCompile(`(?m)^\[` + regexp.QuoteMeta(section) + `\][ \t]*\r?\n`)
	headerLoc := headerRe.FindStringIndex(content)
	if headerLoc == nil {
		return content
	}

	// Search for the key line between the header and the next section.
	rest := content[headerLoc[1]:]
	keyRe := regexp.MustCompile(`(?m)^[ \t]*` + regexp.QuoteMeta(key) + `[ \t]*=.*$`)
	nextSection := regexp.MustCompile(`(?m)^\[`)

	// Limit search to before the next section header.
	searchEnd := len(rest)
	if loc := nextSection.FindStringIndex(rest); loc != nil {
		searchEnd = loc[0]
	}

	keyLoc := keyRe.FindStringIndex(rest[:searchEnd])
	if keyLoc == nil {
		return content
	}

	// Replace the matched line with the new value.
	absStart := headerLoc[1] + keyLoc[0]
	absEnd := headerLoc[1] + keyLoc[1]
	return content[:absStart] + replacement + content[absEnd:]
}

// remoteStatus checks on a running daemon with tailscale enabled.
// Polls until tailscale reaches a known state, then displays the result.
func remoteStatus(stdout, stderr io.Writer) int {
	sock := paths.SocketPath()
	if !unixipc.Healthy(sock) {
		fmt.Fprintln(stdout, "Remote access is enabled but the daemon is not running.")
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "Start it with:")
		fmt.Fprintln(stdout, "  jumpd start")
		return 0
	}

	return remotePoll(stdout, stderr)
}

// tailscaleHealth is the subset of the health response we care about.
type tailscaleHealth struct {
	Listen string
	TS     *tsHealth
}

type tsHealth struct {
	FQDN      string `json:"fqdn"`
	MagicDNS  bool   `json:"magic_dns"`
	HTTPS     bool   `json:"https"`
	AuthURL   string `json:"auth_url"`
	Connected bool   `json:"connected"`
}

// fetchTailscaleHealth fetches the tailscale status from the daemon.
func fetchTailscaleHealth(client *http.Client) (*tailscaleHealth, error) {
	resp, err := client.Get("http://localhost/v1/health")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var health struct {
		OK   bool `json:"ok"`
		Data struct {
			Listen    string    `json:"listen"`
			Tailscale *tsHealth `json:"tailscale"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return nil, fmt.Errorf("unexpected response")
	}
	if !health.OK {
		return nil, fmt.Errorf("unhealthy response")
	}
	return &tailscaleHealth{
		Listen: health.Data.Listen,
		TS:     health.Data.Tailscale,
	}, nil
}

// remotePoll polls the daemon's health endpoint until tailscale reaches
// a known state (connected, needs login, or timeout). Then displays
// the appropriate information.
func remotePoll(stdout, stderr io.Writer) int {
	sock := paths.SocketPath()
	client := unixipc.Client(sock)

	fmt.Fprintf(stdout, "Connecting to Tailscale... ")

	// Poll until tailscale reaches a definitive state.
	// The daemon needs time to start tsnet, contact the control server,
	// and either get an auth URL or establish the connection.
	var result *tailscaleHealth
	deadline := time.Now().Add(30 * time.Second)
	tick := time.NewTicker(1 * time.Second)
	defer tick.Stop()

	for time.Now().Before(deadline) {
		h, err := fetchTailscaleHealth(client)
		if err != nil {
			// Daemon might have just restarted; keep trying.
			<-tick.C
			continue
		}
		if h.TS == nil {
			// Tailscale object not yet present in response.
			<-tick.C
			continue
		}
		if h.TS.Connected || h.TS.AuthURL != "" {
			result = h
			break
		}
		// Still connecting, keep polling.
		<-tick.C
	}

	if result == nil {
		// Last-ditch fetch for whatever state we have.
		if h, err := fetchTailscaleHealth(client); err == nil {
			result = h
		}
	}

	fmt.Fprintln(stdout) // end the "Connecting..." line

	if result == nil || result.TS == nil {
		fmt.Fprintln(stdout)
		fmt.Fprintln(stderr, "Could not reach the daemon. Check that it's running:")
		fmt.Fprintln(stderr, "  jumpd start")
		return 1
	}

	return displayStatus(result, stdout)
}

// displayStatus renders the tailscale connection status.
func displayStatus(h *tailscaleHealth, stdout io.Writer) int {
	ts := h.TS

	// Needs login: show the auth URL and nothing else. The user must
	// complete login before we can know about HTTPS/MagicDNS.
	if ts.AuthURL != "" {
		fmt.Fprintln(stdout, "To complete setup, log in to Tailscale:")
		fmt.Fprintf(stdout, "  %s\n", ts.AuthURL)
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "After logging in, run `jumpd tsnet` again to check the connection.")
		fmt.Fprintln(stdout)
		fmt.Fprintf(stdout, "Docs: %s\n", remoteDocsURL)
		return 0
	}

	// Connected and fully operational.
	if ts.Connected {
		fmt.Fprintf(stdout, "  local:  http://%s\n", h.Listen)
		if ts.FQDN != "" {
			fmt.Fprintf(stdout, "  remote: https://%s\n", ts.FQDN)
		}
		fmt.Fprintln(stdout)

		problems := 0
		if !ts.HTTPS {
			fmt.Fprintln(stdout, "  ✗ HTTPS is not enabled in your tailnet")
			problems++
		}
		if !ts.MagicDNS {
			fmt.Fprintln(stdout, "  ✗ MagicDNS is not enabled in your tailnet")
			problems++
		}

		if problems > 0 {
			fmt.Fprintln(stdout)
			fmt.Fprintln(stdout, "Enable these in your Tailscale admin console:")
			fmt.Fprintln(stdout, "  https://login.tailscale.com/admin/dns")
			fmt.Fprintln(stdout)
			fmt.Fprintf(stdout, "Docs: %s\n", remoteDocsURL)
			return 1
		}

		fmt.Fprintln(stdout, "Remote access is active.")
		fmt.Fprintln(stdout)
		fmt.Fprintf(stdout, "Docs: %s\n", remoteDocsURL)
		return 0
	}

	// Not connected and no auth URL. Tailscale is in some intermediate state.
	fmt.Fprintln(stdout, "Tailscale is still connecting. This can take a minute on first setup.")
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "Try again shortly:")
	fmt.Fprintln(stdout, "  jumpd tsnet")
	fmt.Fprintln(stdout)
	fmt.Fprintf(stdout, "Docs: %s\n", remoteDocsURL)
	return 0
}
