package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/qutaq/short_url/internal/repository"
	"github.com/qutaq/short_url/internal/service"
)

func setupBenchmarkHandler() http.Handler {
	repo := repository.NewMemoryRepository()
	svc := service.NewShortener(repo, testBaseURL)
	h := NewShortenerHandler(svc)

	r := chi.NewRouter()
	r.Post("/", h.PostShorten)
	r.Post("/api/shorten", h.PostShortenJSON)
	r.Get("/{id}", h.GetRedirect)
	return r
}

func BenchmarkPostShorten(b *testing.B) {
	r := setupBenchmarkHandler()

	for i := 0; b.Loop(); i++ {
		b.StopTimer()
		req := httptest.NewRequest(http.MethodPost, "http://test/", bytes.NewBufferString(fmt.Sprintf("https://example.com/%d", i)))
		rec := httptest.NewRecorder()
		b.StartTimer()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			b.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
		}
	}
}

func BenchmarkPostShortenJSON(b *testing.B) {
	r := setupBenchmarkHandler()

	for i := 0; b.Loop(); i++ {
		b.StopTimer()
		body, err := json.Marshal(shortenRequest{URL: fmt.Sprintf("https://example.com/%d", i)})
		if err != nil {
			b.Fatal(err)
		}
		req := httptest.NewRequest(http.MethodPost, "http://test/api/shorten", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		b.StartTimer()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			b.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
		}
	}
}
