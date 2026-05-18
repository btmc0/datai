//go:build darwin && !cgo

package hostactions

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

func currentDisplaySleepState(ctx context.Context) string {
	cmdCtx, cancel := context.WithTimeout(ctx, 800*time.Millisecond)
	defer cancel()

	out, err := exec.CommandContext(cmdCtx, pmsetPath, "-g", "powerstate", "IODisplayWrangler").CombinedOutput()
	if err != nil {
		return "unknown"
	}
	return parseDisplaySleepState(string(out))
}

func parseDisplaySleepState(output string) string {
	lower := strings.ToLower(output)
	if strings.Contains(lower, "internal failure") || strings.TrimSpace(lower) == "" {
		return "unknown"
	}
	for _, line := range strings.Split(lower, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.Contains(line, "current state description") {
			continue
		}
		if !strings.Contains(line, "iodisplaywrangler") && !strings.Contains(line, "display is") {
			continue
		}
		if strings.Contains(line, "asleep") || strings.Contains(line, "sleeping") || strings.Contains(line, " off") || strings.HasSuffix(line, " off") {
			return "asleep"
		}
		if strings.Contains(line, "awake") || strings.Contains(line, " on") || strings.HasSuffix(line, " on") || strings.Contains(line, "usable") || strings.Contains(line, "running") {
			return "awake"
		}
	}
	return "unknown"
}
