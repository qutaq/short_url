package handler

import (
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/qutaq/short_url/internal/service"
)

type ShortenerHandler struct {
	shortener *service.Shortener
}

func NewShortenerHandler(shortener *service.Shortener) *ShortenerHandler {
	return &ShortenerHandler{shortener: shortener}
}

func (h *ShortenerHandler) PostShorten(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Bad Request", http.StatusBadRequest)
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
	shortURL, err := h.shortener.Shorten(rawURL)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
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
	originalURL, err := h.shortener.GetOriginal(id)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	w.Header().Set("Location", originalURL)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

func isValidURL(s string) bool {
	u, err := url.Parse(s)
	if err != nil {
		return false
	}
	return u.Scheme != "" && u.Host != ""
}
