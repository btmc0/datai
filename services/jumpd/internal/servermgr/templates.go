package servermgr

import (
	"encoding/json"
	"fmt"

	"github.com/sting8k/jump/services/jumpd/internal/db"
)

// TemplateConfig is the structured form of a Pi template's config_data JSON.
type TemplateConfig struct {
	SystemPrompt string            `json:"system_prompt"`
	Skills       []string          `json:"skills"`
	Settings     map[string]string `json:"settings,omitempty"`
}

// ApplyTemplate creates Pi configs on a server based on a template.
// It reads the template's config_data, creates a system_prompt config and
// individual skill configs for the server.
func (m *Manager) ApplyTemplate(userID, serverID, templateID string) error {
	srv, err := m.db.GetServer(userID, serverID)
	if err != nil {
		return fmt.Errorf("servermgr: get server: %w", err)
	}

	templates, err := m.db.ListTemplates(userID)
	if err != nil {
		return fmt.Errorf("servermgr: list templates: %w", err)
	}

	var tmpl *db.PiTemplate
	for _, t := range templates {
		if t.ID == templateID {
			tmpl = &t
			break
		}
	}
	if tmpl == nil {
		return fmt.Errorf("servermgr: template %s not found", templateID)
	}

	var cfg TemplateConfig
	if err := json.Unmarshal([]byte(tmpl.ConfigData), &cfg); err != nil {
		return fmt.Errorf("servermgr: parse template config: %w", err)
	}

	// Create system prompt config.
	if cfg.SystemPrompt != "" {
		_, err := m.db.SavePiConfig(srv.ID, db.PiConfigInput{
			ConfigType: "system_prompt",
			Name:       "System Prompt",
			Content:    cfg.SystemPrompt,
			RemotePath: fmt.Sprintf("/home/%s/.config/pi/system-prompt.md", srv.Username),
		})
		if err != nil {
			return fmt.Errorf("servermgr: save system prompt: %w", err)
		}
	}

	// Create skill configs.
	for _, skill := range cfg.Skills {
		_, err := m.db.SavePiConfig(srv.ID, db.PiConfigInput{
			ConfigType: "skill",
			Name:       skill,
			Content:    skill, // Skill name as placeholder; user can edit later.
			RemotePath: fmt.Sprintf("/home/%s/.config/pi/skills/%s", srv.Username, skill),
		})
		if err != nil {
			return fmt.Errorf("servermgr: save skill %s: %w", skill, err)
		}
	}

	return nil
}
