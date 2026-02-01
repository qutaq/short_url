package main

import (
	"log"
	"net/http"

	"github.com/qutaq/short_url/internal/handler"
	"github.com/qutaq/short_url/internal/repository"
	"github.com/qutaq/short_url/internal/service"
)

const baseURL = "http://localhost:8080"

func main() {
	repo := repository.NewMemoryRepository()
	shortener := service.NewShortener(repo, baseURL)
	h := handler.NewShortenerHandler(shortener)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /", h.PostShorten)
	mux.HandleFunc("GET /{id}", h.GetRedirect)

	log.Println("Server starting at", baseURL)
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
