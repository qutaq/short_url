package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/qutaq/short_url/internal/audit"
	"github.com/qutaq/short_url/internal/auth"
	"github.com/qutaq/short_url/internal/middleware"
	"github.com/qutaq/short_url/internal/service"
)

type shortenRequest struct {
	URL string `json:"url"`
}

type shortenResponse struct {
	Result string `json:"result"`
}

// ShortenerHandler предоставляет HTTP-обработчики для сокращения, раскрытия,
// просмотра и удаления URL.
type ShortenerHandler struct {
	facade *ShortenerFacade
}

// NewShortenerHandler создаёт ShortenerHandler на основе shortener.
// Первый необязательный auditor получает события успешного сокращения и перехода.
func NewShortenerHandler(shortener *service.Shortener, auditors ...audit.Observer) *ShortenerHandler {
	return &ShortenerHandler{facade: NewShortenerFacade(shortener, auditors...)}
}

// Facade возвращает общий фасад бизнес-логики для HTTP- и gRPC-обработчиков.
func (h *ShortenerHandler) Facade() *ShortenerFacade {
	return h.facade
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

// PostShorten обрабатывает запросы POST / с исходным URL в текстовом теле.
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

	result, err := h.facade.ShortenURL(r.Context(), rawURL)
	if err != nil {
		code, msg := statusFromError(err)
		http.Error(w, msg, code)
		return
	}
	if result.Conflict {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte(result.ShortURL))
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write([]byte(result.ShortURL))
}

// GetRedirect обрабатывает запросы GET /{id} и перенаправляет на исходный URL.
func (h *ShortenerHandler) GetRedirect(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	originalURL, err := h.facade.ExpandURL(r.Context(), id)
	if err != nil {
		code, msg := statusFromError(err)
		http.Error(w, msg, code)
		return
	}
	w.Header().Set("Location", originalURL)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

// PostShortenJSON обрабатывает запросы POST /api/shorten с URL в JSON-теле.
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

	result, err := h.facade.ShortenURL(r.Context(), rawURL)
	if err != nil {
		code, msg := statusFromError(err)
		http.Error(w, msg, code)
		return
	}
	if result.Conflict {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(shortenResponse{Result: result.ShortURL})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(shortenResponse{Result: result.ShortURL})
}

// PostShortenBatch обрабатывает запросы POST /api/shorten/batch с несколькими URL.
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

	results, err := h.facade.Shortener().ShortenBatch(r.Context(), items, userID)
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

// GetUserURLs обрабатывает запросы GET /api/user/urls и возвращает URL,
// принадлежащие аутентифицированному пользователю.
func (h *ShortenerHandler) GetUserURLs(w http.ResponseWriter, r *http.Request) {
	if _, ok := auth.UserIDFromContext(r.Context()); !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	urls, err := h.facade.ListUserURLs(r.Context())
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

// DeleteUserURLs обрабатывает запросы DELETE /api/user/urls и планирует удаление
// URL, принадлежащих аутентифицированному пользователю.
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

	h.facade.Shortener().DeleteUserURLs(ids, userID)
	w.WriteHeader(http.StatusAccepted)
}

type internalStatsResponse struct {
	URLs  int `json:"urls"`
	Users int `json:"users"`
}

// GetInternalStats возвращает обработчик GET /api/internal/stats с проверкой доверенной подсети.
func GetInternalStats(shortener *service.Shortener, checker *middleware.TrustedSubnetChecker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !checker.Allowed(r) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		stats, err := shortener.GetStats(r.Context())
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(internalStatsResponse{
			URLs:  stats.URLs,
			Users: stats.Users,
		})
	}
}

// PingDB возвращает обработчик, проверяющий подключение к PostgreSQL.
func PingDB(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if pool == nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), time.Second)
		defer cancel()
		if err := pool.Ping(ctx); err != nil {
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
