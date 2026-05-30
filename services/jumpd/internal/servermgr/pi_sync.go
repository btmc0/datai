package servermgr

import (
	"fmt"
	"strings"

	"golang.org/x/crypto/ssh"
)

// SyncPiConfig writes a single Pi config from the DB to the remote server
// via SSH, then marks it as synced.
func (m *Manager) SyncPiConfig(userID, serverID, configID string) error {
	// Verify the server belongs to the user.
	_, err := m.db.GetServer(userID, serverID)
	if err != nil {
		return fmt.Errorf("servermgr: get server: %w", err)
	}

	configs, err := m.db.ListPiConfigs(serverID)
	if err != nil {
		return fmt.Errorf("servermgr: list configs: %w", err)
	}

	var found bool
	for _, cfg := range configs {
		if cfg.ID == configID {
			found = true
			m.SSE.Broadcast(userID, SSEEvent{
				Type: "pi-sync",
				Data: map[string]string{"server_id": serverID, "config_id": configID, "name": cfg.Name, "status": "syncing"},
			})

			client, _, err := m.dialSSH(userID, serverID)
			if err != nil {
				m.SSE.Broadcast(userID, SSEEvent{
					Type: "pi-sync",
					Data: map[string]string{"server_id": serverID, "config_id": configID, "name": cfg.Name, "status": "failed", "error": err.Error()},
				})
				return err
			}
			defer client.Close()

			if err := writeRemoteFile(client, cfg.RemotePath, cfg.Content); err != nil {
				m.SSE.Broadcast(userID, SSEEvent{
					Type: "pi-sync",
					Data: map[string]string{"server_id": serverID, "config_id": configID, "name": cfg.Name, "status": "failed", "error": err.Error()},
				})
				return fmt.Errorf("servermgr: write config %s: %w", cfg.Name, err)
			}
			if err := m.db.MarkSynced(configID); err != nil {
				return err
			}
			m.SSE.Broadcast(userID, SSEEvent{
				Type: "pi-sync",
				Data: map[string]string{"server_id": serverID, "config_id": configID, "name": cfg.Name, "status": "synced"},
			})
			return nil
		}
	}
	if !found {
		return fmt.Errorf("servermgr: config %s not found for server %s", configID, serverID)
	}
	return nil
}

// SyncAllConfigs writes all Pi configs for a server to the remote machine.
// Configs that have no remote_path are skipped.
func (m *Manager) SyncAllConfigs(userID, serverID string) error {
	_, err := m.db.GetServer(userID, serverID)
	if err != nil {
		return fmt.Errorf("servermgr: get server: %w", err)
	}

	configs, err := m.db.ListPiConfigs(serverID)
	if err != nil {
		return fmt.Errorf("servermgr: list configs: %w", err)
	}
	if len(configs) == 0 {
		return nil
	}

	client, _, err := m.dialSSH(userID, serverID)
	if err != nil {
		return err
	}
	defer client.Close()

	total := 0
	for _, cfg := range configs {
		if cfg.RemotePath != "" {
			total++
		}
	}

	var errs []string
	done := 0
	for _, cfg := range configs {
		if cfg.RemotePath == "" {
			continue
		}
		m.SSE.Broadcast(userID, SSEEvent{
			Type: "pi-sync",
			Data: map[string]any{"server_id": serverID, "config_id": cfg.ID, "name": cfg.Name, "status": "syncing", "progress": done, "total": total},
		})
		if err := writeRemoteFile(client, cfg.RemotePath, cfg.Content); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", cfg.Name, err))
			m.SSE.Broadcast(userID, SSEEvent{
				Type: "pi-sync",
				Data: map[string]string{"server_id": serverID, "config_id": cfg.ID, "name": cfg.Name, "status": "failed", "error": err.Error()},
			})
			continue
		}
		if err := m.db.MarkSynced(cfg.ID); err != nil {
			errs = append(errs, fmt.Sprintf("%s (mark synced): %v", cfg.Name, err))
		}
		done++
	}
	if len(errs) > 0 {
		m.SSE.Broadcast(userID, SSEEvent{
			Type: "pi-sync",
			Data: map[string]any{"server_id": serverID, "status": "completed", "synced": done, "total": total, "errors": len(errs)},
		})
		return fmt.Errorf("servermgr: sync errors: %s", strings.Join(errs, "; "))
	}
	m.SSE.Broadcast(userID, SSEEvent{
		Type: "pi-sync",
		Data: map[string]any{"server_id": serverID, "status": "completed", "synced": done, "total": total, "errors": 0},
	})
	return nil
}

// writeRemoteFile creates parent directories and writes content to a file
// on the remote server via SSH.
func writeRemoteFile(client *ssh.Client, path, content string) error {
	if path == "" {
		return fmt.Errorf("empty remote path")
	}
	// Ensure parent directory exists, then write the file.
	// Use printf to avoid issues with special characters in content.
	dir := path[:strings.LastIndex(path, "/")]
	mkdirCmd := fmt.Sprintf("mkdir -p %s", shellQuote(dir))
	if _, err := runCommand(client, mkdirCmd); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}

	// Write via heredoc to handle multi-line content safely.
	writeCmd := fmt.Sprintf("cat > %s << 'DATAI_EOF'\n%s\nDATAI_EOF", shellQuote(path), content)
	if _, err := runCommand(client, writeCmd); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// shellQuote wraps a string in single quotes for safe shell argument use.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}
