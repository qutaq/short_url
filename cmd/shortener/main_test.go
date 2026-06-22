package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
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
	if notifier == nil {
		t.Fatal("newAuditNotifier empty returned nil notifier")
	}
	auditObservers.closeFileObserver()

	if _, _, err := newAuditNotifier(&config.Config{AuditURL: "://bad-url"}); err == nil {
		t.Fatal("newAuditNotifier invalid remote URL error = nil, want error")
	}
}

func TestNewTLSConfig(t *testing.T) {
	tlsConfig, err := newTLSConfig()
	if err != nil {
		t.Fatalf("newTLSConfig: %v", err)
	}
	if tlsConfig.MinVersion != tls.VersionTLS12 {
		t.Fatalf("MinVersion = %d, want %d", tlsConfig.MinVersion, tls.VersionTLS12)
	}
	if len(tlsConfig.Certificates) != 1 {
		t.Fatalf("certificates len = %d, want 1", len(tlsConfig.Certificates))
	}
}

func TestNewSelfSignedCertificate(t *testing.T) {
	cert, err := newSelfSignedCertificate()
	if err != nil {
		t.Fatalf("newSelfSignedCertificate: %v", err)
	}
	if len(cert.Certificate) == 0 {
		t.Fatal("certificate chain is empty")
	}
	if cert.PrivateKey == nil {
		t.Fatal("private key is nil")
	}

	parsed, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		t.Fatalf("ParseCertificate: %v", err)
	}
	if got := parsed.Subject.Organization; len(got) != 1 || got[0] != "Yandex.Praktikum" {
		t.Fatalf("organization = %v, want [Yandex.Praktikum]", got)
	}
	if err := parsed.VerifyHostname("localhost"); err != nil {
		t.Fatalf("VerifyHostname localhost: %v", err)
	}
}

func TestListenAndServeReturnsErrorForInvalidAddr(t *testing.T) {
	tests := []struct {
		name        string
		enableHTTPS bool
	}{
		{name: "http", enableHTTPS: false},
		{name: "https", enableHTTPS: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := &http.Server{
				Addr:    "127.0.0.1:-1",
				Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
			}

			if err := listenAndServe(srv, tt.enableHTTPS, zap.NewNop()); err == nil {
				t.Fatal("listenAndServe error = nil, want error")
			}
		})
	}
}
