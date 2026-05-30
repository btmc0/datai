// Package logparser parses Pi/Claude terminal output into structured events.
// It handles both raw terminal output (with ANSI codes) and JSONL conversation format.
package logparser

import (
	"encoding/json"
	"regexp"
	"strings"
	"time"
)

// EventType represents the kind of parsed event.
type EventType string

const (
	EventThinking   EventType = "thinking"
	EventToolCall   EventType = "tool_call"
	EventToolResult EventType = "tool_result"
	EventText       EventType = "text"
	EventError      EventType = "error"
	EventStatus     EventType = "status"
	EventCommand    EventType = "command"
)

// Event represents a parsed structured event from terminal output.
type Event struct {
	Type      EventType         `json:"type"`
	Timestamp string            `json:"timestamp,omitempty"`
	Content   string            `json:"content"`
	Tool      string            `json:"tool,omitempty"`
	Status    string            `json:"status,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

var (
	ansiRe    = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
	toolCallRe = regexp.MustCompile(`^⏺\s+(.+)`)
	// Common tool names emitted by Claude/Pi
	toolNames = []string{
		"Read", "Write", "Edit", "Bash", "Grep",
		"find", "LS", "TodoRead", "TodoWrite",
		"WebFetch", "Glob", "Search",
	}
)

// StripANSI removes ANSI escape sequences from a string.
func StripANSI(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}

// Parse takes raw terminal output and returns structured events.
func Parse(raw string) []Event {
	lines := strings.Split(raw, "\n")
	var events []Event
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if ev := ParseLine(line); ev != nil {
			events = append(events, *ev)
		}
	}
	return events
}

// ParseLine parses a single line and returns an event if recognized.
func ParseLine(line string) *Event {
	stripped := StripANSI(line)
	trimmed := strings.TrimSpace(stripped)

	if trimmed == "" {
		return nil
	}

	now := time.Now().UTC().Format(time.RFC3339)

	// JSONL line
	if IsJSONL(trimmed) {
		return ParseJSONL(trimmed)
	}

	// Tool call indicator (⏺)
	if m := toolCallRe.FindStringSubmatch(trimmed); m != nil {
		ev := &Event{
			Type:      EventToolCall,
			Timestamp: now,
			Content:   m[1],
		}
		// Extract tool name if present
		for _, t := range toolNames {
			if strings.HasPrefix(m[1], t+" ") || strings.HasPrefix(m[1], t+"(") || m[1] == t {
				ev.Tool = t
				break
			}
		}
		return ev
	}

	// Error patterns
	if strings.HasPrefix(trimmed, "Error:") || strings.HasPrefix(trimmed, "error:") ||
		strings.Contains(trimmed, "ERROR") || strings.HasPrefix(trimmed, "E ") {
		return &Event{
			Type:      EventError,
			Timestamp: now,
			Content:   trimmed,
		}
	}

	// Thinking patterns
	if trimmed == "Thinking..." || trimmed == "thinking…" ||
		strings.HasPrefix(trimmed, "🤔") || strings.HasPrefix(trimmed, "<thinking>") {
		return &Event{
			Type:      EventThinking,
			Timestamp: now,
			Content:   trimmed,
		}
	}

	// Command patterns (shell prompts)
	if (strings.HasPrefix(trimmed, "$ ") || strings.HasPrefix(trimmed, "> ")) && len(trimmed) > 2 {
		return &Event{
			Type:      EventCommand,
			Timestamp: now,
			Content:   trimmed[2:],
		}
	}

	// Status patterns
	if strings.HasPrefix(trimmed, "Status:") || strings.HasPrefix(trimmed, "✓") ||
		strings.HasPrefix(trimmed, "✗") || strings.HasPrefix(trimmed, "●") {
		return &Event{
			Type:      EventStatus,
			Timestamp: now,
			Content:   trimmed,
			Status:    extractStatus(trimmed),
		}
	}

	// Default: text
	return &Event{
		Type:      EventText,
		Timestamp: now,
		Content:   trimmed,
	}
}

// IsJSONL checks if the input looks like JSONL format.
func IsJSONL(line string) bool {
	trimmed := strings.TrimSpace(line)
	if len(trimmed) < 2 {
		return false
	}
	return trimmed[0] == '{' && trimmed[len(trimmed)-1] == '}'
}

// ParseJSONL parses a JSONL line (Pi/Claude conversation format).
func ParseJSONL(line string) *Event {
	trimmed := strings.TrimSpace(line)
	var raw map[string]any
	if err := json.Unmarshal([]byte(trimmed), &raw); err != nil {
		return &Event{
			Type:    EventText,
			Content: trimmed,
		}
	}

	ev := &Event{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Metadata:  make(map[string]string),
	}

	// Extract role
	if role, ok := raw["role"].(string); ok {
		ev.Metadata["role"] = role
	}

	// Extract type from JSON
	if typ, ok := raw["type"].(string); ok {
		ev.Metadata["json_type"] = typ
		switch typ {
		case "tool_use", "tool_call":
			ev.Type = EventToolCall
			if name, ok := raw["name"].(string); ok {
				ev.Tool = name
			}
		case "tool_result":
			ev.Type = EventToolResult
		case "thinking":
			ev.Type = EventThinking
		case "error":
			ev.Type = EventError
		default:
			ev.Type = EventText
		}
	} else {
		// Infer from role
		role, _ := raw["role"].(string)
		switch role {
		case "assistant":
			ev.Type = EventText
		case "user":
			ev.Type = EventCommand
		case "system":
			ev.Type = EventStatus
		default:
			ev.Type = EventText
		}
	}

	// Extract content
	switch c := raw["content"].(type) {
	case string:
		ev.Content = c
	case []any:
		// Content blocks array (Claude format)
		var parts []string
		for _, block := range c {
			if m, ok := block.(map[string]any); ok {
				if text, ok := m["text"].(string); ok {
					parts = append(parts, text)
				}
			}
		}
		ev.Content = strings.Join(parts, "\n")
	default:
		b, _ := json.Marshal(raw)
		ev.Content = string(b)
	}

	return ev
}

func extractStatus(s string) string {
	switch {
	case strings.HasPrefix(s, "✓"):
		return "success"
	case strings.HasPrefix(s, "✗"):
		return "failure"
	case strings.HasPrefix(s, "●"):
		return "active"
	case strings.HasPrefix(s, "Status:"):
		return strings.TrimSpace(strings.TrimPrefix(s, "Status:"))
	default:
		return ""
	}
}
