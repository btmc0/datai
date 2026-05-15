package hostmetrics

import "context"

type Data struct {
	CPUPercent float64     `json:"cpu_percent"`
	Memory     MemoryUsage `json:"memory"`
}

type MemoryUsage struct {
	UsedBytes  uint64  `json:"used_bytes"`
	TotalBytes uint64  `json:"total_bytes"`
	Percent    float64 `json:"percent"`
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
	return Data{CPUPercent: round1(cpu), Memory: mem}, nil
}

func round1(v float64) float64 {
	return float64(int(v*10+0.5)) / 10
}
