package energy

import "time"

// Reading represents a single snapshot of energy data from the P1 meter.
type Reading struct {
	Time   time.Time `json:"time"`
	PowerW float64   `json:"power_w"`
	SolarW *float64  `json:"solar_w,omitempty"`
	Tariff string    `json:"tariff,omitempty"`
}

// ExcessW returns the estimated excess solar power available for EV charging.
// A negative PowerW means the house is injecting into the grid.
func (r Reading) ExcessW() float64 {
	if r.PowerW < 0 {
		return -r.PowerW
	}
	return 0
}
