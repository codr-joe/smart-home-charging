package energy_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/smart-charging/api/internal/energy"
)

func TestPushoverNotifier_Notify_Success(t *testing.T) {
	var gotToken, gotUser, gotTitle, gotMessage string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		gotToken = r.FormValue("token")
		gotUser = r.FormValue("user")
		gotTitle = r.FormValue("title")
		gotMessage = r.FormValue("message")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := energy.PushoverConfig{APIToken: "tok123", UserKey: "usr456"}
	n := energy.NewPushoverNotifierWithClient(cfg, srv.Client(), srv.URL)

	if err := n.Notify(context.Background(), "Test Title", "Test Message"); err != nil {
		t.Fatalf("Notify() unexpected error: %v", err)
	}
	if gotToken != "tok123" {
		t.Errorf("token = %q, want %q", gotToken, "tok123")
	}
	if gotUser != "usr456" {
		t.Errorf("user = %q, want %q", gotUser, "usr456")
	}
	if gotTitle != "Test Title" {
		t.Errorf("title = %q, want %q", gotTitle, "Test Title")
	}
	if gotMessage != "Test Message" {
		t.Errorf("message = %q, want %q", gotMessage, "Test Message")
	}
}

func TestPushoverNotifier_Notify_ContentType(t *testing.T) {
	var gotContentType string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContentType = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := energy.PushoverConfig{APIToken: "tok", UserKey: "usr"}
	n := energy.NewPushoverNotifierWithClient(cfg, srv.Client(), srv.URL)

	if err := n.Notify(context.Background(), "t", "m"); err != nil {
		t.Fatalf("Notify() unexpected error: %v", err)
	}
	if !strings.HasPrefix(gotContentType, "application/x-www-form-urlencoded") {
		t.Errorf("Content-Type = %q, want application/x-www-form-urlencoded", gotContentType)
	}
}

func TestPushoverNotifier_Notify_NonOKStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	cfg := energy.PushoverConfig{APIToken: "tok", UserKey: "usr"}
	n := energy.NewPushoverNotifierWithClient(cfg, srv.Client(), srv.URL)

	err := n.Notify(context.Background(), "t", "m")
	if err == nil {
		t.Fatal("Notify() expected error for non-200 response, got nil")
	}
}

func TestPushoverNotifier_Notify_FormValues(t *testing.T) {
	var gotForm url.Values

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		gotForm = r.Form
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := energy.PushoverConfig{APIToken: "mytoken", UserKey: "myuser"}
	n := energy.NewPushoverNotifierWithClient(cfg, srv.Client(), srv.URL)

	if err := n.Notify(context.Background(), "Alert", "Power exceeded 1000 W"); err != nil {
		t.Fatalf("Notify() unexpected error: %v", err)
	}
	for _, key := range []string{"token", "user", "title", "message"} {
		if gotForm.Get(key) == "" {
			t.Errorf("form field %q is empty", key)
		}
	}
}
