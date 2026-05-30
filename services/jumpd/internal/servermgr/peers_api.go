package servermgr

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/sting8k/jump/services/jumpd/internal/config"
	"github.com/sting8k/jump/services/jumpd/internal/db"
	"github.com/sting8k/jump/services/jumpd/internal/jwtauth"
)

// PeerResponse merges DB peer data with live peering status.
type PeerResponse struct {
	db.DataiPeer
	LiveStatus   string `json:"live_status"`
	SessionCount int    `json:"session_count"`
}

// handlePeers handles GET (list) and POST (create) on /v1/datai/peers.
func (m *Manager) handlePeers(w http.ResponseWriter, r *http.Request) {
	userID := jwtauth.UserIDFromContext(r.Context())
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized", "missing user")
		return
	}

	switch r.Method {
	case http.MethodGet:
		peers, err := m.db.ListPeers(userID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "db_error", err.Error())
			return
		}

		// Build live status map from peering.Manager
		liveMap := make(map[string]struct {
			Status       string
			SessionCount int
		})
		if m.PeerManager != nil {
			for _, info := range m.PeerManager.PeerStatus() {
				liveMap[info.Name] = struct {
					Status       string
					SessionCount int
				}{Status: info.Status, SessionCount: info.SessionCount}
			}
		}

		results := make([]PeerResponse, len(peers))
		for i, p := range peers {
			resp := PeerResponse{DataiPeer: p}
			if live, ok := liveMap[p.Name]; ok {
				resp.LiveStatus = live.Status
				resp.SessionCount = live.SessionCount
			} else {
				resp.LiveStatus = "disconnected"
			}
			results[i] = resp
		}
		writeJSON(w, http.StatusOK, results)

	case http.MethodPost:
		var req db.PeerInput
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON")
			return
		}
		if req.Name == "" || req.TailscaleIP == "" {
			writeError(w, http.StatusBadRequest, "bad_request", "name and tailscale_ip are required")
			return
		}

		peer, err := m.db.CreatePeer(userID, req)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "db_error", err.Error())
			return
		}

		// Register with peering.Manager immediately
		if m.PeerManager != nil {
			m.PeerManager.AddPeer(config.PeerConfig{
				Name: peer.Name,
				URL:  fmt.Sprintf("http://%s:%d", peer.TailscaleIP, peer.Port),
			})
		}

		writeJSON(w, http.StatusCreated, peer)

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// handlePeerByID handles GET/DELETE on /v1/datai/peers/{id}.
func (m *Manager) handlePeerByID(w http.ResponseWriter, r *http.Request) {
	userID := jwtauth.UserIDFromContext(r.Context())
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized", "missing user")
		return
	}

	peerID := strings.TrimPrefix(r.URL.Path, "/v1/datai/peers/")
	if peerID == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	switch r.Method {
	case http.MethodGet:
		peer, err := m.db.GetPeer(userID, peerID)
		if err != nil {
			writeError(w, http.StatusNotFound, "not_found", err.Error())
			return
		}

		resp := PeerResponse{DataiPeer: *peer, LiveStatus: "disconnected"}
		if m.PeerManager != nil {
			for _, info := range m.PeerManager.PeerStatus() {
				if info.Name == peer.Name {
					resp.LiveStatus = info.Status
					resp.SessionCount = info.SessionCount
					break
				}
			}
		}
		writeJSON(w, http.StatusOK, resp)

	case http.MethodDelete:
		// Get name before deleting so we can remove from peering
		peer, err := m.db.GetPeer(userID, peerID)
		if err != nil {
			writeError(w, http.StatusNotFound, "not_found", err.Error())
			return
		}

		if err := m.db.DeletePeer(userID, peerID); err != nil {
			writeError(w, http.StatusNotFound, "not_found", err.Error())
			return
		}

		// Remove from peering.Manager
		if m.PeerManager != nil {
			m.PeerManager.RemovePeer(peer.Name)
		}

		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
