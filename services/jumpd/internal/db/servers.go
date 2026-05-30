package db

import (
	"database/sql"
	"fmt"
	"time"
)

// Server represents a remote server managed by datai.
type Server struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	GroupID     string    `json:"group_id"`
	Name        string    `json:"name"`
	Host        string    `json:"host"`
	Port        int       `json:"port"`
	Username    string    `json:"username"`
	SSHKeyID    string    `json:"ssh_key_id"`
	PiInstalled bool      `json:"pi_installed"`
	PiVersion   string    `json:"pi_version"`
	CreatedAt   time.Time `json:"created_at"`
}

// ServerInput holds the fields for creating or updating a server.
type ServerInput struct {
	GroupID  string `json:"group_id"`
	Name     string `json:"name"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	SSHKeyID string `json:"ssh_key_id"`
}

// CreateServer inserts a new server for a user.
func (d *DB) CreateServer(userID string, s ServerInput) (*Server, error) {
	id := newID()
	port := s.Port
	if port == 0 {
		port = 22
	}
	now := time.Now().UTC()
	_, err := d.db.Exec(
		`INSERT INTO servers (id, user_id, group_id, name, host, port, username, ssh_key_id, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, userID, nullStr(s.GroupID), s.Name, s.Host, port, s.Username, nullStr(s.SSHKeyID), now,
	)
	if err != nil {
		return nil, fmt.Errorf("db: insert server: %w", err)
	}
	return &Server{
		ID:       id,
		UserID:   userID,
		GroupID:  s.GroupID,
		Name:     s.Name,
		Host:     s.Host,
		Port:     port,
		Username: s.Username,
		SSHKeyID: s.SSHKeyID,
		CreatedAt: now,
	}, nil
}

// ListServers returns all servers for a user.
func (d *DB) ListServers(userID string) ([]Server, error) {
	rows, err := d.db.Query(
		`SELECT id, user_id, COALESCE(group_id,''), name, host, port, username,
		        COALESCE(ssh_key_id,''), pi_installed, COALESCE(pi_version,''), created_at
		 FROM servers WHERE user_id = ? ORDER BY created_at DESC`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("db: list servers: %w", err)
	}
	defer rows.Close()
	var servers []Server
	for rows.Next() {
		var s Server
		if err := rows.Scan(&s.ID, &s.UserID, &s.GroupID, &s.Name, &s.Host, &s.Port,
			&s.Username, &s.SSHKeyID, &s.PiInstalled, &s.PiVersion, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("db: scan server: %w", err)
		}
		servers = append(servers, s)
	}
	return servers, rows.Err()
}

// GetServer returns a single server by ID, scoped to user.
func (d *DB) GetServer(userID, id string) (*Server, error) {
	var s Server
	err := d.db.QueryRow(
		`SELECT id, user_id, COALESCE(group_id,''), name, host, port, username,
		        COALESCE(ssh_key_id,''), pi_installed, COALESCE(pi_version,''), created_at
		 FROM servers WHERE id = ? AND user_id = ?`, id, userID,
	).Scan(&s.ID, &s.UserID, &s.GroupID, &s.Name, &s.Host, &s.Port,
		&s.Username, &s.SSHKeyID, &s.PiInstalled, &s.PiVersion, &s.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("db: get server %s: %w", id, err)
	}
	return &s, nil
}

// UpdateServer updates an existing server's fields.
func (d *DB) UpdateServer(userID, id string, s ServerInput) error {
	port := s.Port
	if port == 0 {
		port = 22
	}
	res, err := d.db.Exec(
		`UPDATE servers SET group_id=?, name=?, host=?, port=?, username=?, ssh_key_id=?
		 WHERE id=? AND user_id=?`,
		nullStr(s.GroupID), s.Name, s.Host, port, s.Username, nullStr(s.SSHKeyID), id, userID,
	)
	if err != nil {
		return fmt.Errorf("db: update server %s: %w", id, err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("db: server %s not found", id)
	}
	return nil
}

// DeleteServer removes a server by ID, scoped to user.
func (d *DB) DeleteServer(userID, id string) error {
	res, err := d.db.Exec(`DELETE FROM servers WHERE id = ? AND user_id = ?`, id, userID)
	if err != nil {
		return fmt.Errorf("db: delete server %s: %w", id, err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("db: server %s not found", id)
	}
	return nil
}

// UpdateServerPiStatus sets the Pi installation status for a server.
func (d *DB) UpdateServerPiStatus(userID, id string, installed bool, version string) error {
	_, err := d.db.Exec(
		`UPDATE servers SET pi_installed=?, pi_version=? WHERE id=? AND user_id=?`,
		installed, version, id, userID,
	)
	if err != nil {
		return fmt.Errorf("db: update pi status %s: %w", id, err)
	}
	return nil
}

// nullStr converts an empty string to sql.NullString for nullable columns.
func nullStr(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}
