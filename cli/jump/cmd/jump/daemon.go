package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/sting8k/jump/packages/paths"
)

// ensureJumpd checks if jumpd is reachable and starts it if not.
// If a daemon is running but reports a different version, it is replaced
// so the child process always talks to a compatible daemon.
// Called once at startup — if jumpd dies later, we don't restart it.
// Returns true if jumpd was started (or replaced) by this call.
func ensureJumpd() bool {
	if !jumpdNeedsStart() {
		return false
	}

	jumpdBin := findJumpdBin()
	if jumpdBin == "" {
		log.Printf("warning: jumpd not found (install it alongside jump or add it to PATH)")
		return false
	}

	// jumpd run starts in the foreground; we background it ourselves.
	return startJumpd(jumpdBin, []string{"run"})
}

// jumpdNeedsStart checks the running daemon.
func jumpdNeedsStart() bool {
	// "dev" builds never replace — avoids churn during development.
	if version == "dev" {
		return !jumpdHealthy(500 * time.Millisecond)
	}

	client := jumpdClient()
	client.Timeout = 500 * time.Millisecond
	resp, err := client.Get(jumpdBaseURL() + "/v1/health")
	if err != nil {
		return true // not running
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return true // not healthy
	}

	var health struct {
		Data struct {
			Version string `json:"version"`
		} `json:"data"`
	}
	body, _ := io.ReadAll(resp.Body)
	if json.Unmarshal(body, &health) != nil {
		return false // can't parse, leave it alone
	}

	// Same version: no action needed. Different version: replace.
	return health.Data.Version != version
}

// findJumpdBin locates the jumpd binary: sibling first, then PATH.
func findJumpdBin() string {
	if self, err := os.Executable(); err == nil {
		sibling := filepath.Join(filepath.Dir(self), "jumpd")
		if _, err := os.Stat(sibling); err == nil {
			return sibling
		}
	}
	if p, err := exec.LookPath("jumpd"); err == nil {
		return p
	}
	return ""
}

// startJumpd launches jumpd in the background with the given args.
func startJumpd(jumpdBin string, args []string) bool {
	// Log jumpd output to a file so users can diagnose startup failures.
	logPath := filepath.Join(os.TempDir(), "jumpd.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		logFile = nil
	}

	cmd := exec.Command(jumpdBin, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	cmd.Stdout = nil
	cmd.Stderr = logFile
	if err := cmd.Start(); err != nil {
		log.Printf("warning: could not start jumpd: %v", err)
		if logFile != nil {
			logFile.Close()
		}
		return false
	}
	go func() {
		cmd.Wait()
		if logFile != nil {
			logFile.Close()
		}
	}()

	return true
}

// jumpdClient returns an HTTP client connected to jumpd via Unix socket.
func jumpdClient() *http.Client {
	sockPath := paths.SocketPath()
	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return net.DialTimeout("unix", sockPath, 2*time.Second)
			},
		},
		Timeout: 5 * time.Second,
	}
}

// jumpdBaseURL returns the base URL for jumpd HTTP requests.
// The host is ignored by the Unix socket transport.
func jumpdBaseURL() string {
	return "http://localhost"
}

func jumpdHealthy(timeout time.Duration) bool {
	client := jumpdClient()
	client.Timeout = timeout
	resp, err := client.Get(jumpdBaseURL() + "/v1/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// registerWithJumpd posts the session's registration to jumpd and
// returns true once jumpd accepts it. Retries a handful of times to
// cover the case where jumpd is still starting up. Returns false if
// every attempt fails: callers that care about registration outcome
// (e.g. the --no-attach handshake) treat false as "give up".
func registerWithJumpd(sessionID, socketPath string) bool {
	baseURL := jumpdBaseURL()

	payload, _ := json.Marshal(map[string]string{
		"session_id":  sessionID,
		"socket_path": socketPath,
	})

	// Retry a few times — jump may start before the HTTP server is ready
	for i := 0; i < 5; i++ {
		if i > 0 {
			time.Sleep(500 * time.Millisecond)
		}
		client := jumpdClient()
		resp, err := client.Post(baseURL+"/v1/register", "application/json", bytes.NewReader(payload))
		if err != nil {
			continue
		}
		resp.Body.Close()
		if resp.StatusCode == 200 {
			return true
		}
	}
	return false
}

func deregisterFromJumpd(sessionID string) {
	baseURL := jumpdBaseURL()

	payload, _ := json.Marshal(map[string]string{"session_id": sessionID})
	client := jumpdClient()
	resp, err := client.Post(baseURL+"/v1/deregister", "application/json", bytes.NewReader(payload))
	if err != nil {
		return
	}
	resp.Body.Close()
}

// parseHealthField extracts a string field from the data object
// of a /v1/health JSON response.
func parseHealthField(body []byte, field string) string {
	var resp struct {
		Data map[string]json.RawMessage `json:"data"`
	}
	if json.Unmarshal(body, &resp) != nil {
		return ""
	}
	raw, ok := resp.Data[field]
	if !ok {
		return ""
	}
	var val string
	if json.Unmarshal(raw, &val) != nil {
		return ""
	}
	return val
}

// parseTailscaleURL extracts the tailscale_url from a /v1/health JSON response.
func parseTailscaleURL(body []byte) string {
	var resp struct {
		Data struct {
			TailscaleURL string `json:"tailscale_url"`
		} `json:"data"`
	}
	if json.Unmarshal(body, &resp) == nil {
		return resp.Data.TailscaleURL
	}
	return ""
}

// parseUpdateAvailable extracts update_available from a /v1/health JSON response.
func parseUpdateAvailable(body []byte) string {
	var resp struct {
		Data struct {
			UpdateAvailable string `json:"update_available"`
		} `json:"data"`
	}
	if json.Unmarshal(body, &resp) == nil {
		return resp.Data.UpdateAvailable
	}
	return ""
}
