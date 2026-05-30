package servermgr

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/sting8k/jump/services/jumpd/internal/db"
	"github.com/sting8k/jump/services/jumpd/internal/jwtauth"
	"github.com/sting8k/jump/services/jumpd/internal/logparser"
)

// RegisterRoutes mounts all datai management endpoints on the given mux.
func (m *Manager) RegisterRoutes(mux *http.ServeMux) {
	// SSH Keys
	mux.HandleFunc("/v1/datai/ssh-keys", m.handleSSHKeys)

	// Servers
	mux.HandleFunc("/v1/datai/servers", m.handleServers)
	mux.HandleFunc("/v1/datai/servers/", m.handleServerByID)

	// Templates
	mux.HandleFunc("/v1/datai/templates", m.handleTemplates)

	// Conversations
	mux.HandleFunc("/v1/datai/conversations", m.handleConversations)
	mux.HandleFunc("/v1/datai/conversations/", m.handleConversationByID)

	// Peers
	mux.HandleFunc("/v1/datai/peers", m.handlePeers)
	mux.HandleFunc("/v1/datai/peers/", m.handlePeerByID)

	// Session Logs
	mux.HandleFunc("/v1/datai/sessions/", m.handleSessionLogs)

	// SSE events
	mux.HandleFunc("/v1/datai/events", m.SSE.Handler())
}

// --- SSH Keys ---

func (m *Manager) handleSSHKeys(w http.ResponseWriter, r *http.Request) {
	userID := jwtauth.UserIDFromContext(r.Context())
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized", "missing user")
		return
	}
	switch r.Method {
	case http.MethodGet:
		keys, err := m.db.ListSSHKeys(userID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "db_error", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, keys)
	case http.MethodPost:
		var req struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
			writeError(w, http.StatusBadRequest, "bad_request", "name is required")
			return
		}
		key, err := m.GenerateSSHKey(userID, req.Name)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "generate_error", err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, key)
	case http.MethodDelete:
		id := r.URL.Query().Get("id")
		if id == "" {
			writeError(w, http.StatusBadRequest, "bad_request", "id is required")
			return
		}
		if err := m.db.DeleteSSHKey(userID, id); err != nil {
			writeError(w, http.StatusNotFound, "not_found", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// --- Servers ---

func (m *Manager) handleServers(w http.ResponseWriter, r *http.Request) {
	userID := jwtauth.UserIDFromContext(r.Context())
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized", "missing user")
		return
	}
	switch r.Method {
	case http.MethodGet:
		servers, err := m.db.ListServers(userID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "db_error", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, servers)
	case http.MethodPost:
		var req db.ServerInput
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON")
			return
		}
		if req.Name == "" || req.Host == "" || req.Username == "" {
			writeError(w, http.StatusBadRequest, "bad_request", "name, host, username are required")
			return
		}
		srv, err := m.db.CreateServer(userID, req)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "db_error", err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, srv)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (m *Manager) handleServerByID(w http.ResponseWriter, r *http.Request) {
	userID := jwtauth.UserIDFromContext(r.Context())
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized", "missing user")
		return
	}

	// Parse: /v1/datai/servers/{id}[/sub/path]
	path := strings.TrimPrefix(r.URL.Path, "/v1/datai/servers/")
	parts := strings.SplitN(path, "/", 2)
	serverID := parts[0]
	subPath := ""
	if len(parts) > 1 {
		subPath = parts[1]
	}

	switch {
	case subPath == "" && r.Method == http.MethodGet:
		srv, err := m.db.GetServer(userID, serverID)
		if err != nil {
			writeError(w, http.StatusNotFound, "not_found", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, srv)

	case subPath == "" && r.Method == http.MethodPut:
		var req db.ServerInput
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON")
			return
		}
		if err := m.db.UpdateServer(userID, serverID, req); err != nil {
			writeError(w, http.StatusNotFound, "not_found", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})

	case subPath == "" && r.Method == http.MethodDelete:
		if err := m.db.DeleteServer(userID, serverID); err != nil {
			writeError(w, http.StatusNotFound, "not_found", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})

	case subPath == "test" && r.Method == http.MethodPost:
		if err := m.TestConnection(userID, serverID); err != nil {
			writeError(w, http.StatusBadGateway, "connection_failed", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "connected"})

	case subPath == "pi/check" && r.Method == http.MethodPost:
		status, err := m.CheckPi(userID, serverID)
		if err != nil {
			writeError(w, http.StatusBadGateway, "pi_check_failed", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, status)

	case subPath == "pi/install" && r.Method == http.MethodPost:
		if err := m.InstallPi(userID, serverID); err != nil {
			writeError(w, http.StatusBadGateway, "pi_install_failed", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "installed"})

	case subPath == "pi/configs" && r.Method == http.MethodGet:
		configs, err := m.db.ListPiConfigs(serverID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "db_error", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, configs)

	case subPath == "pi/configs" && r.Method == http.MethodPost:
		var req db.PiConfigInput
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON")
			return
		}
		if req.ConfigType == "" || req.Name == "" || req.Content == "" {
			writeError(w, http.StatusBadRequest, "bad_request", "config_type, name, content are required")
			return
		}
		cfg, err := m.db.SavePiConfig(serverID, req)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "db_error", err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, cfg)

	case subPath == "pi/sync" && r.Method == http.MethodPost:
		if err := m.SyncAllConfigs(userID, serverID); err != nil {
			writeError(w, http.StatusBadGateway, "sync_failed", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "synced"})

	case subPath == "pi/template" && r.Method == http.MethodPost:
		var req struct {
			TemplateID string `json:"template_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TemplateID == "" {
			writeError(w, http.StatusBadRequest, "bad_request", "template_id is required")
			return
		}
		if err := m.ApplyTemplate(userID, serverID, req.TemplateID); err != nil {
			writeError(w, http.StatusInternalServerError, "template_error", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "applied"})

	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

// --- Templates ---

func (m *Manager) handleTemplates(w http.ResponseWriter, r *http.Request) {
	userID := jwtauth.UserIDFromContext(r.Context())
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized", "missing user")
		return
	}
	switch r.Method {
	case http.MethodGet:
		templates, err := m.db.ListTemplates(userID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "db_error", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, templates)
	case http.MethodPost:
		var req db.TemplateInput
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON")
			return
		}
		if req.Name == "" || req.ConfigData == "" {
			writeError(w, http.StatusBadRequest, "bad_request", "name and config_data are required")
			return
		}
		tmpl, err := m.db.CreateTemplate(userID, req)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "db_error", err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, tmpl)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// --- Session Logs ---

func (m *Manager) handleSessionLogs(w http.ResponseWriter, r *http.Request) {
	userID := jwtauth.UserIDFromContext(r.Context())
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized", "missing user")
		return
	}

	// Parse: /v1/datai/sessions/{sessionID}/logs[/parsed]
	path := strings.TrimPrefix(r.URL.Path, "/v1/datai/sessions/")
	parts := strings.SplitN(path, "/", 3)
	if len(parts) < 2 || parts[1] != "logs" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	sessionID := parts[0]
	subPath := ""
	if len(parts) > 2 {
		subPath = parts[2]
	}

	switch {
	case subPath == "" && r.Method == http.MethodGet:
		logType := r.URL.Query().Get("type")
		limit := 100
		offset := 0
		if v := r.URL.Query().Get("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				limit = n
			}
		}
		if v := r.URL.Query().Get("offset"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n >= 0 {
				offset = n
			}
		}
		logs, err := m.db.GetSessionLogs(userID, sessionID, logType, limit, offset)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "db_error", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, logs)

	case subPath == "" && r.Method == http.MethodPost:
		var req struct {
			ServerID string `json:"server_id"`
			LogType  string `json:"log_type"`
			Content  string `json:"content"`
			Metadata string `json:"metadata"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Content == "" {
			writeError(w, http.StatusBadRequest, "bad_request", "content is required")
			return
		}
		if req.LogType == "" {
			req.LogType = "raw"
		}
		if err := m.db.AppendLog(userID, sessionID, req.ServerID, req.LogType, req.Content, req.Metadata); err != nil {
			writeError(w, http.StatusInternalServerError, "db_error", err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, map[string]bool{"ok": true})

	case subPath == "" && r.Method == http.MethodDelete:
		if err := m.db.DeleteSessionLogs(userID, sessionID); err != nil {
			writeError(w, http.StatusInternalServerError, "db_error", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})

	case subPath == "parsed" && r.Method == http.MethodGet:
		logs, err := m.db.GetSessionLogs(userID, sessionID, "", 1000, 0)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "db_error", err.Error())
			return
		}
		var allContent strings.Builder
		for _, l := range logs {
			allContent.WriteString(l.Content)
			allContent.WriteByte('\n')
		}
		events := logparser.Parse(allContent.String())
		writeJSON(w, http.StatusOK, events)

	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

// --- Helpers ---

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok": false,
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}
