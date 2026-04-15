package energy

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"
)

const defaultPushoverURL = "https://api.pushover.net/1/messages.json"

// Notifier sends alert messages to an external notification service.
type Notifier interface {
	Notify(ctx context.Context, title, message string) error
}

// PushoverConfig holds credentials for the Pushover notification service.
type PushoverConfig struct {
	APIToken string
	UserKey  string
}

type pushoverNotifier struct {
	cfg    PushoverConfig
	client *http.Client
	apiURL string
}

// NewPushoverNotifier creates a Notifier that sends alerts via the Pushover API.
func NewPushoverNotifier(cfg PushoverConfig) Notifier {
	return &pushoverNotifier{
		cfg:    cfg,
		client: &http.Client{Timeout: 10 * time.Second},
		apiURL: defaultPushoverURL,
	}
}

// NewPushoverNotifierWithClient creates a Notifier with a custom HTTP client and API URL.
// Intended for testing.
func NewPushoverNotifierWithClient(cfg PushoverConfig, client *http.Client, apiURL string) Notifier {
	return &pushoverNotifier{
		cfg:    cfg,
		client: client,
		apiURL: apiURL,
	}
}

func (n *pushoverNotifier) Notify(ctx context.Context, title, message string) error {
	form := url.Values{}
	form.Set("token", n.cfg.APIToken)
	form.Set("user", n.cfg.UserKey)
	form.Set("title", title)
	form.Set("message", message)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.apiURL, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("build pushover request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("send pushover notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("pushover returned status %d", resp.StatusCode)
	}
	return nil
}

// excessBand returns the notification band for a given excess wattage.
// Returns 0 when excess is below the minimum alert threshold (1 000 W).
// Otherwise returns the largest 500 W multiple that does not exceed excess.
//
//	0 W – 999 W  → 0
//	1 000 W – 1 499 W → 1 000
//	1 500 W – 1 999 W → 1 500
//	…
func excessBand(excess float64) int {
	if excess < 1000 {
		return 0
	}
	return int(excess/500) * 500
}

// TogglableNotifier wraps a Notifier and allows enabling/disabling notifications at runtime.
// Notify is a no-op when disabled.
type TogglableNotifier struct {
	base    Notifier
	enabled atomic.Bool
}

// NewTogglableNotifier creates a TogglableNotifier wrapping base with an initial enabled state.
func NewTogglableNotifier(base Notifier, enabled bool) *TogglableNotifier {
	n := &TogglableNotifier{base: base}
	n.enabled.Store(enabled)
	return n
}

// Notify sends a notification if enabled, and is a no-op otherwise.
func (n *TogglableNotifier) Notify(ctx context.Context, title, message string) error {
	if !n.enabled.Load() {
		return nil
	}
	return n.base.Notify(ctx, title, message)
}

// SetEnabled enables or disables notifications at runtime.
func (n *TogglableNotifier) SetEnabled(enabled bool) {
	n.enabled.Store(enabled)
}

// IsEnabled reports whether notifications are currently enabled.
func (n *TogglableNotifier) IsEnabled() bool {
	return n.enabled.Load()
}
