package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/qutaq/short_url/internal/auth"
	"github.com/qutaq/short_url/internal/service"
)

type shortenRequest struct {
	URL string `json:"url"`
}

type shortenResponse struct {
	Result string `json:"result"`
}

type ShortenerHandler struct {
	shortener *service.Shortener
}

func NewShortenerHandler(shortener *service.Shortener) *ShortenerHandler {
	return &ShortenerHandler{shortener: shortener}
}

type batchRequest struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

type batchResponse struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}

type userURLResponse struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

func statusFromError(err error) (code int, body string) {
	switch {
	case errors.Is(err, service.ErrDeleted):
		return http.StatusGone, "Gone"
	case errors.Is(err, service.ErrNotFound):
		return http.StatusNotFound, "Not Found"
	case errors.Is(err, service.ErrInvalidInput):
		return http.StatusBadRequest, "Bad Request"
	default:
		return http.StatusInternalServerError, "Internal Server Error"
	}
}

func (h *ShortenerHandler) PostShorten(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	rawURL := strings.TrimSpace(string(body))
	if rawURL == "" || !isValidURL(rawURL) {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	userID, _ := auth.UserIDFromContext(r.Context())

	shortURL, err := h.shortener.Shorten(r.Context(), rawURL, userID)
	if err != nil {
		if errors.Is(err, service.ErrURLConflict) {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusConflict)
			w.Write([]byte(shortURL))
			return
		}
		code, msg := statusFromError(err)
		http.Error(w, msg, code)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(shortURL))
}

func (h *ShortenerHandler) GetRedirect(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	originalURL, err := h.shortener.GetOriginal(r.Context(), id)
	if err != nil {
		code, msg := statusFromError(err)
		http.Error(w, msg, code)
		return
	}
	w.Header().Set("Location", originalURL)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

func (h *ShortenerHandler) PostShortenJSON(w http.ResponseWriter, r *http.Request) {
	var req shortenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	rawURL := strings.TrimSpace(req.URL)
	if rawURL == "" || !isValidURL(rawURL) {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	userID, _ := auth.UserIDFromContext(r.Context())

	shortURL, err := h.shortener.Shorten(r.Context(), rawURL, userID)
	if err != nil {
		if errors.Is(err, service.ErrURLConflict) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(shortenResponse{Result: shortURL})
			return
		}
		code, msg := statusFromError(err)
		http.Error(w, msg, code)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(shortenResponse{Result: shortURL})
}

func (h *ShortenerHandler) PostShortenBatch(w http.ResponseWriter, r *http.Request) {
	var req []batchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if len(req) == 0 {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	items := make([]service.BatchInput, len(req))
	for i, r := range req {
		u := strings.TrimSpace(r.OriginalURL)
		if u == "" || !isValidURL(u) {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		items[i] = service.BatchInput{
			CorrelationID: r.CorrelationID,
			OriginalURL:   u,
		}
	}

	userID, _ := auth.UserIDFromContext(r.Context())

	results, err := h.shortener.ShortenBatch(r.Context(), items, userID)
	if err != nil {
		code, msg := statusFromError(err)
		http.Error(w, msg, code)
		return
	}

	resp := make([]batchResponse, len(results))
	for i, res := range results {
		resp[i] = batchResponse{
			CorrelationID: res.CorrelationID,
			ShortURL:      res.ShortURL,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func (h *ShortenerHandler) GetUserURLs(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	urls, err := h.shortener.GetUserURLs(r.Context(), userID)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if len(urls) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	resp := make([]userURLResponse, len(urls))
	for i, u := range urls {
		resp[i] = userURLResponse{
			ShortURL:    u.ShortURL,
			OriginalURL: u.OriginalURL,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *ShortenerHandler) DeleteUserURLs(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var ids []string
	if err := json.NewDecoder(r.Body).Decode(&ids); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if len(ids) == 0 {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	h.shortener.DeleteUserURLs(ids, userID)
	w.WriteHeader(http.StatusAccepted)
}

func PingDB(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if db == nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		if err := db.PingContext(r.Context()); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

func isValidURL(s string) bool {
	u, err := url.Parse(s)
	if err != nil {
		return false
	}
	return u.Scheme != "" && u.Host != ""
}
