//go:build darwin

package hostactions

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

var pmsetPath = "/usr/bin/pmset"

func getDisplaySleepStatus(ctx context.Context) DisplaySleepCapability {
	if !isExecutable(pmsetPath) {
		return DisplaySleepCapability{
			Available: false,
			Status:    "unavailable",
			Platform:  "darwin",
			State:     "unknown",
			Reason:    "pmset is not available",
		}
	}
	return DisplaySleepCapability{
		Available: true,
		Status:    "available",
		Platform:  "darwin",
		State:     currentDisplaySleepState(ctx),
	}
}

func sleepDisplay(ctx context.Context) (DisplaySleepCapability, error) {
	status := getDisplaySleepStatus(ctx)
	if !status.Available {
		return status, ErrUnavailable
	}

	cmdCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	out, err := exec.CommandContext(cmdCtx, pmsetPath, "displaysleepnow").CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if len(msg) > 512 {
			msg = msg[:512] + "…"
		}
		if msg != "" {
			return status, fmt.Errorf("pmset displaysleepnow: %w: %s", err, msg)
		}
		return status, fmt.Errorf("pmset displaysleepnow: %w", err)
	}
	return getDisplaySleepStatus(ctx), nil
}

func isExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	return info.Mode()&0o111 != 0
}
