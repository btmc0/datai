package servermgr

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/sting8k/jump/services/jumpd/internal/db"
	"github.com/sting8k/jump/services/jumpd/internal/jwtauth"
)

// handleConversations handles GET (list) and POST (create) on /v1/datai/conversations.
func (m *Manager) handleConversations(w http.ResponseWriter, r *http.Request) {
	userID := jwtauth.UserIDFromContext(r.Context())
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized", "missing user")
		return
	}
	switch r.Method {
	case http.MethodGet:
		convs, err := m.db.ListConversations(userID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "db_error", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, convs)
	case http.MethodPost:
		var req db.ConversationInput
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
			writeError(w, http.StatusBadRequest, "bad_request", "name is required")
			return
		}
		conv, err := m.db.CreateConversation(userID, req)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "db_error", err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, conv)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// handleConversationByID routes /v1/datai/conversations/{id}[/sessions[/{sessionId}]].
func (m *Manager) handleConversationByID(w http.ResponseWriter, r *http.Request) {
	userID := jwtauth.UserIDFromContext(r.Context())
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized", "missing user")
		return
	}

	// Parse: /v1/datai/conversations/{id}[/sessions[/{sessionId}]]
	path := strings.TrimPrefix(r.URL.Path, "/v1/datai/conversations/")
	parts := strings.SplitN(path, "/", 3)
	convID := parts[0]
	if convID == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "conversation id required")
		return
	}

	subPath := ""
	if len(parts) > 1 {
		subPath = strings.Join(parts[1:], "/")
	}

	switch {
	// GET /v1/datai/conversations/{id}
	case subPath == "" && r.Method == http.MethodGet:
		conv, err := m.db.GetConversation(userID, convID)
		if err != nil {
			writeError(w, http.StatusNotFound, "not_found", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, conv)

	// PUT /v1/datai/conversations/{id}
	case subPath == "" && r.Method == http.MethodPut:
		var req db.ConversationInput
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
			writeError(w, http.StatusBadRequest, "bad_request", "name is required")
			return
		}
		if err := m.db.UpdateConversation(userID, convID, req); err != nil {
			writeError(w, http.StatusNotFound, "not_found", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})

	// DELETE /v1/datai/conversations/{id}
	case subPath == "" && r.Method == http.MethodDelete:
		if err := m.db.DeleteConversation(userID, convID); err != nil {
			writeError(w, http.StatusNotFound, "not_found", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})

	// POST /v1/datai/conversations/{id}/sessions — add session
	case subPath == "sessions" && r.Method == http.MethodPost:
		// Verify conversation belongs to user.
		if _, err := m.db.GetConversation(userID, convID); err != nil {
			writeError(w, http.StatusNotFound, "not_found", err.Error())
			return
		}
		var req db.ConversationSessionInput
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.SessionID == "" {
			writeError(w, http.StatusBadRequest, "bad_request", "session_id is required")
			return
		}
		if err := m.db.AddConversationSession(convID, req); err != nil {
			writeError(w, http.StatusInternalServerError, "db_error", err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, map[string]bool{"ok": true})

	default:
		// Try to match sessions/{sessionId} sub-routes.
		m.handleConversationSessionByID(w, r, userID, convID, subPath)
	}
}

// handleConversationSessionByID handles DELETE and PUT on
// /v1/datai/conversations/{id}/sessions/{sessionId}.
func (m *Manager) handleConversationSessionByID(
	w http.ResponseWriter, r *http.Request,
	userID, convID, subPath string,
) {
	// Expect subPath = "sessions/{sessionId}"
	if !strings.HasPrefix(subPath, "sessions/") {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	sessionID := strings.TrimPrefix(subPath, "sessions/")
	if sessionID == "" || strings.Contains(sessionID, "/") {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// Verify conversation belongs to user.
	if _, err := m.db.GetConversation(userID, convID); err != nil {
		writeError(w, http.StatusNotFound, "not_found", err.Error())
		return
	}

	switch r.Method {
	// DELETE /v1/datai/conversations/{id}/sessions/{sessionId}
	case http.MethodDelete:
		if err := m.db.RemoveConversationSession(convID, sessionID); err != nil {
			writeError(w, http.StatusNotFound, "not_found", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})

	// PUT /v1/datai/conversations/{id}/sessions/{sessionId}
	case http.MethodPut:
		var req db.ConversationSessionUpdate
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON")
			return
		}
		if err := m.db.UpdateConversationSession(convID, sessionID, req); err != nil {
			writeError(w, http.StatusNotFound, "not_found", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
