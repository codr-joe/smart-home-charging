package energy_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/smart-charging/api/internal/energy"
)

type mockBroadcaster struct {
	mu   sync.Mutex
	msgs [][]byte
}

func (m *mockBroadcaster) Broadcast(data []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.msgs = append(m.msgs, data)
}

func (m *mockBroadcaster) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.msgs)
}

type mockRepo struct {
	mu       sync.Mutex
	readings []energy.Reading
}

func (m *mockRepo) Save(_ context.Context, r energy.Reading) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.readings = append(m.readings, r)
	return nil
}

func (m *mockRepo) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.readings)
}

type mockNotifier struct {
	mu    sync.Mutex
	calls []struct{ title, message string }
}

func (m *mockNotifier) Notify(_ context.Context, title, message string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, struct{ title, message string }{title, message})
	return nil
}

func (m *mockNotifier) callCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.calls)
}

func (m *mockNotifier) messageAt(i int) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls[i].message
}

// p1Server creates a test HTTP server that returns the given active_power_w value.
func p1Server(t *testing.T, powerW float64) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/data" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"active_power_w": %f, "active_tariff": 1}`, powerW)
	}))
}

func TestPollerPollsP1Meter(t *testing.T) {
	body := `{"active_power_w": -1200.0, "active_tariff": 1}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/data" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(body))
	}))
	defer srv.Close()

	repo := &mockRepo{}
	hub := &mockBroadcaster{}
	poller := energy.NewPollerWithClient(srv.URL, repo, hub, nil, time.Second, srv.Client())

	ctx := context.Background()
	if err := poller.PollOnce(ctx); err != nil {
		t.Fatalf("PollOnce() error: %v", err)
	}
	if repo.count() != 1 {
		t.Errorf("expected 1 saved reading, got %d", repo.count())
	}
	if hub.count() != 1 {
		t.Errorf("expected 1 broadcast, got %d", hub.count())
	}
	if repo.readings[0].PowerW != -1200.0 {
		t.Errorf("expected power_w -1200.0, got %v", repo.readings[0].PowerW)
	}
}

func TestPollerNoNotificationWithNilNotifier(t *testing.T) {
	srv := p1Server(t, -1500)
	defer srv.Close()

	poller := energy.NewPollerWithClient(srv.URL, &mockRepo{}, &mockBroadcaster{}, nil, time.Second, srv.Client())

	// Must not panic when notifier is nil.
	if err := poller.PollOnce(context.Background()); err != nil {
		t.Fatalf("PollOnce() unexpected error: %v", err)
	}
}

func TestPollerNotifiesAtFirstThreshold(t *testing.T) {
	// excess = 1 200 W → band 1 000 W
	srv := p1Server(t, -1200)
	defer srv.Close()

	notifier := &mockNotifier{}
	poller := energy.NewPollerWithClient(srv.URL, &mockRepo{}, &mockBroadcaster{}, notifier, time.Second, srv.Client())

	if err := poller.PollOnce(context.Background()); err != nil {
		t.Fatalf("PollOnce() error: %v", err)
	}
	if notifier.callCount() != 1 {
		t.Fatalf("expected 1 notification, got %d", notifier.callCount())
	}
	if !strings.Contains(notifier.messageAt(0), "1000") {
		t.Errorf("expected message to mention 1000 W, got %q", notifier.messageAt(0))
	}
}

func TestPollerNotifiesForEachBandCrossed(t *testing.T) {
	// excess = 2 100 W → bands 1 000, 1 500, 2 000 all crossed at once
	srv := p1Server(t, -2100)
	defer srv.Close()

	notifier := &mockNotifier{}
	poller := energy.NewPollerWithClient(srv.URL, &mockRepo{}, &mockBroadcaster{}, notifier, time.Second, srv.Client())

	if err := poller.PollOnce(context.Background()); err != nil {
		t.Fatalf("PollOnce() error: %v", err)
	}
	if notifier.callCount() != 3 {
		t.Fatalf("expected 3 notifications (1000, 1500, 2000), got %d", notifier.callCount())
	}
	for i, want := range []string{"1000", "1500", "2000"} {
		if !strings.Contains(notifier.messageAt(i), want) {
			t.Errorf("notification %d: expected message to contain %q W, got %q", i, want, notifier.messageAt(i))
		}
	}
}

func TestPollerNotifiesOnIncrementalBandRise(t *testing.T) {
	// First poll: excess 1 200 W → notify at 1 000 W
	// Second poll: excess 1 700 W → notify at 1 500 W only
	notifier := &mockNotifier{}

	srv1 := p1Server(t, -1200)
	defer srv1.Close()
	poller := energy.NewPollerWithClient(srv1.URL, &mockRepo{}, &mockBroadcaster{}, notifier, time.Second, srv1.Client())
	if err := poller.PollOnce(context.Background()); err != nil {
		t.Fatalf("first PollOnce() error: %v", err)
	}

	srv2 := p1Server(t, -1700)
	defer srv2.Close()
	poller2 := energy.NewPollerWithClient(srv2.URL, &mockRepo{}, &mockBroadcaster{}, notifier, time.Second, srv2.Client())
	// Reuse the same notifier but a fresh poller so lastBand starts at 0.
	// To test incremental rises, drive the same poller instance: use a mutable server.
	_ = poller2

	// Use a server whose response changes between calls.
	step := 0
	bodies := []float64{-1200, -1700}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/data" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"active_power_w": %f, "active_tariff": 1}`, bodies[step])
		step++
	}))
	defer srv.Close()

	n2 := &mockNotifier{}
	p := energy.NewPollerWithClient(srv.URL, &mockRepo{}, &mockBroadcaster{}, n2, time.Second, srv.Client())

	if err := p.PollOnce(context.Background()); err != nil {
		t.Fatalf("first PollOnce() error: %v", err)
	}
	if err := p.PollOnce(context.Background()); err != nil {
		t.Fatalf("second PollOnce() error: %v", err)
	}

	if n2.callCount() != 2 {
		t.Fatalf("expected 2 notifications (1000 W then 1500 W), got %d", n2.callCount())
	}
	if !strings.Contains(n2.messageAt(0), "1000") {
		t.Errorf("first notification should mention 1000 W, got %q", n2.messageAt(0))
	}
	if !strings.Contains(n2.messageAt(1), "1500") {
		t.Errorf("second notification should mention 1500 W, got %q", n2.messageAt(1))
	}
}

func TestPollerNotifiesOnFallingBelowThreshold(t *testing.T) {
	// First poll: excess 1 200 W → notify at 1 000 W (rising)
	// Second poll: excess 300 W → notify "dropped below 500 W" (falling)
	step := 0
	bodies := []float64{-1200, -300}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/data" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"active_power_w": %f, "active_tariff": 1}`, bodies[step])
		step++
	}))
	defer srv.Close()

	notifier := &mockNotifier{}
	poller := energy.NewPollerWithClient(srv.URL, &mockRepo{}, &mockBroadcaster{}, notifier, time.Second, srv.Client())

	if err := poller.PollOnce(context.Background()); err != nil {
		t.Fatalf("first PollOnce() error: %v", err)
	}
	if err := poller.PollOnce(context.Background()); err != nil {
		t.Fatalf("second PollOnce() error: %v", err)
	}

	if notifier.callCount() != 2 {
		t.Fatalf("expected 2 notifications, got %d", notifier.callCount())
	}
	if !strings.Contains(notifier.messageAt(1), "500") {
		t.Errorf("falling notification should mention 500 W, got %q", notifier.messageAt(1))
	}
}

func TestPollerNoNotificationWhenExcessBelowThreshold(t *testing.T) {
	// excess = 600 W → below 1 000 W minimum, no notification
	srv := p1Server(t, -600)
	defer srv.Close()

	notifier := &mockNotifier{}
	poller := energy.NewPollerWithClient(srv.URL, &mockRepo{}, &mockBroadcaster{}, notifier, time.Second, srv.Client())

	if err := poller.PollOnce(context.Background()); err != nil {
		t.Fatalf("PollOnce() error: %v", err)
	}
	if notifier.callCount() != 0 {
		t.Errorf("expected 0 notifications for excess below 1000 W, got %d", notifier.callCount())
	}
}

func TestPollerNoFallingNotificationWhenNeverRose(t *testing.T) {
	// excess = 200 W without ever having risen above 1 000 W → no notification
	srv := p1Server(t, -200)
	defer srv.Close()

	notifier := &mockNotifier{}
	poller := energy.NewPollerWithClient(srv.URL, &mockRepo{}, &mockBroadcaster{}, notifier, time.Second, srv.Client())

	if err := poller.PollOnce(context.Background()); err != nil {
		t.Fatalf("PollOnce() error: %v", err)
	}
	if notifier.callCount() != 0 {
		t.Errorf("expected 0 notifications when lastBand=0 and excess<500, got %d", notifier.callCount())
	}
}
