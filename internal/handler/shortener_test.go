package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/qutaq/short_url/internal/repository"
	"github.com/qutaq/short_url/internal/service"
)

const testBaseURL = "http://localhost:8080"

func setupHandler(t *testing.T) (http.Handler, *repository.MemoryRepository) {
	t.Helper()
	repo := repository.NewMemoryRepository()
	svc := service.NewShortener(repo, testBaseURL)
	h := NewShortenerHandler(svc)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /", h.PostShorten)
	mux.HandleFunc("GET /{id}", h.GetRedirect)
	return mux, repo
}

func TestPostShorten_MethodNotAllowed(t *testing.T) {
	mux, _ := setupHandler(t)

	req := httptest.NewRequest(http.MethodGet, "http://test/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("PostShorten GET: status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
	if body := rec.Body.String(); body != "Method Not Allowed\n" {
		t.Errorf("PostShorten GET: body = %q, want %q", body, "Method Not Allowed\n")
	}
}

func TestPostShorten_EmptyBody(t *testing.T) {
	mux, _ := setupHandler(t)

	req := httptest.NewRequest(http.MethodPost, "http://test/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("PostShorten empty body: status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestPostShorten_InvalidURL(t *testing.T) {
	mux, _ := setupHandler(t)

	tests := []string{
		"",
		"   ",
		"not-a-url",
		"ftp://",
		"://host",
		"http://",
		"no-scheme.com",
	}
	for _, rawURL := range tests {
		t.Run(strings.ReplaceAll(rawURL, " ", "_"), func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "http://test/", bytes.NewBufferString(rawURL))
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("PostShorten %q: status = %d, want %d", rawURL, rec.Code, http.StatusBadRequest)
			}
		})
	}
}

func TestPostShorten_ValidURL(t *testing.T) {
	mux, _ := setupHandler(t)

	req := httptest.NewRequest(http.MethodPost, "http://test/", bytes.NewBufferString("https://example.com/page"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("PostShorten valid URL: status = %d, want %d", rec.Code, http.StatusCreated)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/plain; charset=utf-8" {
		t.Errorf("PostShorten Content-Type = %q, want %q", ct, "text/plain; charset=utf-8")
	}
	body := strings.TrimSpace(rec.Body.String())
	if body == "" || !strings.HasPrefix(body, testBaseURL+"/") {
		t.Errorf("PostShorten body = %q, want prefix %q", body, testBaseURL+"/")
	}
	if len(body) != len(testBaseURL)+1+8 {
		t.Errorf("PostShorten short ID length: got %d, want 8", len(body)-len(testBaseURL)-1)
	}
}

func TestPostShorten_ValidURLWithWhitespace(t *testing.T) {
	mux, _ := setupHandler(t)

	req := httptest.NewRequest(http.MethodPost, "http://test/", bytes.NewBufferString("  https://example.com  "))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("PostShorten trimmed URL: status = %d, want %d", rec.Code, http.StatusCreated)
	}
}

func TestGetRedirect_RootPath(t *testing.T) {
	mux, _ := setupHandler(t)

	req := httptest.NewRequest(http.MethodGet, "http://test/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("GetRedirect root path: status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestGetRedirect_UnknownID(t *testing.T) {
	mux, _ := setupHandler(t)

	req := httptest.NewRequest(http.MethodGet, "http://test/unknown123", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("GetRedirect unknown id: status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestGetRedirect_KnownID(t *testing.T) {
	mux, repo := setupHandler(t)

	originalURL := "https://example.com/original"
	id := "abc12345"
	if err := repo.Save(id, originalURL); err != nil {
		t.Fatalf("repo.Save: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "http://test/"+id, nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusTemporaryRedirect {
		t.Errorf("GetRedirect known id: status = %d, want %d", rec.Code, http.StatusTemporaryRedirect)
	}
	if loc := rec.Header().Get("Location"); loc != originalURL {
		t.Errorf("GetRedirect Location = %q, want %q", loc, originalURL)
	}
}

func TestIsValidURL(t *testing.T) {
	tests := []struct {
		url   string
		valid bool
	}{
		{"https://example.com", true},
		{"http://localhost:8080", true},
		{"", false},
		{"not-a-url", false},
		{"ftp://host/path", true},
		{"://host", false},
		{"http://", false},
	}
	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got := isValidURL(tt.url)
			if got != tt.valid {
				t.Errorf("isValidURL(%q) = %v, want %v", tt.url, got, tt.valid)
			}
		})
	}
}
