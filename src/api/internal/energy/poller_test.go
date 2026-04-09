package energy_test

import (
	"context"
	"net/http"
	"net/http/httptest"
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
	poller := energy.NewPollerWithClient(srv.URL, repo, hub, time.Second, srv.Client())

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
