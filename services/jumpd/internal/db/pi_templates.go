package db

import (
	"fmt"
	"time"
)

// PiTemplate represents a reusable Pi agent configuration template.
type PiTemplate struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	ConfigData  string    `json:"config_data"` // JSON: {system_prompt, skills, settings}
	IsBuiltin   bool      `json:"is_builtin"`
	UserID      string    `json:"user_id"`
	CreatedAt   time.Time `json:"created_at"`
}

// TemplateInput holds fields for creating a custom template.
type TemplateInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	ConfigData  string `json:"config_data"`
}

// ListTemplates returns builtin templates plus any custom templates for a user.
func (d *DB) ListTemplates(userID string) ([]PiTemplate, error) {
	rows, err := d.db.Query(
		`SELECT id, name, COALESCE(description,''), config_data, is_builtin,
		        COALESCE(user_id,''), created_at
		 FROM pi_templates
		 WHERE is_builtin = true OR user_id = ?
		 ORDER BY is_builtin DESC, name`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("db: list templates: %w", err)
	}
	defer rows.Close()
	var templates []PiTemplate
	for rows.Next() {
		var t PiTemplate
		if err := rows.Scan(&t.ID, &t.Name, &t.Description, &t.ConfigData,
			&t.IsBuiltin, &t.UserID, &t.CreatedAt); err != nil {
			return nil, fmt.Errorf("db: scan template: %w", err)
		}
		templates = append(templates, t)
	}
	return templates, rows.Err()
}

// CreateTemplate inserts a custom template for a user.
func (d *DB) CreateTemplate(userID string, t TemplateInput) (*PiTemplate, error) {
	id := newID()
	now := time.Now().UTC()
	_, err := d.db.Exec(
		`INSERT INTO pi_templates (id, name, description, config_data, is_builtin, user_id, created_at)
		 VALUES (?, ?, ?, ?, false, ?, ?)`,
		id, t.Name, t.Description, t.ConfigData, userID, now,
	)
	if err != nil {
		return nil, fmt.Errorf("db: insert template: %w", err)
	}
	return &PiTemplate{
		ID:          id,
		Name:        t.Name,
		Description: t.Description,
		ConfigData:  t.ConfigData,
		UserID:      userID,
		CreatedAt:   now,
	}, nil
}

// DeleteTemplate removes a custom template. Builtin templates cannot be deleted.
func (d *DB) DeleteTemplate(userID, id string) error {
	res, err := d.db.Exec(
		`DELETE FROM pi_templates WHERE id = ? AND user_id = ? AND is_builtin = false`, id, userID,
	)
	if err != nil {
		return fmt.Errorf("db: delete template %s: %w", id, err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("db: template %s not found or is builtin", id)
	}
	return nil
}

// SeedBuiltinTemplates inserts default templates if they don't exist.
func (d *DB) SeedBuiltinTemplates() error {
	builtins := []struct {
		id, name, desc, data string
	}{
		{
			"builtin-coding-assistant",
			"Coding Assistant",
			"General coding: refactoring, debugging, code review",
			`{"system_prompt":"You are a coding assistant. Help with code review, refactoring, debugging, and writing clean code.","skills":["code-review","refactor","debug"],"settings":{}}`,
		},
		{
			"builtin-devops",
			"DevOps",
			"Infrastructure, deployment, CI/CD, monitoring",
			`{"system_prompt":"You are a DevOps engineer. Help with infrastructure, deployment pipelines, monitoring, and system administration.","skills":["docker","k8s","ci-cd","monitoring"],"settings":{}}`,
		},
		{
			"builtin-data-engineering",
			"Data Engineering",
			"Data pipelines, ETL, data analysis, SQL",
			`{"system_prompt":"You are a data engineer. Help with data pipelines, ETL processes, SQL optimization, and data analysis.","skills":["sql","etl","data-analysis"],"settings":{}}`,
		},
	}
	for _, b := range builtins {
		_, err := d.db.Exec(
			`INSERT OR IGNORE INTO pi_templates (id, name, description, config_data, is_builtin, created_at)
			 VALUES (?, ?, ?, ?, true, CURRENT_TIMESTAMP)`,
			b.id, b.name, b.desc, b.data,
		)
		if err != nil {
			return fmt.Errorf("db: seed template %s: %w", b.name, err)
		}
	}
	return nil
}
