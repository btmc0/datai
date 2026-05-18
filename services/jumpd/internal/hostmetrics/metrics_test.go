package hostmetrics

import "testing"

func TestClampPercent(t *testing.T) {
	cases := []struct {
		name string
		in   float64
		want float64
	}{
		{name: "low", in: -4, want: 0},
		{name: "in range", in: 72.4, want: 72.4},
		{name: "high", in: 104, want: 100},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := clampPercent(tc.in); got != tc.want {
				t.Fatalf("clampPercent(%v) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestNormalizeBatteryState(t *testing.T) {
	if got := normalizeBatteryState(" Not_Charging \n"); got != "not charging" {
		t.Fatalf("normalizeBatteryState = %q, want %q", got, "not charging")
	}
}
