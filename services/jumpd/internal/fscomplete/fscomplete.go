package fscomplete

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	MaxInputLen     = 1024
	DefaultMaxItems = 25
	HardMaxItems    = 50
)

var ErrUnsupportedPath = errors.New("path must start with / or ~")

type Suggestion struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

func Complete(input string, maxItems int) ([]Suggestion, error) {
	input = strings.TrimSpace(input)
	if input == "" || len(input) > MaxInputLen {
		return nil, ErrUnsupportedPath
	}
	if maxItems <= 0 {
		maxItems = DefaultMaxItems
	}
	if maxItems > HardMaxItems {
		maxItems = HardMaxItems
	}

	usesHomePrefix := input == "~" || strings.HasPrefix(input, "~/")
	if !usesHomePrefix && !strings.HasPrefix(input, "/") {
		return nil, ErrUnsupportedPath
	}

	expanded, err := expandInput(input)
	if err != nil {
		return nil, err
	}

	parent, prefix := splitCompletionTarget(expanded, input)
	entries, err := os.ReadDir(parent)
	if err != nil {
		return nil, err
	}

	lowerPrefix := strings.ToLower(prefix)
	allowHidden := strings.HasPrefix(prefix, ".")
	out := make([]Suggestion, 0, min(maxItems, len(entries)))
	for _, entry := range entries {
		name := entry.Name()
		if !entry.IsDir() {
			continue
		}
		if !allowHidden && strings.HasPrefix(name, ".") {
			continue
		}
		if lowerPrefix != "" && !strings.HasPrefix(strings.ToLower(name), lowerPrefix) {
			continue
		}

		candidate := filepath.Join(parent, name)
		out = append(out, Suggestion{Name: name, Path: displayPath(candidate, usesHomePrefix)})
		if len(out) >= maxItems {
			break
		}
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func expandInput(input string) (string, error) {
	if input == "~" {
		return os.UserHomeDir()
	}
	if strings.HasPrefix(input, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, input[2:]), nil
	}
	return filepath.Clean(input), nil
}

func splitCompletionTarget(expanded, raw string) (parent, prefix string) {
	if raw == "~" || strings.HasSuffix(raw, string(os.PathSeparator)) {
		return filepath.Clean(expanded), ""
	}
	if strings.HasSuffix(raw, string(os.PathSeparator)+".") {
		return filepath.Clean(expanded), "."
	}
	return filepath.Dir(expanded), filepath.Base(expanded)
}

func displayPath(abs string, preferHomePrefix bool) string {
	abs = filepath.Clean(abs)
	if !preferHomePrefix {
		return abs
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return abs
	}
	home = filepath.Clean(home)
	if abs == home {
		return "~"
	}
	if strings.HasPrefix(abs, home+string(os.PathSeparator)) {
		return "~" + abs[len(home):]
	}
	return abs
}
