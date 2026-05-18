//go:build !darwin

package hostactions

import (
	"context"
	"runtime"
)

func getDisplaySleepStatus(context.Context) DisplaySleepCapability {
	return DisplaySleepCapability{
		Available: false,
		Status:    "unsupported",
		Platform:  runtime.GOOS,
		State:     "unknown",
		Reason:    "display sleep is only available on macOS",
	}
}

func sleepDisplay(ctx context.Context) (DisplaySleepCapability, error) {
	return getDisplaySleepStatus(ctx), ErrUnavailable
}
