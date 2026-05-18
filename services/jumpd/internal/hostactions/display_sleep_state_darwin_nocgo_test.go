//go:build darwin && !cgo

package hostactions

import "testing"

func TestParseDisplaySleepState(t *testing.T) {
	tests := []struct {
		name string
		out  string
		want string
	}{
		{
			name: "awake description",
			out:  "Driver ID  Current State  Max State  Current State Description\nIODisplayWrangler  4  4  Display is on",
			want: "awake",
		},
		{
			name: "asleep description",
			out:  "Driver ID  Current State  Max State  Current State Description\nIODisplayWrangler  0  4  Display is off",
			want: "asleep",
		},
		{
			name: "internal failure",
			out:  "Driver ID Current State Max State Current State Description\nInternal failure: Failed to get power state information",
			want: "unknown",
		},
		{
			name: "unrelated sleep text",
			out:  "Display Sleep Timer = 10\nIODisplayWrangler 4 4 available",
			want: "unknown",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseDisplaySleepState(tt.out); got != tt.want {
				t.Fatalf("parseDisplaySleepState() = %q, want %q", got, tt.want)
			}
		})
	}
}
