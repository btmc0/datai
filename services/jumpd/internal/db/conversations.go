package db

import (
	"fmt"
	"time"
)

// Conversation represents a group of terminal sessions.
type Conversation struct {
	ID        string                `json:"id"`
	UserID    string                `json:"user_id"`
	Name      string                `json:"name"`
	CreatedAt time.Time             `json:"created_at"`
	UpdatedAt time.Time             `json:"updated_at"`
	Sessions  []ConversationSession `json:"sessions,omitempty"`
}

// ConversationSession links a session to a conversation with layout info.
type ConversationSession struct {
	SessionID    string  `json:"session_id"`
	ServerID     string  `json:"server_id"`
	Position     int     `json:"position"`
	WidthPercent float64 `json:"width_percent"`
}

// ConversationInput holds fields for creating or updating a conversation.
type ConversationInput struct {
	Name string `json:"name"`
}

// ConversationSessionInput holds fields for adding a session to a conversation.
type ConversationSessionInput struct {
	SessionID    string  `json:"session_id"`
	ServerID     string  `json:"server_id"`
	Position     int     `json:"position"`
	WidthPercent float64 `json:"width_percent"`
}

// ConversationSessionUpdate holds fields for updating a session's layout.
type ConversationSessionUpdate struct {
	Position     int     `json:"position"`
	WidthPercent float64 `json:"width_percent"`
}

// CreateConversation inserts a new conversation for a user.
func (d *DB) CreateConversation(userID string, input ConversationInput) (*Conversation, error) {
	id := newID()
	now := time.Now().UTC()
	_, err := d.db.Exec(
		`INSERT INTO conversations (id, user_id, name, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		id, userID, input.Name, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("db: insert conversation: %w", err)
	}
	return &Conversation{
		ID:        id,
		UserID:    userID,
		Name:      input.Name,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// ListConversations returns all conversations for a user (without sessions).
func (d *DB) ListConversations(userID string) ([]Conversation, error) {
	rows, err := d.db.Query(
		`SELECT id, user_id, name, created_at, updated_at
		 FROM conversations WHERE user_id = ? ORDER BY updated_at DESC`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("db: list conversations: %w", err)
	}
	defer rows.Close()
	var convs []Conversation
	for rows.Next() {
		var c Conversation
		if err := rows.Scan(&c.ID, &c.UserID, &c.Name, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("db: scan conversation: %w", err)
		}
		convs = append(convs, c)
	}
	return convs, rows.Err()
}

// GetConversation returns a conversation with its sessions, scoped to user.
func (d *DB) GetConversation(userID, id string) (*Conversation, error) {
	var c Conversation
	err := d.db.QueryRow(
		`SELECT id, user_id, name, created_at, updated_at
		 FROM conversations WHERE id = ? AND user_id = ?`, id, userID,
	).Scan(&c.ID, &c.UserID, &c.Name, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("db: get conversation %s: %w", id, err)
	}

	rows, err := d.db.Query(
		`SELECT session_id, COALESCE(server_id,''), position, width_percent
		 FROM conversation_sessions WHERE conversation_id = ? ORDER BY position`, id,
	)
	if err != nil {
		return nil, fmt.Errorf("db: list conversation sessions: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var cs ConversationSession
		if err := rows.Scan(&cs.SessionID, &cs.ServerID, &cs.Position, &cs.WidthPercent); err != nil {
			return nil, fmt.Errorf("db: scan conversation session: %w", err)
		}
		c.Sessions = append(c.Sessions, cs)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return &c, nil
}

// UpdateConversation updates a conversation's name.
func (d *DB) UpdateConversation(userID, id string, input ConversationInput) error {
	now := time.Now().UTC()
	res, err := d.db.Exec(
		`UPDATE conversations SET name=?, updated_at=? WHERE id=? AND user_id=?`,
		input.Name, now, id, userID,
	)
	if err != nil {
		return fmt.Errorf("db: update conversation %s: %w", id, err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("db: conversation %s not found", id)
	}
	return nil
}

// DeleteConversation removes a conversation and its session links (CASCADE).
func (d *DB) DeleteConversation(userID, id string) error {
	res, err := d.db.Exec(`DELETE FROM conversations WHERE id = ? AND user_id = ?`, id, userID)
	if err != nil {
		return fmt.Errorf("db: delete conversation %s: %w", id, err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("db: conversation %s not found", id)
	}
	return nil
}

// AddConversationSession adds a session to a conversation.
func (d *DB) AddConversationSession(convID string, input ConversationSessionInput) error {
	wp := input.WidthPercent
	if wp <= 0 {
		wp = 50.0
	}
	_, err := d.db.Exec(
		`INSERT INTO conversation_sessions (conversation_id, session_id, server_id, position, width_percent)
		 VALUES (?, ?, ?, ?, ?)`,
		convID, input.SessionID, nullStr(input.ServerID), input.Position, wp,
	)
	if err != nil {
		return fmt.Errorf("db: add conversation session: %w", err)
	}
	// Touch updated_at on parent conversation.
	_, _ = d.db.Exec(`UPDATE conversations SET updated_at=? WHERE id=?`, time.Now().UTC(), convID)
	return nil
}

// RemoveConversationSession removes a session from a conversation.
func (d *DB) RemoveConversationSession(convID, sessionID string) error {
	res, err := d.db.Exec(
		`DELETE FROM conversation_sessions WHERE conversation_id = ? AND session_id = ?`,
		convID, sessionID,
	)
	if err != nil {
		return fmt.Errorf("db: remove conversation session: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("db: conversation session %s not found", sessionID)
	}
	_, _ = d.db.Exec(`UPDATE conversations SET updated_at=? WHERE id=?`, time.Now().UTC(), convID)
	return nil
}

// UpdateConversationSession updates a session's layout within a conversation.
func (d *DB) UpdateConversationSession(convID, sessionID string, input ConversationSessionUpdate) error {
	wp := input.WidthPercent
	if wp <= 0 {
		wp = 50.0
	}
	res, err := d.db.Exec(
		`UPDATE conversation_sessions SET position=?, width_percent=?
		 WHERE conversation_id=? AND session_id=?`,
		input.Position, wp, convID, sessionID,
	)
	if err != nil {
		return fmt.Errorf("db: update conversation session layout: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("db: conversation session %s not found in %s", sessionID, convID)
	}
	_, _ = d.db.Exec(`UPDATE conversations SET updated_at=? WHERE id=?`, time.Now().UTC(), convID)
	return nil
}
