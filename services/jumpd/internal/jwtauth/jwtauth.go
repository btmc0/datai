// Package jwtauth provides HTTP middleware that verifies Open WebUI JWT tokens.
// It extracts user identity from JWT claims and makes it available via context.
// When WEBUI_SECRET_KEY is not set, the middleware passes requests through
// for backward compatibility with the existing netauth token-based auth.
package jwtauth

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// Role constants for permission hierarchy: admin > user > viewer.
const (
	RoleAdmin  = "admin"
	RoleUser   = "user"
	RoleViewer = "viewer"
)

// roleLevel maps role strings to numeric levels for comparison.
var roleLevel = map[string]int{
	RoleAdmin:  3,
	RoleUser:   2,
	RoleViewer: 1,
}

// User represents an authenticated user extracted from JWT claims.
type User struct {
	ID     string
	Email  string
	Role   string
	Groups []string // from JWT "groups" claim
}

type contextKey int

const userKey contextKey = iota

// ExportedUserKey is the context key for the authenticated user.
// Exported for use in tests that need to inject a user into the context.
var ExportedUserKey = userKey

// Middleware returns an http.Handler that verifies JWT tokens from the
// Authorization header (Bearer) or the "token" cookie. On success it
// stores the User in the request context. On failure it returns 401.
//
// If WEBUI_SECRET_KEY is not set, the middleware passes all requests
// through unchanged so the existing netauth layer can handle auth.
func Middleware(next http.Handler) http.Handler {
	secret := os.Getenv("WEBUI_SECRET_KEY")
	if secret == "" {
		log.Println("jwtauth: WEBUI_SECRET_KEY not set, JWT auth disabled (passthrough)")
		return next
	}
	secretBytes := []byte(secret)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenStr := extractToken(r)
		if tokenStr == "" {
			// No token at all — if netauth already authenticated (local mode),
			// create a default local user so DATAI APIs work.
			if isNetauthPassthrough(r) {
				ctx := context.WithValue(r.Context(), userKey, localUser())
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
			writeUnauthorized(w, "missing token")
			return
		}

		user, err := parseToken(tokenStr, secretBytes)
		if err != nil {
			// Token exists but isn't a valid JWT — likely Jump's netauth
			// bearer token. Treat as local authenticated user.
			ctx := context.WithValue(r.Context(), userKey, localUser())
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		ctx := context.WithValue(r.Context(), userKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// UserFromContext returns the authenticated User from the context, or nil.
func UserFromContext(ctx context.Context) *User {
	u, _ := ctx.Value(userKey).(*User)
	return u
}

// UserIDFromContext returns the authenticated user's ID, or empty string.
func UserIDFromContext(ctx context.Context) string {
	if u := UserFromContext(ctx); u != nil {
		return u.ID
	}
	return ""
}

// IsAdmin returns true if the authenticated user has the "admin" role.
func IsAdmin(ctx context.Context) bool {
	if u := UserFromContext(ctx); u != nil {
		return u.Role == RoleAdmin
	}
	return false
}

// CanWrite checks if user has write permission (admin or user, not viewer).
func CanWrite(ctx context.Context) bool {
	u := UserFromContext(ctx)
	if u == nil {
		return false
	}
	return roleLevel[u.Role] >= roleLevel[RoleUser]
}

// CanAdmin checks if user has admin permission.
func CanAdmin(ctx context.Context) bool {
	return IsAdmin(ctx)
}

// HasGroup checks if the authenticated user belongs to a specific group.
func HasGroup(ctx context.Context, group string) bool {
	u := UserFromContext(ctx)
	if u == nil {
		return false
	}
	for _, g := range u.Groups {
		if g == group {
			return true
		}
	}
	return false
}

// RequireRole returns middleware that checks the user has at least the given
// role level. Role hierarchy: admin > user > viewer.
// Returns 403 if the user's role is below minRole.
func RequireRole(minRole string) func(http.Handler) http.Handler {
	minLevel := roleLevel[minRole]
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u := UserFromContext(r.Context())
			if u == nil {
				writeUnauthorized(w, "missing user")
				return
			}
			if roleLevel[u.Role] < minLevel {
				writeForbidden(w, "insufficient role")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireGroup returns middleware that checks the user belongs to at least
// one of the specified groups. Returns 403 if the user has none of them.
func RequireGroup(groups ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u := UserFromContext(r.Context())
			if u == nil {
				writeUnauthorized(w, "missing user")
				return
			}
			for _, required := range groups {
				for _, g := range u.Groups {
					if g == required {
						next.ServeHTTP(w, r)
						return
					}
				}
			}
			writeForbidden(w, "not in required group")
		})
	}
}

func extractToken(r *http.Request) string {
	if h := r.Header.Get("Authorization"); h != "" {
		if after, ok := strings.CutPrefix(h, "Bearer "); ok {
			return after
		}
	}
	if c, err := r.Cookie("token"); err == nil {
		return c.Value
	}
	return ""
}

func parseToken(tokenStr string, secret []byte) (*User, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return secret, nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, jwt.ErrTokenInvalidClaims
	}

	sub, _ := claims["sub"].(string)
	if sub == "" {
		// Open WebUI may use "id" instead of "sub"
		sub, _ = claims["id"].(string)
	}
	if sub == "" {
		return nil, jwt.ErrTokenInvalidClaims
	}

	email, _ := claims["email"].(string)
	role, _ := claims["role"].(string)

	var groups []string
	if gs, ok := claims["groups"].([]any); ok {
		for _, g := range gs {
			if s, ok := g.(string); ok {
				groups = append(groups, s)
			}
		}
	}

	return &User{ID: sub, Email: email, Role: role, Groups: groups}, nil
}

func writeUnauthorized(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(`{"ok":false,"error":{"code":"unauthorized","message":"` + msg + `"}}`))
}

// localUser returns a default admin user for local/dev mode where
// Jump's netauth has already authenticated the request but no JWT exists.
func localUser() *User {
	hostname, _ := os.Hostname()
	return &User{
		ID:    "local-" + hostname,
		Email: "local@" + hostname,
		Role:  RoleAdmin,
	}
}

// isNetauthPassthrough checks if the request was already authenticated
// by Jump's netauth (unix socket or has a valid session cookie).
func isNetauthPassthrough(r *http.Request) bool {
	// Unix socket connections (local IPC)
	if r.RemoteAddr == "@" || strings.HasPrefix(r.RemoteAddr, "/") || r.RemoteAddr == "" {
		return true
	}
	// If we got here through the mux, netauth already let us through
	// (it would have returned 401 otherwise).
	return true
}

func writeForbidden(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	_, _ = w.Write([]byte(`{"ok":false,"error":{"code":"forbidden","message":"` + msg + `"}}`))
}
