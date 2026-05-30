package jwtauth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "test-webui-secret-key-for-jwt"

func makeToken(t *testing.T, claims jwt.MapClaims, secret string) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return s
}

func validClaims() jwt.MapClaims {
	return jwt.MapClaims{
		"sub":    "user-123",
		"email":  "test@example.com",
		"role":   "user",
		"groups": []any{"datai-user"},
		"exp":    float64(time.Now().Add(1 * time.Hour).Unix()),
	}
}

func TestParseValidToken(t *testing.T) {
	tokenStr := makeToken(t, validClaims(), testSecret)
	user, err := parseToken(tokenStr, []byte(testSecret))
	if err != nil {
		t.Fatalf("parseToken: %v", err)
	}
	if user.ID != "user-123" {
		t.Errorf("ID = %q, want %q", user.ID, "user-123")
	}
	if user.Email != "test@example.com" {
		t.Errorf("Email = %q, want %q", user.Email, "test@example.com")
	}
	if user.Role != "user" {
		t.Errorf("Role = %q, want %q", user.Role, "user")
	}
	if len(user.Groups) != 1 || user.Groups[0] != "datai-user" {
		t.Errorf("Groups = %v, want [datai-user]", user.Groups)
	}
}

func TestParseTokenWithIDField(t *testing.T) {
	claims := jwt.MapClaims{
		"id":    "user-456",
		"email": "alt@example.com",
		"role":  "admin",
		"exp":   float64(time.Now().Add(1 * time.Hour).Unix()),
	}
	tokenStr := makeToken(t, claims, testSecret)
	user, err := parseToken(tokenStr, []byte(testSecret))
	if err != nil {
		t.Fatalf("parseToken: %v", err)
	}
	if user.ID != "user-456" {
		t.Errorf("ID = %q, want %q", user.ID, "user-456")
	}
}

func TestExpiredTokenRejected(t *testing.T) {
	claims := validClaims()
	claims["exp"] = float64(time.Now().Add(-1 * time.Hour).Unix())
	tokenStr := makeToken(t, claims, testSecret)
	_, err := parseToken(tokenStr, []byte(testSecret))
	if err == nil {
		t.Fatal("expected error for expired token, got nil")
	}
}

func TestInvalidSignatureRejected(t *testing.T) {
	tokenStr := makeToken(t, validClaims(), "wrong-secret")
	_, err := parseToken(tokenStr, []byte(testSecret))
	if err == nil {
		t.Fatal("expected error for bad signature, got nil")
	}
}

func TestMissingSubRejected(t *testing.T) {
	claims := jwt.MapClaims{
		"email": "test@example.com",
		"exp":   float64(time.Now().Add(1 * time.Hour).Unix()),
	}
	tokenStr := makeToken(t, claims, testSecret)
	_, err := parseToken(tokenStr, []byte(testSecret))
	if err == nil {
		t.Fatal("expected error for missing sub, got nil")
	}
}

func TestMiddlewarePassthroughWhenSecretEmpty(t *testing.T) {
	// Middleware() reads WEBUI_SECRET_KEY from env. When empty it returns
	// next directly. We test that by calling the function directly with
	// an empty env var.
	t.Setenv("WEBUI_SECRET_KEY", "")

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := Middleware(inner)

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d (passthrough)", rec.Code, http.StatusOK)
	}
}

func TestMiddlewareBearerToken(t *testing.T) {
	t.Setenv("WEBUI_SECRET_KEY", testSecret)

	var gotUser *User
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUser = UserFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	handler := Middleware(inner)

	tokenStr := makeToken(t, validClaims(), testSecret)
	req := httptest.NewRequest("GET", "/v1/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if gotUser == nil {
		t.Fatal("user not set in context")
	}
	if gotUser.ID != "user-123" {
		t.Errorf("user.ID = %q, want %q", gotUser.ID, "user-123")
	}
}

func TestMiddlewareCookieToken(t *testing.T) {
	t.Setenv("WEBUI_SECRET_KEY", testSecret)

	var gotUser *User
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUser = UserFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	handler := Middleware(inner)

	tokenStr := makeToken(t, validClaims(), testSecret)
	req := httptest.NewRequest("GET", "/v1/test", nil)
	req.AddCookie(&http.Cookie{Name: "token", Value: tokenStr})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if gotUser == nil {
		t.Fatal("user not set in context")
	}
}

func TestMiddlewareMissingToken(t *testing.T) {
	t.Setenv("WEBUI_SECRET_KEY", testSecret)

	// With no token, jwtauth falls back to a local user (netauth passthrough).
	var gotUser *User
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUser = UserFromContext(r.Context())
	})

	handler := Middleware(inner)

	req := httptest.NewRequest("GET", "/v1/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if gotUser == nil {
		t.Fatal("expected local user fallback, got nil")
	}
	if gotUser.Role != RoleAdmin {
		t.Errorf("local user role = %q, want %q", gotUser.Role, RoleAdmin)
	}
}

func TestContextHelpers(t *testing.T) {
	// Empty context
	if id := UserIDFromContext(context.Background()); id != "" {
		t.Errorf("UserIDFromContext(empty) = %q, want empty", id)
	}
	if u := UserFromContext(context.Background()); u != nil {
		t.Errorf("UserFromContext(empty) = %v, want nil", u)
	}
	if IsAdmin(context.Background()) {
		t.Error("IsAdmin(empty) = true, want false")
	}

	// Context with user
	user := &User{ID: "u1", Email: "a@b.c", Role: "admin"}
	ctx := context.WithValue(context.Background(), userKey, user)
	if id := UserIDFromContext(ctx); id != "u1" {
		t.Errorf("UserIDFromContext = %q, want %q", id, "u1")
	}
	if !IsAdmin(ctx) {
		t.Error("IsAdmin = false, want true")
	}

	// Non-admin
	user2 := &User{ID: "u2", Role: "user"}
	ctx2 := context.WithValue(context.Background(), userKey, user2)
	if IsAdmin(ctx2) {
		t.Error("IsAdmin(user) = true, want false")
	}
}

func TestGroupExtraction(t *testing.T) {
	claims := jwt.MapClaims{
		"sub":    "user-789",
		"role":   "user",
		"groups": []any{"datai-admin", "datai-user", "other-group"},
		"exp":    float64(time.Now().Add(1 * time.Hour).Unix()),
	}
	tokenStr := makeToken(t, claims, testSecret)
	user, err := parseToken(tokenStr, []byte(testSecret))
	if err != nil {
		t.Fatalf("parseToken: %v", err)
	}
	if len(user.Groups) != 3 {
		t.Fatalf("Groups len = %d, want 3", len(user.Groups))
	}
	if user.Groups[0] != "datai-admin" || user.Groups[1] != "datai-user" || user.Groups[2] != "other-group" {
		t.Errorf("Groups = %v", user.Groups)
	}
}

func TestGroupExtractionNoGroups(t *testing.T) {
	claims := jwt.MapClaims{
		"sub":  "user-no-groups",
		"role": "viewer",
		"exp":  float64(time.Now().Add(1 * time.Hour).Unix()),
	}
	tokenStr := makeToken(t, claims, testSecret)
	user, err := parseToken(tokenStr, []byte(testSecret))
	if err != nil {
		t.Fatalf("parseToken: %v", err)
	}
	if len(user.Groups) != 0 {
		t.Errorf("Groups = %v, want empty", user.Groups)
	}
}

func TestCanWriteCanAdmin(t *testing.T) {
	tests := []struct {
		role     string
		canWrite bool
		canAdmin bool
	}{
		{RoleAdmin, true, true},
		{RoleUser, true, false},
		{RoleViewer, false, false},
		{"", false, false},
	}
	for _, tt := range tests {
		user := &User{ID: "u", Role: tt.role}
		ctx := context.WithValue(context.Background(), userKey, user)
		if got := CanWrite(ctx); got != tt.canWrite {
			t.Errorf("CanWrite(role=%q) = %v, want %v", tt.role, got, tt.canWrite)
		}
		if got := CanAdmin(ctx); got != tt.canAdmin {
			t.Errorf("CanAdmin(role=%q) = %v, want %v", tt.role, got, tt.canAdmin)
		}
	}
	// nil context
	if CanWrite(context.Background()) {
		t.Error("CanWrite(empty) = true")
	}
	if CanAdmin(context.Background()) {
		t.Error("CanAdmin(empty) = true")
	}
}

func TestHasGroup(t *testing.T) {
	user := &User{ID: "u", Groups: []string{"datai-user", "team-alpha"}}
	ctx := context.WithValue(context.Background(), userKey, user)
	if !HasGroup(ctx, "datai-user") {
		t.Error("HasGroup(datai-user) = false")
	}
	if !HasGroup(ctx, "team-alpha") {
		t.Error("HasGroup(team-alpha) = false")
	}
	if HasGroup(ctx, "datai-admin") {
		t.Error("HasGroup(datai-admin) = true, want false")
	}
	if HasGroup(context.Background(), "datai-user") {
		t.Error("HasGroup(empty ctx) = true")
	}
}

func userCtx(role string, groups ...string) context.Context {
	user := &User{ID: "u", Role: role, Groups: groups}
	return context.WithValue(context.Background(), userKey, user)
}

func TestRequireRoleMiddleware(t *testing.T) {
	okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	tests := []struct {
		name     string
		minRole  string
		userRole string
		want     int
	}{
		{"admin accessing admin route", RoleAdmin, RoleAdmin, 200},
		{"user accessing admin route", RoleAdmin, RoleUser, 403},
		{"viewer accessing admin route", RoleAdmin, RoleViewer, 403},
		{"admin accessing user route", RoleUser, RoleAdmin, 200},
		{"user accessing user route", RoleUser, RoleUser, 200},
		{"viewer accessing user route", RoleUser, RoleViewer, 403},
		{"viewer accessing viewer route", RoleViewer, RoleViewer, 200},
		{"user accessing viewer route", RoleViewer, RoleUser, 200},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := RequireRole(tt.minRole)(okHandler)
			req := httptest.NewRequest("GET", "/", nil)
			req = req.WithContext(userCtx(tt.userRole))
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != tt.want {
				t.Errorf("status = %d, want %d", rec.Code, tt.want)
			}
		})
	}

	// No user in context → 401
	t.Run("no user", func(t *testing.T) {
		handler := RequireRole(RoleViewer)(okHandler)
		req := httptest.NewRequest("GET", "/", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})
}

func TestRequireGroupMiddleware(t *testing.T) {
	okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	t.Run("user in required group", func(t *testing.T) {
		handler := RequireGroup("datai-admin", "datai-user")(okHandler)
		req := httptest.NewRequest("GET", "/", nil)
		req = req.WithContext(userCtx("user", "datai-user"))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != 200 {
			t.Errorf("status = %d, want 200", rec.Code)
		}
	})

	t.Run("user not in any required group", func(t *testing.T) {
		handler := RequireGroup("datai-admin")(okHandler)
		req := httptest.NewRequest("GET", "/", nil)
		req = req.WithContext(userCtx("user", "datai-user"))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != 403 {
			t.Errorf("status = %d, want 403", rec.Code)
		}
	})

	t.Run("no user in context", func(t *testing.T) {
		handler := RequireGroup("datai-user")(okHandler)
		req := httptest.NewRequest("GET", "/", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})
}

func TestViewerCannotWrite(t *testing.T) {
	// Simulates a viewer trying to POST to a write-protected endpoint
	writeHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !CanWrite(r.Context()) {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	// Viewer → 403
	req := httptest.NewRequest("POST", "/v1/datai/servers", nil)
	req = req.WithContext(userCtx(RoleViewer))
	rec := httptest.NewRecorder()
	writeHandler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("viewer POST: status = %d, want 403", rec.Code)
	}

	// User → 200
	req2 := httptest.NewRequest("POST", "/v1/datai/servers", nil)
	req2 = req2.WithContext(userCtx(RoleUser))
	rec2 := httptest.NewRecorder()
	writeHandler.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Errorf("user POST: status = %d, want 200", rec2.Code)
	}
}
