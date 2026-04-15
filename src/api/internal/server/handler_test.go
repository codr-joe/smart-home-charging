package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/smart-charging/api/internal/energy"
	"github.com/smart-charging/api/internal/server"
)

// mockNotifier records Notify calls and returns a configurable error.
type mockNotifier struct {
	calls []struct{ title, message string }
	err   error
}

func (m *mockNotifier) Notify(_ context.Context, title, message string) error {
	m.calls = append(m.calls, struct{ title, message string }{title, message})
	return m.err
}

// newTestApp creates a Fiber app for handler testing.
// If notifier is non-nil it is wrapped in an enabled TogglableNotifier.
func newTestApp(notifier energy.Notifier) *fiber.App {
	var toggle *energy.TogglableNotifier
	if notifier != nil {
		toggle = energy.NewTogglableNotifier(notifier, true)
	}
	return server.New(nil, nil, toggle)
}

func doRequest(t *testing.T, app *fiber.App, method, path string, body io.Reader) *http.Response {
	t.Helper()
	req := httptest.NewRequest(method, path, body)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test() error: %v", err)
	}
	return resp
}

func doJSONRequest(t *testing.T, app *fiber.App, method, path string, payload any) *http.Response {
	t.Helper()
	b, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test() error: %v", err)
	}
	return resp
}

func TestTestNotification_Success(t *testing.T) {
	notifier := &mockNotifier{}
	app := newTestApp(notifier)

	resp := doRequest(t, app, http.MethodPost, "/api/v1/notifications/test", nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if len(notifier.calls) != 1 {
		t.Fatalf("expected 1 Notify call, got %d", len(notifier.calls))
	}
	if notifier.calls[0].message != "Hello World" {
		t.Errorf("message = %q, want %q", notifier.calls[0].message, "Hello World")
	}
	if notifier.calls[0].title != "Smart Charging" {
		t.Errorf("title = %q, want %q", notifier.calls[0].title, "Smart Charging")
	}
}

func TestTestNotification_NoNotifier(t *testing.T) {
	app := newTestApp(nil)

	resp := doRequest(t, app, http.MethodPost, "/api/v1/notifications/test", nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusServiceUnavailable)
	}
}

func TestTestNotification_NotifierError(t *testing.T) {
	notifier := &mockNotifier{err: &fiber.Error{Code: 500, Message: "pushover down"}}
	app := newTestApp(notifier)

	resp := doRequest(t, app, http.MethodPost, "/api/v1/notifications/test", nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusInternalServerError)
	}
}

func TestTestNotification_ResponseBody(t *testing.T) {
	notifier := &mockNotifier{}
	app := newTestApp(notifier)

	resp := doRequest(t, app, http.MethodPost, "/api/v1/notifications/test", nil)
	defer resp.Body.Close()

	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf(`body["status"] = %q, want "ok"`, body["status"])
	}
}

func TestTestNotification_DisabledReturns503(t *testing.T) {
	toggle := energy.NewTogglableNotifier(&mockNotifier{}, false)
	app := server.New(nil, nil, toggle)

	resp := doRequest(t, app, http.MethodPost, "/api/v1/notifications/test", nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusServiceUnavailable)
	}
}

func TestGetNotificationSettings_Enabled(t *testing.T) {
	toggle := energy.NewTogglableNotifier(&mockNotifier{}, true)
	app := server.New(nil, nil, toggle)

	resp := doRequest(t, app, http.MethodGet, "/api/v1/notifications/settings", nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if body["enabled"] != true {
		t.Errorf(`body["enabled"] = %v, want true`, body["enabled"])
	}
}

func TestGetNotificationSettings_Disabled(t *testing.T) {
	toggle := energy.NewTogglableNotifier(&mockNotifier{}, false)
	app := server.New(nil, nil, toggle)

	resp := doRequest(t, app, http.MethodGet, "/api/v1/notifications/settings", nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if body["enabled"] != false {
		t.Errorf(`body["enabled"] = %v, want false`, body["enabled"])
	}
}

func TestGetNotificationSettings_NoNotifier(t *testing.T) {
	app := server.New(nil, nil, nil)

	resp := doRequest(t, app, http.MethodGet, "/api/v1/notifications/settings", nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusServiceUnavailable)
	}
}

func TestUpdateNotificationSettings_Disable(t *testing.T) {
	toggle := energy.NewTogglableNotifier(&mockNotifier{}, true)
	app := server.New(nil, nil, toggle)

	resp := doJSONRequest(t, app, http.MethodPut, "/api/v1/notifications/settings", map[string]bool{"enabled": false})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if body["enabled"] != false {
		t.Errorf(`body["enabled"] = %v, want false`, body["enabled"])
	}
	if toggle.IsEnabled() {
		t.Error("toggle.IsEnabled() = true after disabling, want false")
	}
}

func TestUpdateNotificationSettings_Enable(t *testing.T) {
	toggle := energy.NewTogglableNotifier(&mockNotifier{}, false)
	app := server.New(nil, nil, toggle)

	resp := doJSONRequest(t, app, http.MethodPut, "/api/v1/notifications/settings", map[string]bool{"enabled": true})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if !toggle.IsEnabled() {
		t.Error("toggle.IsEnabled() = false after enabling, want true")
	}
}

func TestUpdateNotificationSettings_NoNotifier(t *testing.T) {
	app := server.New(nil, nil, nil)

	resp := doJSONRequest(t, app, http.MethodPut, "/api/v1/notifications/settings", map[string]bool{"enabled": true})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusServiceUnavailable)
	}
}
