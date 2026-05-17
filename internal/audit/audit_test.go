package audit

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewEvent(t *testing.T) {
	event := NewEvent(ActionShorten, "user1", "https://example.com")

	if event.Timestamp == 0 {
		t.Fatal("NewEvent timestamp = 0, want unix timestamp")
	}
	if event.Action != ActionShorten || event.UserID != "user1" || event.URL != "https://example.com" {
		t.Fatalf("NewEvent = %#v, want provided fields", event)
	}
}

func TestNotifierJoinsObserverErrors(t *testing.T) {
	errOne := errors.New("one")
	errTwo := errors.New("two")
	notifier := NewNotifier(
		observerFunc(func(context.Context, Event) error { return errOne }),
		observerFunc(func(context.Context, Event) error { return errTwo }),
	)

	err := notifier.Notify(context.Background(), Event{})
	if !errors.Is(err, errOne) || !errors.Is(err, errTwo) {
		t.Fatalf("Notify error = %v, want both observer errors", err)
	}

	var nilNotifier *Notifier
	if err := nilNotifier.Notify(context.Background(), Event{}); err != nil {
		t.Fatalf("nil notifier Notify error = %v, want nil", err)
	}
}

func TestFileObserverWritesEvents(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.jsonl")
	observer, err := NewFileObserver(path)
	if err != nil {
		t.Fatalf("NewFileObserver: %v", err)
	}
	defer observer.Close()

	event := Event{Timestamp: 1, Action: ActionFollow, UserID: "user1", URL: "https://example.com"}
	if err := observer.Notify(context.Background(), event); err != nil {
		t.Fatalf("Notify: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var got Event
	if err := json.Unmarshal([]byte(strings.TrimSpace(string(raw))), &got); err != nil {
		t.Fatalf("Unmarshal event: %v", err)
	}
	if got != event {
		t.Fatalf("stored event = %#v, want %#v", got, event)
	}

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if err := observer.Notify(cancelled, event); !errors.Is(err, context.Canceled) {
		t.Fatalf("Notify cancelled error = %v, want %v", err, context.Canceled)
	}
}

func TestRemoteObserver(t *testing.T) {
	var got Event
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Fatalf("Content-Type = %q, want application/json", ct)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("Decode body: %v", err)
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	observer, err := NewRemoteObserver(server.URL)
	if err != nil {
		t.Fatalf("NewRemoteObserver: %v", err)
	}
	event := Event{Timestamp: 1, Action: ActionShorten, URL: "https://example.com"}
	if err := observer.Notify(context.Background(), event); err != nil {
		t.Fatalf("Notify: %v", err)
	}
	if got != event {
		t.Fatalf("remote event = %#v, want %#v", got, event)
	}
}

func TestRemoteObserverErrors(t *testing.T) {
	if _, err := NewRemoteObserver("://bad-url"); err == nil {
		t.Fatal("NewRemoteObserver invalid URL error = nil, want error")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad", http.StatusBadGateway)
	}))
	defer server.Close()

	observer, err := NewRemoteObserver(server.URL)
	if err != nil {
		t.Fatalf("NewRemoteObserver: %v", err)
	}
	if err := observer.Notify(context.Background(), Event{}); err == nil {
		t.Fatal("Notify bad status error = nil, want error")
	}
}

type observerFunc func(context.Context, Event) error

func (f observerFunc) Notify(ctx context.Context, event Event) error {
	return f(ctx, event)
}
