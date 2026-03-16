package main

import (
	"database/sql"
	"embed"
	"errors"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/zap"

	"github.com/qutaq/short_url/internal/config"
	"github.com/qutaq/short_url/internal/handler"
	"github.com/qutaq/short_url/internal/middleware"
	"github.com/qutaq/short_url/internal/repository"
	"github.com/qutaq/short_url/internal/service"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func main() {
	cfg := config.Load()

	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal(err)
	}
	defer logger.Sync()

	r := chi.NewRouter()
	r.Use(middleware.RequestLogger(logger))
	r.Use(middleware.GzipMiddleware)

	var db *sql.DB
	if cfg.DatabaseDSN != "" {
		var err error
		db, err = sql.Open("pgx", cfg.DatabaseDSN)
		if err != nil {
			log.Fatal("failed to open database:", err)
		}
		defer db.Close()

		if err := runMigrations(db); err != nil {
			log.Fatal("failed to run migrations:", err)
		}
	}

	var repo repository.URLRepository
	switch {
	case cfg.DatabaseDSN != "" && db != nil:
		repo = repository.NewPostgresRepository(db)
	case cfg.FileStoragePath != "":
		fileRepo, err := repository.NewFileRepository(cfg.FileStoragePath)
		if err != nil {
			log.Fatal("failed to open file storage:", err)
		}
		repo = fileRepo
	default:
		repo = repository.NewMemoryRepository()
	}

	shortener := service.NewShortener(repo, cfg.BaseURL)
	h := handler.NewShortenerHandler(shortener)

	r.Get("/ping", handler.PingDB(db))
	r.Post("/", h.PostShorten)
	r.Post("/api/shorten", h.PostShortenJSON)
	r.Post("/api/shorten/batch", h.PostShortenBatch)
	r.Get("/{id}", h.GetRedirect)

	log.Println("Server starting at", cfg.ServerAddr)
	if err := http.ListenAndServe(cfg.ServerAddr, r); err != nil {
		log.Fatal(err)
	}
}

func runMigrations(db *sql.DB) error {
	source, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return err
	}
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return err
	}
	m, err := migrate.NewWithInstance("iofs", source, "postgres", driver)
	if err != nil {
		return err
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}
