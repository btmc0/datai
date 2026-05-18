package hostactions

import (
	"context"
	"testing"
)

func TestDisplaySleepStatusShape(t *testing.T) {
	status := GetDisplaySleepStatus(context.Background())
	if status.Status == "" {
		t.Fatalf("status = %#v, want non-empty status", status)
	}
	if status.Platform == "" {
		t.Fatalf("status = %#v, want platform", status)
	}
	if status.State == "" {
		t.Fatalf("status = %#v, want state", status)
	}
	if !status.Available && status.Reason == "" {
		t.Fatalf("status = %#v, unavailable status should include reason", status)
	}
}
