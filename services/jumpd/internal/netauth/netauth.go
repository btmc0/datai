// Package netauth provides HTTP middleware and a login endpoint for the
// network listener. It authenticates requests via bearer token (header)
// or session cookie, and accepts token-bearing login links for browser-based access.
//
// The login flow:
//  1. Browser opens any page without a valid cookie.
//  2. Middleware redirects to /auth/login.
//  3. A token-bearing link (/auth/login?token=...) validates the token, sets
//     an HttpOnly cookie, and redirects to /.
//  4. All subsequent requests carry the cookie.
//
// Programmatic clients use the Authorization: Bearer <token> header instead.
package netauth

import (
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/sting8k/jump/services/jumpd/internal/authtoken"
)

const (
	cookieName = "jump-token"
	// cookieMaxAge is 90 days. The token itself doesn't expire, so the cookie
	// just needs to be long-lived enough that users don't have to re-enter it
	// constantly, but short enough that a stolen cookie eventually stops working
	// if the token is rotated.
	cookieMaxAge = 90 * 24 * 60 * 60

	authLimitWindow      = 5 * time.Minute
	authLimitMaxFailures = 8
)

// Middleware returns an http.Handler that wraps next with token authentication.
// Requests with a valid bearer token or cookie are passed through.
// API/WebSocket requests without valid auth get 401.
// Browser requests without valid auth are redirected to the login page.
func Middleware(token string, next http.Handler) http.Handler {
	loginLimiter := newAuthRateLimiter(authLimitWindow, authLimitMaxFailures)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The login page and its POST handler must be accessible without auth.
		if r.URL.Path == "/auth/login" {
			handleLogin(token, loginLimiter, w, r)
			return
		}

		// The web app manifest must be publicly accessible. Browsers fetch
		// it without cookies, so auth-gating it returns the login HTML
		// page which Chrome then fails to parse as JSON.
		if r.URL.Path == "/manifest.json" {
			next.ServeHTTP(w, r)
			return
		}

		// Shutdown is a local-only operation (available via Unix socket).
		// Block it entirely on the TCP listener regardless of auth.
		if r.URL.Path == "/v1/shutdown" {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		if isAuthorized(r, token) {
			next.ServeHTTP(w, r)
			return
		}

		// Distinguish API requests from browser navigation.
		if isAPIRequest(r) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"ok":false,"error":{"code":"unauthorized","message":"valid bearer token or session cookie required"}}`))
			return
		}

		// Browser: redirect to login page.
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
	})
}

func isAuthorized(r *http.Request, token string) bool {
	// Check Authorization header.
	if h := r.Header.Get("Authorization"); h != "" {
		val := strings.TrimPrefix(h, "Bearer ")
		if val != h && authtoken.Equal(val, token) {
			return true
		}
	}

	// Check cookie.
	if c, err := r.Cookie(cookieName); err == nil && authtoken.Equal(c.Value, token) {
		return true
	}

	return false
}

func isAPIRequest(r *http.Request) bool {
	// WebSocket upgrades, API paths, and SSE requests are programmatic.
	if strings.HasPrefix(r.URL.Path, "/v1/") || strings.HasPrefix(r.URL.Path, "/ws/") {
		return true
	}
	if strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		return true
	}
	accept := r.Header.Get("Accept")
	if strings.Contains(accept, "application/json") || strings.Contains(accept, "text/event-stream") {
		return true
	}
	return false
}

type authRateLimiter struct {
	mu          sync.Mutex
	window      time.Duration
	maxFailures int
	failures    map[string]authFailureWindow
}

type authFailureWindow struct {
	count int
	start time.Time
}

func newAuthRateLimiter(window time.Duration, maxFailures int) *authRateLimiter {
	return &authRateLimiter{
		window:      window,
		maxFailures: maxFailures,
		failures:    map[string]authFailureWindow{},
	}
}

// recordFailure records one failed login attempt. It returns true when this
// request should be rate-limited.
func (l *authRateLimiter) recordFailure(key string) bool {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()

	state := l.failures[key]
	if state.start.IsZero() || now.Sub(state.start) > l.window {
		state = authFailureWindow{start: now}
	}
	state.count++
	l.failures[key] = state

	return state.count > l.maxFailures
}

func (l *authRateLimiter) reset(key string) {
	l.mu.Lock()
	delete(l.failures, key)
	l.mu.Unlock()
}

func clientKey(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}
	if r.RemoteAddr != "" {
		return r.RemoteAddr
	}
	return "unknown"
}

func handleLogin(token string, limiter *authRateLimiter, w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Check if already authenticated; redirect to home.
		if isAuthorized(r, token) {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		// If the token is in the query string (QR code flow), validate
		// and set the cookie immediately. This avoids displaying the
		// login page when scanning a QR code.
		if qToken := strings.TrimSpace(r.URL.Query().Get("token")); qToken != "" {
			if authtoken.Equal(qToken, token) {
				limiter.reset(clientKey(r))
				setAuthCookie(w, token)
				log.Printf("netauth: successful login via URL token from %s", r.RemoteAddr)
				http.Redirect(w, r, "/", http.StatusSeeOther)
				return
			}
			// Invalid token in URL: rate-limit failed attempts and show an error.
			if limiter.recordFailure(clientKey(r)) {
				serveRateLimit(w)
				return
			}
			serveLoginPage(w, "Invalid token in URL.", false)
			return
		}

		serveLoginPage(w, "", r.URL.Query().Get("form") == "1")

	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			serveLoginPage(w, "Invalid request.", true)
			return
		}

		submitted := strings.TrimSpace(r.FormValue("token"))
		if !authtoken.Equal(submitted, token) {
			log.Printf("netauth: login attempt with invalid token from %s", r.RemoteAddr)
			if limiter.recordFailure(clientKey(r)) {
				serveRateLimit(w)
				return
			}
			serveLoginPage(w, "Invalid token. Check the value and try again.", true)
			return
		}

		limiter.reset(clientKey(r))
		setAuthCookie(w, token)
		log.Printf("netauth: successful login from %s", r.RemoteAddr)
		http.Redirect(w, r, "/", http.StatusSeeOther)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func setAuthCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   cookieMaxAge,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
}

func serveRateLimit(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Retry-After", "300")
	w.WriteHeader(http.StatusTooManyRequests)
	_, _ = w.Write([]byte(`<!DOCTYPE html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<title>jump - Too Many Attempts</title></head>
<body style="font-family:system-ui,-apple-system,sans-serif;background:#0a0a0a;color:#e0e0e0;display:flex;align-items:center;justify-content:center;min-height:100vh;margin:0;padding:1em">
<div style="background:#1a1a1a;border:1px solid #333;border-radius:8px;padding:2em;max-width:420px;width:100%">
<h1 style="font-size:1.2em;margin:0 0 .5em;color:#fff">Too many login attempts</h1>
<p style="font-size:.9em;line-height:1.5;margin:0;color:#999">Wait a few minutes, then open a fresh jump auth link again.</p>
</div></body></html>`))
}

func serveLoginPage(w http.ResponseWriter, errMsg string, showForm bool) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")

	errorHTML := ""
	if errMsg != "" {
		errorHTML = `<p style="color:#e74c3c;margin-bottom:1em">` + errMsg + `</p>`
	}

	introHTML := `<p>This jump instance requires authentication. Open a token-bearing jump auth link to continue.</p>`
	formHTML := ``
	if showForm {
		introHTML = `<p>This jump instance requires authentication. Enter the access token to continue.</p>`
		formHTML = `<form method="POST" action="/auth/login" autocomplete="off">
    <label for="token">Access Token</label>
    <input type="password" id="token" name="token" required autofocus
           placeholder="Paste token here" autocomplete="off">
    <button type="submit">Sign In</button>
  </form>`
	}

	// Minimal inline page. No external dependencies, no JavaScript required.
	_, _ = w.Write([]byte(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>jump - Authentication Required</title>
<style>
  *, *::before, *::after { box-sizing: border-box; }
  body {
    font-family: system-ui, -apple-system, sans-serif;
    background: #0a0a0a; color: #e0e0e0;
    display: flex; align-items: center; justify-content: center;
    min-height: 100vh; margin: 0; padding: 1em;
  }
  .card {
    background: #1a1a1a; border: 1px solid #333; border-radius: 8px;
    padding: 2em; max-width: 420px; width: 100%;
  }
  h1 { font-size: 1.2em; margin: 0 0 0.5em; color: #fff; }
  p { font-size: 0.9em; line-height: 1.5; margin: 0 0 1.5em; color: #999; }
  label { display: block; font-size: 0.85em; margin-bottom: 0.4em; color: #ccc; }
  input[type="password"] {
    width: 100%; padding: 0.6em 0.8em; font-size: 0.95em;
    font-family: monospace; background: #111; color: #fff;
    border: 1px solid #444; border-radius: 4px; outline: none;
  }
  input[type="password"]:focus { border-color: #666; }
  button {
    width: 100%; padding: 0.7em; margin-top: 1em; font-size: 0.95em;
    background: #fff; color: #000; border: none; border-radius: 4px;
    cursor: pointer; font-weight: 500;
  }
  button:hover { background: #ddd; }
  .hint { font-size: 0.8em; color: #666; margin-top: 1em; }
  code { background: #222; padding: 0.15em 0.4em; border-radius: 3px; font-size: 0.9em; }
</style>
</head>
<body>
<div class="card">
  <h1>jump</h1>
  ` + introHTML + `
  ` + errorHTML + `
  ` + formHTML + `
  <p class="hint">Find your login link by running <code>jumpd auth</code> on the host machine.</p>
</div>
</body>
</html>`))
}
