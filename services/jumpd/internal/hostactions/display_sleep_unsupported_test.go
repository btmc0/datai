//go:build !darwin

package hostactions

import (
	"context"
	"errors"
	"testing"
)

func TestDisplaySleepUnsupported(t *testing.T) {
	status := GetDisplaySleepStatus(context.Background())
	if status.Available {
		t.Fatal("display sleep should not be available on non-darwin platforms")
	}
	if status.Status != "unsupported" {
		t.Fatalf("status = %q, want unsupported", status.Status)
	}
	if status.Platform == "" || status.Reason == "" {
		t.Fatalf("status missing platform/reason: %#v", status)
	}

	got, err := SleepDisplay(context.Background())
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("SleepDisplay error = %v, want ErrUnavailable", err)
	}
	if got.Available {
		t.Fatalf("SleepDisplay capability = %#v, want unavailable", got)
	}
}
