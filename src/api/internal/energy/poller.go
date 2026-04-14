package energy

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// p1Response models the relevant fields from the HomeWizard P1 meter local API.
type p1Response struct {
	ActivePowerW float64 `json:"active_power_w"`
	ActiveTariff int     `json:"active_tariff"`
}

// Broadcaster is implemented by the WebSocket hub to fan-out live readings.
type Broadcaster interface {
	Broadcast(data []byte)
}

// Saver is implemented by the energy repository.
type Saver interface {
	Save(ctx context.Context, r Reading) error
}

// Poller periodically fetches data from the HomeWizard P1 meter HTTP API,
// persists it to the database, and broadcasts it to connected WebSocket clients.
// When a Notifier is provided it sends Pushover alerts on significant excess-power changes.
type Poller struct {
	p1URL    string
	repo     Saver
	hub      Broadcaster
	notifier Notifier
	interval time.Duration
	client   *http.Client
	lastBand int // last excess-power band for which a rising notification was sent
}

// NewPoller creates a Poller using the default HTTP client.
// Pass nil for notifier to disable Pushover alerts.
func NewPoller(p1URL string, repo Saver, hub Broadcaster, notifier Notifier, interval time.Duration) *Poller {
	return &Poller{
		p1URL:    p1URL,
		repo:     repo,
		hub:      hub,
		notifier: notifier,
		interval: interval,
		client:   &http.Client{Timeout: 5 * time.Second},
	}
}

// NewPollerWithClient creates a Poller with a custom HTTP client (useful for testing).
// Pass nil for notifier to disable Pushover alerts.
func NewPollerWithClient(p1URL string, repo Saver, hub Broadcaster, notifier Notifier, interval time.Duration, client *http.Client) *Poller {
	return &Poller{
		p1URL:    p1URL,
		repo:     repo,
		hub:      hub,
		notifier: notifier,
		interval: interval,
		client:   client,
	}
}

// Run starts the polling loop. It blocks until ctx is cancelled.
func (p *Poller) Run(ctx context.Context) {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := p.PollOnce(ctx); err != nil {
				log.Printf("p1 poll error: %v", err)
			}
		}
	}
}

// PollOnce performs a single fetch-save-broadcast cycle.
func (p *Poller) PollOnce(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.p1URL+"/api/v1/data", nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("fetch p1 data: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d from P1 meter", resp.StatusCode)
	}
	var payload p1Response
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return fmt.Errorf("decode p1 response: %w", err)
	}
	reading := Reading{
		Time:   time.Now().UTC(),
		PowerW: payload.ActivePowerW,
		Tariff: fmt.Sprintf("T%d", payload.ActiveTariff),
	}
	if err := p.repo.Save(ctx, reading); err != nil {
		return fmt.Errorf("save reading: %w", err)
	}
	msg, err := json.Marshal(reading)
	if err != nil {
		return fmt.Errorf("marshal reading: %w", err)
	}
	p.hub.Broadcast(msg)
	if p.notifier != nil {
		p.handleNotifications(ctx, reading.ExcessW())
	}
	return nil
}

// handleNotifications evaluates the current excess power against the last notified band
// and sends Pushover alerts when significant thresholds are crossed.
//
// Rising alerts fire at 1 000 W and every 500 W increment above that.
// A falling alert fires when excess drops below 500 W after previously being above 1 000 W.
func (p *Poller) handleNotifications(ctx context.Context, excess float64) {
	currentBand := excessBand(excess)

	switch {
	case currentBand > p.lastBand:
		// Notify for every newly crossed 500 W band.
		start := p.lastBand + 500
		if start < 1000 {
			start = 1000
		}
		for band := start; band <= currentBand; band += 500 {
			msg := fmt.Sprintf("Excess solar power has reached %d W.", band)
			if err := p.notifier.Notify(ctx, "Solar Charging Alert", msg); err != nil {
				log.Printf("pushover notify error: %v", err)
			}
		}
		p.lastBand = currentBand

	case excess < 500 && p.lastBand > 0:
		// Notify once when excess falls well below the charging threshold.
		if err := p.notifier.Notify(ctx, "Solar Charging Alert", "Excess solar power has dropped below 500 W."); err != nil {
			log.Printf("pushover notify error: %v", err)
		}
		p.lastBand = 0
	}
}
