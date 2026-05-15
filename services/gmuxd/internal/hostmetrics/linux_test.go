//go:build linux

package hostmetrics

import "testing"

func TestRound1(t *testing.T) {
	if got := round1(12.34); got != 12.3 {
		t.Fatalf("round1 = %v, want 12.3", got)
	}
}
