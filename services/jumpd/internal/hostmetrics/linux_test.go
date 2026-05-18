//go:build linux

package hostmetrics

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRound1(t *testing.T) {
	if got := round1(12.34); got != 12.3 {
		t.Fatalf("round1 = %v, want 12.3", got)
	}
}

func TestReadBatteryStatus(t *testing.T) {
	root := t.TempDir()
	bat := filepath.Join(root, "BAT0")
	if err := os.Mkdir(bat, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, value := range map[string]string{
		"type":     "Battery\n",
		"capacity": "88\n",
		"status":   "Discharging\n",
	} {
		if err := os.WriteFile(filepath.Join(bat, name), []byte(value), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	battery, err := readBatteryStatus(root)
	if err != nil {
		t.Fatal(err)
	}
	if battery == nil {
		t.Fatal("battery = nil, want status")
	}
	if battery.Percent != 88 {
		t.Fatalf("Percent = %v, want 88", battery.Percent)
	}
	if battery.State != "Discharging" {
		t.Fatalf("State = %q, want Discharging", battery.State)
	}
}

func TestReadBatteryStatusNoBattery(t *testing.T) {
	battery, err := readBatteryStatus(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if battery != nil {
		t.Fatalf("battery = %#v, want nil", battery)
	}
}
