package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/gmuxapp/gmux/packages/paths"
	"github.com/gmuxapp/gmux/services/gmuxd/internal/config"
	"github.com/gmuxapp/gmux/services/gmuxd/internal/unixipc"
)

type daemonHealthData struct {
	Version      string    `json:"version"`
	Status       string    `json:"status"`
	Listen       string    `json:"listen"`
	TailscaleURL string    `json:"tailscale_url,omitempty"`
	Tailscale    *tsHealth `json:"tailscale,omitempty"`
}

type daemonHealthResponse struct {
	OK   bool             `json:"ok"`
	Data daemonHealthData `json:"data"`
}

func runDoctor(stdout, stderr io.Writer) int {
	failures := 0

	_, _ = fmt.Fprintln(stdout, "gmuxd doctor")

	cfg, cfgErr := config.Load()
	if cfgErr != nil {
		failures++
		_, _ = fmt.Fprintf(stdout, "✗ config invalid: %v\n", cfgErr)
		_, _ = fmt.Fprintf(stdout, "  Fix: edit %s and rerun `gmuxd doctor`.\n", config.Path())
	} else {
		_, _ = fmt.Fprintf(stdout, "✓ config valid: %s\n", config.Path())
		_, _ = fmt.Fprintf(stdout, "✓ remote mode: %s\n", doctorRemoteMode(cfg))
	}

	sock := paths.SocketPath()
	health, err := fetchDaemonHealth(sock)
	if err != nil {
		failures++
		_, _ = fmt.Fprintf(stdout, "✗ daemon not reachable: %v\n", err)
		_, _ = fmt.Fprintln(stdout, "  Fix: run `gmuxd start`.")
		return doctorExitCode(failures)
	}

	_, _ = fmt.Fprintf(stdout, "✓ daemon running: gmuxd %s (%s)\n", health.Version, health.Status)
	_, _ = fmt.Fprintf(stdout, "✓ socket reachable: %s\n", sock)
	if health.Listen != "" {
		_, _ = fmt.Fprintf(stdout, "✓ local UI reachable through daemon: http://%s\n", health.Listen)
	} else {
		failures++
		_, _ = fmt.Fprintln(stdout, "✗ daemon health did not report a local listen address")
	}

	if cfgErr == nil {
		failures += doctorRemoteChecks(stdout, cfg, health)
	}

	return doctorExitCode(failures)
}

func fetchDaemonHealth(sock string) (daemonHealthData, error) {
	client := unixipc.Client(sock)
	resp, err := client.Get("http://localhost/v1/health")
	if err != nil {
		return daemonHealthData{}, fmt.Errorf("socket %s: %w", sock, err)
	}
	defer resp.Body.Close()

	var health daemonHealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return daemonHealthData{}, fmt.Errorf("unexpected health response: %w", err)
	}
	if !health.OK {
		return daemonHealthData{}, fmt.Errorf("daemon returned unhealthy response")
	}
	return health.Data, nil
}

func doctorRemoteMode(cfg config.Config) string {
	switch {
	case cfg.Remote.Mode != "":
		return cfg.Remote.Mode
	case cfg.Tailscale.Enabled:
		return "tsnet (legacy tailscale.enabled)"
	case cfg.Relay.Enabled:
		return "relay (legacy relay.enabled)"
	default:
		return "local-only"
	}
}

func doctorRemoteChecks(stdout io.Writer, cfg config.Config, health daemonHealthData) int {
	switch {
	case cfg.Tailscale.Enabled:
		return doctorTsnet(stdout, health)
	case cfg.Relay.Enabled:
		return doctorRelay(stdout, cfg)
	default:
		_, _ = fmt.Fprintln(stdout, "✓ remote access disabled; local-only baseline")
		return 0
	}
}

func doctorTsnet(stdout io.Writer, health daemonHealthData) int {
	if health.Tailscale == nil {
		_, _ = fmt.Fprintln(stdout, "✗ tsnet enabled but daemon health did not report Tailscale status")
		_, _ = fmt.Fprintln(stdout, "  Fix: restart the daemon with `gmuxd restart`.")
		return 1
	}

	ts := health.Tailscale
	if ts.AuthURL != "" {
		_, _ = fmt.Fprintln(stdout, "✗ tsnet needs Tailscale login")
		_, _ = fmt.Fprintf(stdout, "  Fix: open %s, then rerun `gmuxd doctor`.\n", ts.AuthURL)
		return 1
	}
	if !ts.Connected {
		_, _ = fmt.Fprintln(stdout, "✗ tsnet is still connecting")
		_, _ = fmt.Fprintln(stdout, "  Fix: wait a moment or check daemon logs with `gmuxd log-path`.")
		return 1
	}

	remoteURL := health.TailscaleURL
	if remoteURL == "" && ts.FQDN != "" {
		remoteURL = "https://" + ts.FQDN
	}
	if remoteURL != "" {
		_, _ = fmt.Fprintf(stdout, "✓ tsnet remote URL: %s\n", remoteURL)
	} else {
		_, _ = fmt.Fprintln(stdout, "✓ tsnet connected")
	}

	failures := 0
	if ts.MagicDNS {
		_, _ = fmt.Fprintln(stdout, "✓ Tailscale MagicDNS enabled")
	} else {
		failures++
		_, _ = fmt.Fprintln(stdout, "✗ Tailscale MagicDNS is disabled")
		_, _ = fmt.Fprintln(stdout, "  Fix: enable MagicDNS in the Tailscale admin console.")
	}
	if ts.HTTPS {
		_, _ = fmt.Fprintln(stdout, "✓ Tailscale HTTPS enabled")
	} else {
		failures++
		_, _ = fmt.Fprintln(stdout, "✗ Tailscale HTTPS is disabled")
		_, _ = fmt.Fprintln(stdout, "  Fix: enable HTTPS certificates in the Tailscale admin console.")
	}
	return failures
}

func doctorRelay(stdout io.Writer, cfg config.Config) int {
	_, _ = fmt.Fprintf(stdout, "✓ relay config: %s\n", cfg.Relay.URL)
	if cfg.Remote.PublicURL != "" {
		_, _ = fmt.Fprintf(stdout, "✓ relay public URL: %s\n", cfg.Remote.PublicURL)
	}

	healthURL, err := relayHealthURL(cfg.Relay.URL)
	if err != nil {
		_, _ = fmt.Fprintf(stdout, "✗ relay health URL: %v\n", err)
		return 1
	}

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(healthURL)
	if err != nil {
		_, _ = fmt.Fprintf(stdout, "✗ relay server not reachable: %v\n", err)
		_, _ = fmt.Fprintln(stdout, "  Fix: check relay.url, networking, and that gmux-relayd is running.")
		return 1
	}
	defer resp.Body.Close()

	var health struct {
		OK             bool `json:"ok"`
		AgentConnected bool `json:"agent_connected"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil || !health.OK {
		_, _ = fmt.Fprintln(stdout, "✗ relay server returned an unexpected health response")
		_, _ = fmt.Fprintln(stdout, "  Fix: confirm relay.url points at gmux-relayd.")
		return 1
	}
	if !health.AgentConnected {
		_, _ = fmt.Fprintln(stdout, "✗ relay server reachable but gmuxd agent is not connected")
		_, _ = fmt.Fprintln(stdout, "  Fix: check relay token and daemon logs.")
		return 1
	}

	_, _ = fmt.Fprintln(stdout, "✓ relay server reachable and agent connected")
	return 0
}

func relayHealthURL(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	switch u.Scheme {
	case "ws":
		u.Scheme = "http"
	case "wss":
		u.Scheme = "https"
	default:
		return "", fmt.Errorf("relay URL must use ws or wss scheme")
	}
	u.Path = "/_gmux/health"
	u.RawQuery = ""
	u.Fragment = ""
	return u.String(), nil
}

func doctorExitCode(failures int) int {
	if failures > 0 {
		return 1
	}
	return 0
}
