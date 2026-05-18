//go:build linux

package hostmetrics

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var powerSupplyRoot = "/sys/class/power_supply"

type cpuTimes struct {
	total uint64
	idle  uint64
}

func cpuPercent(ctx context.Context) (float64, error) {
	first, err := readCPUTimes()
	if err != nil {
		return 0, err
	}
	t := time.NewTimer(150 * time.Millisecond)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	case <-t.C:
	}
	second, err := readCPUTimes()
	if err != nil {
		return 0, err
	}
	total := second.total - first.total
	if total == 0 {
		return 0, nil
	}
	idle := second.idle - first.idle
	if idle > total {
		idle = total
	}
	return float64(total-idle) * 100 / float64(total), nil
}

func readCPUTimes() (cpuTimes, error) {
	b, err := os.ReadFile("/proc/stat")
	if err != nil {
		return cpuTimes{}, err
	}
	line, _, _ := strings.Cut(string(b), "\n")
	fields := strings.Fields(line)
	if len(fields) < 5 || fields[0] != "cpu" {
		return cpuTimes{}, fmt.Errorf("unexpected /proc/stat cpu line")
	}
	var total uint64
	var idle uint64
	for i, f := range fields[1:] {
		v, err := strconv.ParseUint(f, 10, 64)
		if err != nil {
			return cpuTimes{}, err
		}
		total += v
		if i == 3 || i == 4 {
			idle += v
		}
	}
	return cpuTimes{total: total, idle: idle}, nil
}

func batteryStatus(context.Context) (*BatteryStatus, error) {
	return readBatteryStatus(powerSupplyRoot)
}

func readBatteryStatus(root string) (*BatteryStatus, error) {
	entries, err := os.ReadDir(root)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dir := filepath.Join(root, entry.Name())
		typeText, _ := readTrimmedFile(filepath.Join(dir, "type"))
		if typeText != "" && !strings.EqualFold(typeText, "Battery") {
			continue
		}
		if typeText == "" && !strings.HasPrefix(strings.ToUpper(entry.Name()), "BAT") {
			continue
		}

		capacityText, err := readTrimmedFile(filepath.Join(dir, "capacity"))
		if err != nil {
			continue
		}
		percent, err := strconv.ParseFloat(capacityText, 64)
		if err != nil {
			continue
		}
		state, _ := readTrimmedFile(filepath.Join(dir, "status"))
		return &BatteryStatus{Percent: percent, State: state}, nil
	}
	return nil, nil
}

func readTrimmedFile(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

func memoryUsage(context.Context) (MemoryUsage, error) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return MemoryUsage{}, err
	}
	defer f.Close()

	var totalKB, availKB uint64
	s := bufio.NewScanner(f)
	for s.Scan() {
		fields := strings.Fields(s.Text())
		if len(fields) < 2 {
			continue
		}
		v, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			continue
		}
		switch fields[0] {
		case "MemTotal:":
			totalKB = v
		case "MemAvailable:":
			availKB = v
		}
	}
	if err := s.Err(); err != nil {
		return MemoryUsage{}, err
	}
	if totalKB == 0 {
		return MemoryUsage{}, fmt.Errorf("MemTotal missing from /proc/meminfo")
	}
	usedKB := totalKB
	if availKB < totalKB {
		usedKB = totalKB - availKB
	}
	return MemoryUsage{UsedBytes: usedKB * 1024, TotalBytes: totalKB * 1024}, nil
}
