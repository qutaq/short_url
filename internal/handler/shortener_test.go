package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/qutaq/short_url/internal/repository"
	"github.com/qutaq/short_url/internal/service"
)

const testBaseURL = "http://localhost:8080"

func setupHandler(t *testing.T) (http.Handler, *repository.MemoryRepository) {
	t.Helper()
	repo := repository.NewMemoryRepository()
	svc := service.NewShortener(repo, testBaseURL)
	h := NewShortenerHandler(svc)
	r := chi.NewRouter()
	r.Post("/", h.PostShorten)
	r.Post("/api/shorten", h.PostShortenJSON)
	r.Get("/{id}", h.GetRedirect)
	return r, repo
}

func TestPostShorten_MethodNotAllowed(t *testing.T) {
	r, _ := setupHandler(t)

	req := httptest.NewRequest(http.MethodGet, "http://test/", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("PostShorten GET: status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestPostShorten_EmptyBody(t *testing.T) {
	r, _ := setupHandler(t)

	req := httptest.NewRequest(http.MethodPost, "http://test/", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("PostShorten empty body: status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestPostShorten_InvalidURL(t *testing.T) {
	r, _ := setupHandler(t)

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
			r.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("PostShorten %q: status = %d, want %d", rawURL, rec.Code, http.StatusBadRequest)
			}
		})
	}
}

func TestPostShorten_ValidURL(t *testing.T) {
	r, _ := setupHandler(t)

	req := httptest.NewRequest(http.MethodPost, "http://test/", bytes.NewBufferString("https://example.com/page"))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

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
	r, _ := setupHandler(t)

	req := httptest.NewRequest(http.MethodPost, "http://test/", bytes.NewBufferString("  https://example.com  "))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("PostShorten trimmed URL: status = %d, want %d", rec.Code, http.StatusCreated)
	}
}

func TestGetRedirect_RootPath(t *testing.T) {
	r, _ := setupHandler(t)

	req := httptest.NewRequest(http.MethodGet, "http://test/", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("GetRedirect root path: status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestGetRedirect_UnknownID(t *testing.T) {
	r, _ := setupHandler(t)

	req := httptest.NewRequest(http.MethodGet, "http://test/unknown123", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("GetRedirect unknown id: status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestGetRedirect_KnownID(t *testing.T) {
	r, repo := setupHandler(t)

	originalURL := "https://example.com/original"
	id := "abc12345"
	if err := repo.Save(id, originalURL, ""); err != nil {
		t.Fatalf("repo.Save: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "http://test/"+id, nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusTemporaryRedirect {
		t.Errorf("GetRedirect known id: status = %d, want %d", rec.Code, http.StatusTemporaryRedirect)
	}
	if loc := rec.Header().Get("Location"); loc != originalURL {
		t.Errorf("GetRedirect Location = %q, want %q", loc, originalURL)
	}
}

func TestPostShortenJSON_ValidURL(t *testing.T) {
	r, _ := setupHandler(t)

	body, _ := json.Marshal(map[string]string{"url": "https://practicum.yandex.ru"})
	req := httptest.NewRequest(http.MethodPost, "http://test/api/shorten", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("PostShortenJSON valid URL: status = %d, want %d", rec.Code, http.StatusCreated)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("PostShortenJSON Content-Type = %q, want application/json", ct)
	}

	var resp shortenResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("PostShortenJSON: failed to decode response: %v", err)
	}
	if resp.Result == "" || !strings.HasPrefix(resp.Result, testBaseURL+"/") {
		t.Errorf("PostShortenJSON result = %q, want prefix %q", resp.Result, testBaseURL+"/")
	}
}

func TestPostShortenJSON_EmptyURL(t *testing.T) {
	r, _ := setupHandler(t)

	body, _ := json.Marshal(map[string]string{"url": ""})
	req := httptest.NewRequest(http.MethodPost, "http://test/api/shorten", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("PostShortenJSON empty URL: status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestPostShortenJSON_InvalidURL(t *testing.T) {
	r, _ := setupHandler(t)

	tests := []string{"not-a-url", "://host", "http://", "no-scheme.com"}
	for _, rawURL := range tests {
		t.Run(rawURL, func(t *testing.T) {
			body, _ := json.Marshal(map[string]string{"url": rawURL})
			req := httptest.NewRequest(http.MethodPost, "http://test/api/shorten", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("PostShortenJSON %q: status = %d, want %d", rawURL, rec.Code, http.StatusBadRequest)
			}
		})
	}
}

func TestPostShortenJSON_InvalidJSON(t *testing.T) {
	r, _ := setupHandler(t)

	req := httptest.NewRequest(http.MethodPost, "http://test/api/shorten", bytes.NewBufferString("{invalid"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("PostShortenJSON invalid JSON: status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestPostShortenJSON_MissingURLField(t *testing.T) {
	r, _ := setupHandler(t)

	body, _ := json.Marshal(map[string]string{"link": "https://example.com"})
	req := httptest.NewRequest(http.MethodPost, "http://test/api/shorten", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("PostShortenJSON missing url field: status = %d, want %d", rec.Code, http.StatusBadRequest)
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
