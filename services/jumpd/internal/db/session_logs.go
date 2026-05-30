package db

import (
	"fmt"
	"time"
)

// SessionLog represents a stored log entry for a terminal session.
type SessionLog struct {
	ID        int64     `json:"id"`
	SessionID string    `json:"session_id"`
	ServerID  string    `json:"server_id"`
	UserID    string    `json:"user_id"`
	LogType   string    `json:"log_type"` // raw, structured, event
	Content   string    `json:"content"`
	Metadata  string    `json:"metadata"` // JSON string
	CreatedAt time.Time `json:"created_at"`
}

// AppendLog inserts a new log entry for a session.
func (d *DB) AppendLog(userID, sessionID, serverID, logType, content, metadata string) error {
	_, err := d.db.Exec(
		`INSERT INTO session_logs (session_id, server_id, user_id, log_type, content, metadata)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		sessionID, nullStr(serverID), userID, logType, content, nullStr(metadata),
	)
	if err != nil {
		return fmt.Errorf("db: append log: %w", err)
	}
	return nil
}

// GetSessionLogs returns log entries for a session, optionally filtered by type.
// Pass empty logType to get all types. Results are ordered oldest-first.
func (d *DB) GetSessionLogs(userID, sessionID string, logType string, limit, offset int) ([]SessionLog, error) {
	if limit <= 0 {
		limit = 100
	}

	var query string
	var args []any

	if logType != "" {
		query = `SELECT id, session_id, COALESCE(server_id,''), user_id, log_type, content,
		                COALESCE(metadata,''), created_at
		         FROM session_logs
		         WHERE user_id = ? AND session_id = ? AND log_type = ?
		         ORDER BY id ASC LIMIT ? OFFSET ?`
		args = []any{userID, sessionID, logType, limit, offset}
	} else {
		query = `SELECT id, session_id, COALESCE(server_id,''), user_id, log_type, content,
		                COALESCE(metadata,''), created_at
		         FROM session_logs
		         WHERE user_id = ? AND session_id = ?
		         ORDER BY id ASC LIMIT ? OFFSET ?`
		args = []any{userID, sessionID, limit, offset}
	}

	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("db: get session logs: %w", err)
	}
	defer rows.Close()

	var logs []SessionLog
	for rows.Next() {
		var l SessionLog
		if err := rows.Scan(&l.ID, &l.SessionID, &l.ServerID, &l.UserID,
			&l.LogType, &l.Content, &l.Metadata, &l.CreatedAt); err != nil {
			return nil, fmt.Errorf("db: scan session log: %w", err)
		}
		logs = append(logs, l)
	}
	return logs, rows.Err()
}

// GetSessionLogCount returns the number of log entries for a session.
func (d *DB) GetSessionLogCount(userID, sessionID string) (int, error) {
	var count int
	err := d.db.QueryRow(
		`SELECT COUNT(*) FROM session_logs WHERE user_id = ? AND session_id = ?`,
		userID, sessionID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("db: count session logs: %w", err)
	}
	return count, nil
}

// DeleteSessionLogs removes all log entries for a session.
func (d *DB) DeleteSessionLogs(userID, sessionID string) error {
	_, err := d.db.Exec(
		`DELETE FROM session_logs WHERE user_id = ? AND session_id = ?`,
		userID, sessionID,
	)
	if err != nil {
		return fmt.Errorf("db: delete session logs: %w", err)
	}
	return nil
}
