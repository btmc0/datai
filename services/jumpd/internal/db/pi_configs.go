package db

import (
	"fmt"
	"time"
)

// PiConfig represents a Pi agent configuration stored in datai.
type PiConfig struct {
	ID         string     `json:"id"`
	ServerID   string     `json:"server_id"`
	ConfigType string     `json:"config_type"` // "system_prompt", "skill", "project_system"
	Name       string     `json:"name"`
	Content    string     `json:"content"`
	RemotePath string     `json:"remote_path"`
	SyncedAt   *time.Time `json:"synced_at"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// PiConfigInput holds the fields for creating or updating a Pi config.
type PiConfigInput struct {
	ConfigType string `json:"config_type"`
	Name       string `json:"name"`
	Content    string `json:"content"`
	RemotePath string `json:"remote_path"`
}

// SavePiConfig inserts or updates a Pi config for a server.
func (d *DB) SavePiConfig(serverID string, cfg PiConfigInput) (*PiConfig, error) {
	id := newID()
	now := time.Now().UTC()
	_, err := d.db.Exec(
		`INSERT INTO pi_configs (id, server_id, config_type, name, content, remote_path, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		id, serverID, cfg.ConfigType, cfg.Name, cfg.Content, cfg.RemotePath, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("db: insert pi_config: %w", err)
	}
	return &PiConfig{
		ID:         id,
		ServerID:   serverID,
		ConfigType: cfg.ConfigType,
		Name:       cfg.Name,
		Content:    cfg.Content,
		RemotePath: cfg.RemotePath,
		CreatedAt:  now,
		UpdatedAt:  now,
	}, nil
}

// UpdatePiConfig updates an existing Pi config's content.
func (d *DB) UpdatePiConfig(id string, cfg PiConfigInput) error {
	now := time.Now().UTC()
	res, err := d.db.Exec(
		`UPDATE pi_configs SET config_type=?, name=?, content=?, remote_path=?, updated_at=?, synced_at=NULL
		 WHERE id=?`,
		cfg.ConfigType, cfg.Name, cfg.Content, cfg.RemotePath, now, id,
	)
	if err != nil {
		return fmt.Errorf("db: update pi_config %s: %w", id, err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("db: pi_config %s not found", id)
	}
	return nil
}

// ListPiConfigs returns all Pi configs for a server.
func (d *DB) ListPiConfigs(serverID string) ([]PiConfig, error) {
	rows, err := d.db.Query(
		`SELECT id, server_id, config_type, name, content, COALESCE(remote_path,''),
		        synced_at, created_at, updated_at
		 FROM pi_configs WHERE server_id = ? ORDER BY config_type, name`, serverID,
	)
	if err != nil {
		return nil, fmt.Errorf("db: list pi_configs: %w", err)
	}
	defer rows.Close()
	var configs []PiConfig
	for rows.Next() {
		var c PiConfig
		if err := rows.Scan(&c.ID, &c.ServerID, &c.ConfigType, &c.Name, &c.Content,
			&c.RemotePath, &c.SyncedAt, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("db: scan pi_config: %w", err)
		}
		configs = append(configs, c)
	}
	return configs, rows.Err()
}

// DeletePiConfig removes a Pi config by ID.
func (d *DB) DeletePiConfig(id string) error {
	res, err := d.db.Exec(`DELETE FROM pi_configs WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("db: delete pi_config %s: %w", id, err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("db: pi_config %s not found", id)
	}
	return nil
}

// MarkSynced updates the synced_at timestamp for a Pi config.
func (d *DB) MarkSynced(id string) error {
	now := time.Now().UTC()
	_, err := d.db.Exec(`UPDATE pi_configs SET synced_at = ? WHERE id = ?`, now, id)
	if err != nil {
		return fmt.Errorf("db: mark synced %s: %w", id, err)
	}
	return nil
}
