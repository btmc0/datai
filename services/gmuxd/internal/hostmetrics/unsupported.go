//go:build !linux && !darwin

package hostmetrics

import (
	"context"
	"fmt"
)

func cpuPercent(context.Context) (float64, error) {
	return 0, fmt.Errorf("host metrics unsupported on this platform")
}

func memoryUsage(context.Context) (MemoryUsage, error) {
	return MemoryUsage{}, fmt.Errorf("host metrics unsupported on this platform")
}
