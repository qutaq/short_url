package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/qutaq/short_url/internal/config"
	"github.com/qutaq/short_url/internal/handler"
	"github.com/qutaq/short_url/internal/repository"
	"github.com/qutaq/short_url/internal/service"
)

func main() {
	cfg := config.Load()

	r := chi.NewRouter()
	repo := repository.NewMemoryRepository()
	shortener := service.NewShortener(repo, cfg.BaseURL)
	h := handler.NewShortenerHandler(shortener)

	r.Post("/", h.PostShorten)
	r.Get("/{id}", h.GetRedirect)

	log.Println("Server starting at", cfg.ServerAddr)
	if err := http.ListenAndServe(cfg.ServerAddr, r); err != nil {
		log.Fatal(err)
	}
}
