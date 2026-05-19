package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"go.uber.org/zap"

	"github.com/qutaq/short_url/internal/audit"
	"github.com/qutaq/short_url/internal/config"
)

func TestNewRouterAppliesMiddleware(t *testing.T) {
	router := newRouter(zap.NewNop())
	router.Get("/ok", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if cookies := rec.Result().Cookies(); len(cookies) != 1 {
		t.Fatalf("cookies len = %d, want auth cookie from middleware", len(cookies))
	}
}

func TestNewAuditNotifier(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.jsonl")
	notifier, auditObservers, err := newAuditNotifier(&config.Config{AuditFile: path})
	if err != nil {
		t.Fatalf("newAuditNotifier file: %v", err)
	}
	if notifier == nil {
		t.Fatal("newAuditNotifier file returned nil notifier")
	}
	if auditObservers.file == nil {
		t.Fatal("newAuditNotifier file did not configure file observer")
	}
	if err := notifier.Notify(context.Background(), audit.Event{Timestamp: 1, Action: audit.ActionShorten, URL: "https://example.com"}); err != nil {
		t.Fatalf("Notify file observer: %v", err)
	}
	auditObservers.closeFileObserver()

	notifier, auditObservers, err = newAuditNotifier(&config.Config{})
	if err != nil {
		t.Fatalf("newAuditNotifier empty: %v", err)
	}
	if notifier != nil {
		t.Fatal("newAuditNotifier empty returned notifier, want nil")
	}
	auditObservers.closeFileObserver()

	if _, _, err := newAuditNotifier(&config.Config{AuditURL: "://bad-url"}); err == nil {
		t.Fatal("newAuditNotifier invalid remote URL error = nil, want error")
	}
}
