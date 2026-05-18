package hostmetrics

import (
	"context"
	"strings"
)

type Data struct {
	CPUPercent float64        `json:"cpu_percent"`
	Memory     MemoryUsage    `json:"memory"`
	Battery    *BatteryStatus `json:"battery,omitempty"`
}

type MemoryUsage struct {
	UsedBytes  uint64  `json:"used_bytes"`
	TotalBytes uint64  `json:"total_bytes"`
	Percent    float64 `json:"percent"`
}

type BatteryStatus struct {
	Percent float64 `json:"percent"`
	State   string  `json:"state,omitempty"`
}

func Collect(ctx context.Context) (Data, error) {
	cpu, err := cpuPercent(ctx)
	if err != nil {
		return Data{}, err
	}
	mem, err := memoryUsage(ctx)
	if err != nil {
		return Data{}, err
	}
	if mem.TotalBytes > 0 {
		mem.Percent = round1(float64(mem.UsedBytes) * 100 / float64(mem.TotalBytes))
	}
	battery, _ := batteryStatus(ctx)
	if battery != nil {
		battery.Percent = round1(clampPercent(battery.Percent))
		battery.State = normalizeBatteryState(battery.State)
	}
	return Data{CPUPercent: round1(cpu), Memory: mem, Battery: battery}, nil
}

func clampPercent(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}

func normalizeBatteryState(state string) string {
	s := strings.ToLower(strings.TrimSpace(state))
	s = strings.ReplaceAll(s, "_", " ")
	s = strings.Join(strings.Fields(s), " ")
	return s
}

func round1(v float64) float64 {
	return float64(int(v*10+0.5)) / 10
}
