package logparser

import (
	"testing"
)

func TestStripANSI(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"hello", "hello"},
		{"\x1b[31mred\x1b[0m", "red"},
		{"\x1b[1;32mbold green\x1b[0m text", "bold green text"},
		{"\x1b[38;5;196mcolor\x1b[0m", "color"},
		{"no ansi here", "no ansi here"},
	}
	for _, tt := range tests {
		got := StripANSI(tt.input)
		if got != tt.want {
			t.Errorf("StripANSI(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestParseLineToolCall(t *testing.T) {
	tests := []struct {
		line     string
		wantType EventType
		wantTool string
	}{
		{"⏺ Read file.go", EventToolCall, "Read"},
		{"⏺ Write output.txt", EventToolCall, "Write"},
		{"⏺ Bash ls -la", EventToolCall, "Bash"},
		{"⏺ some other action", EventToolCall, ""},
	}
	for _, tt := range tests {
		ev := ParseLine(tt.line)
		if ev == nil {
			t.Fatalf("ParseLine(%q) returned nil", tt.line)
		}
		if ev.Type != tt.wantType {
			t.Errorf("ParseLine(%q).Type = %q, want %q", tt.line, ev.Type, tt.wantType)
		}
		if ev.Tool != tt.wantTool {
			t.Errorf("ParseLine(%q).Tool = %q, want %q", tt.line, ev.Tool, tt.wantTool)
		}
	}
}

func TestParseLineError(t *testing.T) {
	tests := []string{
		"Error: file not found",
		"error: compilation failed",
		"FATAL ERROR occurred",
	}
	for _, line := range tests {
		ev := ParseLine(line)
		if ev == nil {
			t.Fatalf("ParseLine(%q) returned nil", line)
		}
		if ev.Type != EventError {
			t.Errorf("ParseLine(%q).Type = %q, want %q", line, ev.Type, EventError)
		}
	}
}

func TestParseLineThinking(t *testing.T) {
	tests := []string{
		"Thinking...",
		"thinking…",
		"🤔 hmm",
		"<thinking>",
	}
	for _, line := range tests {
		ev := ParseLine(line)
		if ev == nil {
			t.Fatalf("ParseLine(%q) returned nil", line)
		}
		if ev.Type != EventThinking {
			t.Errorf("ParseLine(%q).Type = %q, want %q", line, ev.Type, EventThinking)
		}
	}
}

func TestParseLineCommand(t *testing.T) {
	ev := ParseLine("$ ls -la /tmp")
	if ev == nil {
		t.Fatal("ParseLine returned nil")
	}
	if ev.Type != EventCommand {
		t.Errorf("Type = %q, want %q", ev.Type, EventCommand)
	}
	if ev.Content != "ls -la /tmp" {
		t.Errorf("Content = %q, want %q", ev.Content, "ls -la /tmp")
	}

	ev2 := ParseLine("> git status")
	if ev2 == nil {
		t.Fatal("ParseLine returned nil")
	}
	if ev2.Type != EventCommand {
		t.Errorf("Type = %q, want %q", ev2.Type, EventCommand)
	}
}

func TestParseLineStatus(t *testing.T) {
	tests := []struct {
		line       string
		wantStatus string
	}{
		{"✓ done", "success"},
		{"✗ failed", "failure"},
		{"● running", "active"},
		{"Status: completed", "completed"},
	}
	for _, tt := range tests {
		ev := ParseLine(tt.line)
		if ev == nil {
			t.Fatalf("ParseLine(%q) returned nil", tt.line)
		}
		if ev.Type != EventStatus {
			t.Errorf("ParseLine(%q).Type = %q, want %q", tt.line, ev.Type, EventStatus)
		}
		if ev.Status != tt.wantStatus {
			t.Errorf("ParseLine(%q).Status = %q, want %q", tt.line, ev.Status, tt.wantStatus)
		}
	}
}

func TestParseLineText(t *testing.T) {
	ev := ParseLine("just some regular text output")
	if ev == nil {
		t.Fatal("returned nil")
	}
	if ev.Type != EventText {
		t.Errorf("Type = %q, want %q", ev.Type, EventText)
	}
}

func TestParseLineWithANSI(t *testing.T) {
	ev := ParseLine("\x1b[31mError: something broke\x1b[0m")
	if ev == nil {
		t.Fatal("returned nil")
	}
	if ev.Type != EventError {
		t.Errorf("Type = %q, want %q", ev.Type, EventError)
	}
	if ev.Content != "Error: something broke" {
		t.Errorf("Content = %q, want stripped ANSI", ev.Content)
	}
}

func TestIsJSONL(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{`{"role":"assistant","content":"hello"}`, true},
		{`{"type":"tool_use"}`, true},
		{`not json`, false},
		{`{incomplete`, false},
		{``, false},
		{`{}`, true},
	}
	for _, tt := range tests {
		got := IsJSONL(tt.input)
		if got != tt.want {
			t.Errorf("IsJSONL(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestParseJSONLToolCall(t *testing.T) {
	line := `{"type":"tool_use","name":"Read","content":"file.go"}`
	ev := ParseJSONL(line)
	if ev == nil {
		t.Fatal("returned nil")
	}
	if ev.Type != EventToolCall {
		t.Errorf("Type = %q, want %q", ev.Type, EventToolCall)
	}
	if ev.Tool != "Read" {
		t.Errorf("Tool = %q, want %q", ev.Tool, "Read")
	}
}

func TestParseJSONLAssistant(t *testing.T) {
	line := `{"role":"assistant","content":"Here is the answer"}`
	ev := ParseJSONL(line)
	if ev == nil {
		t.Fatal("returned nil")
	}
	if ev.Type != EventText {
		t.Errorf("Type = %q, want %q", ev.Type, EventText)
	}
	if ev.Content != "Here is the answer" {
		t.Errorf("Content = %q", ev.Content)
	}
}

func TestParseJSONLContentBlocks(t *testing.T) {
	line := `{"role":"assistant","content":[{"type":"text","text":"block1"},{"type":"text","text":"block2"}]}`
	ev := ParseJSONL(line)
	if ev == nil {
		t.Fatal("returned nil")
	}
	if ev.Content != "block1\nblock2" {
		t.Errorf("Content = %q, want %q", ev.Content, "block1\nblock2")
	}
}

func TestParseJSONLInvalid(t *testing.T) {
	ev := ParseJSONL("{invalid json")
	if ev == nil {
		t.Fatal("returned nil")
	}
	// Should fall back to text
	if ev.Type != EventText {
		t.Errorf("Type = %q, want %q", ev.Type, EventText)
	}
}

func TestParseMixedOutput(t *testing.T) {
	raw := `Thinking...
⏺ Read main.go
file contents here
$ ls -la
Error: not found
✓ done
{"role":"assistant","content":"hello"}
`
	events := Parse(raw)
	if len(events) != 7 {
		t.Fatalf("got %d events, want 7", len(events))
	}
	expected := []EventType{
		EventThinking,
		EventToolCall,
		EventText,
		EventCommand,
		EventError,
		EventStatus,
		EventText, // JSONL assistant
	}
	for i, want := range expected {
		if events[i].Type != want {
			t.Errorf("events[%d].Type = %q, want %q", i, events[i].Type, want)
		}
	}
}

func TestParseEmptyLine(t *testing.T) {
	ev := ParseLine("")
	if ev != nil {
		t.Errorf("expected nil for empty line, got %+v", ev)
	}
	ev2 := ParseLine("   ")
	if ev2 != nil {
		t.Errorf("expected nil for whitespace line, got %+v", ev2)
	}
}
