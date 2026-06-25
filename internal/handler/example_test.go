package handler_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/qutaq/short_url/internal/handler"
	"github.com/qutaq/short_url/internal/middleware"
	"github.com/qutaq/short_url/internal/repository"
	"github.com/qutaq/short_url/internal/service"
)

const exampleBaseURL = "http://localhost:8080"

func setupExampleRouter() (*chi.Mux, *repository.MemoryRepository) {
	repo := repository.NewMemoryRepository()
	svc := service.NewShortener(repo, exampleBaseURL)
	h := handler.NewShortenerHandler(svc, nil)

	r := chi.NewRouter()
	r.Use(middleware.AuthMiddleware)
	r.Post("/", h.PostShorten)
	r.Post("/api/shorten", h.PostShortenJSON)
	r.Post("/api/shorten/batch", h.PostShortenBatch)
	r.Get("/api/user/urls", h.GetUserURLs)
	r.Delete("/api/user/urls", h.DeleteUserURLs)
	r.Get("/{id}", h.GetRedirect)

	return r, repo
}

func ExampleShortenerHandler_PostShorten() {
	r, _ := setupExampleRouter()

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("https://practicum.yandex.ru"))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	shortURL := strings.TrimSpace(rec.Body.String())
	fmt.Println("status:", rec.Code)
	fmt.Println("is short URL:", strings.HasPrefix(shortURL, exampleBaseURL+"/"))
	fmt.Println("short ID length:", len(strings.TrimPrefix(shortURL, exampleBaseURL+"/")))

	// Output:
	// status: 201
	// is short URL: true
	// short ID length: 8
}

func ExampleShortenerHandler_GetRedirect() {
	r, repo := setupExampleRouter()
	if err := repo.Save(httptest.NewRequest(http.MethodGet, "/", nil).Context(), "abc12345", "https://example.com/page", ""); err != nil {
		panic(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/abc12345", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	fmt.Println("status:", rec.Code)
	fmt.Println("location:", rec.Header().Get("Location"))

	// Output:
	// status: 307
	// location: https://example.com/page
}

func ExampleShortenerHandler_PostShortenJSON() {
	r, _ := setupExampleRouter()

	body := bytes.NewBufferString(`{"url":"https://practicum.yandex.ru/go-advanced"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/shorten", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	var resp struct {
		Result string `json:"result"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		panic(err)
	}

	fmt.Println("status:", rec.Code)
	fmt.Println("is short URL:", strings.HasPrefix(resp.Result, exampleBaseURL+"/"))

	// Output:
	// status: 201
	// is short URL: true
}

func ExampleShortenerHandler_PostShortenBatch() {
	r, _ := setupExampleRouter()

	body := bytes.NewBufferString(`[
		{"correlation_id":"first","original_url":"https://example.com/first"},
		{"correlation_id":"second","original_url":"https://example.com/second"}
	]`)
	req := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	var resp []struct {
		CorrelationID string `json:"correlation_id"`
		ShortURL      string `json:"short_url"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		panic(err)
	}

	fmt.Println("status:", rec.Code)
	fmt.Println("items:", len(resp))
	fmt.Println("correlations:", resp[0].CorrelationID, resp[1].CorrelationID)
	fmt.Println("short URLs:", strings.HasPrefix(resp[0].ShortURL, exampleBaseURL+"/"), strings.HasPrefix(resp[1].ShortURL, exampleBaseURL+"/"))

	// Output:
	// status: 201
	// items: 2
	// correlations: first second
	// short URLs: true true
}

func ExampleShortenerHandler_GetUserURLs() {
	r, _ := setupExampleRouter()

	createReq := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("https://example.com/user-link"))
	createRec := httptest.NewRecorder()
	r.ServeHTTP(createRec, createReq)

	req := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)
	req.AddCookie(createRec.Result().Cookies()[0])
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	var resp []struct {
		ShortURL    string `json:"short_url"`
		OriginalURL string `json:"original_url"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		panic(err)
	}

	fmt.Println("status:", rec.Code)
	fmt.Println("items:", len(resp))
	fmt.Println("original URL:", resp[0].OriginalURL)
	fmt.Println("is short URL:", strings.HasPrefix(resp[0].ShortURL, exampleBaseURL+"/"))

	// Output:
	// status: 200
	// items: 1
	// original URL: https://example.com/user-link
	// is short URL: true
}

func ExampleShortenerHandler_DeleteUserURLs() {
	r, _ := setupExampleRouter()

	createReq := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("https://example.com/to-delete"))
	createRec := httptest.NewRecorder()
	r.ServeHTTP(createRec, createReq)
	shortID := strings.TrimPrefix(strings.TrimSpace(createRec.Body.String()), exampleBaseURL+"/")

	body, _ := json.Marshal([]string{shortID})
	req := httptest.NewRequest(http.MethodDelete, "/api/user/urls", bytes.NewReader(body))
	req.AddCookie(createRec.Result().Cookies()[0])
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	fmt.Println("status:", rec.Code)

	// Output:
	// status: 202
}
