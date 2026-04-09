package energy_test

import (
	"testing"
	"time"

	"github.com/smart-charging/api/internal/energy"
)

func TestReadingExcessW(t *testing.T) {
	cases := []struct {
		name   string
		powerW float64
		wantW  float64
	}{
		{"injecting into grid", -1500, 1500},
		{"importing from grid", 800, 0},
		{"zero net power", 0, 0},
		{"large injection", -3200.5, 3200.5},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := energy.Reading{Time: time.Now(), PowerW: tc.powerW}
			got := r.ExcessW()
			if got != tc.wantW {
				t.Errorf("ExcessW() = %v, want %v", got, tc.wantW)
			}
		})
	}
}
