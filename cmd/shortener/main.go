package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/zap"

	"github.com/qutaq/short_url/internal/audit"
	"github.com/qutaq/short_url/internal/config"
	"github.com/qutaq/short_url/internal/handler"
	"github.com/qutaq/short_url/internal/middleware"
	"github.com/qutaq/short_url/internal/repository"
	"github.com/qutaq/short_url/internal/service"
	dbmigrations "github.com/qutaq/short_url/migrations"
)

var buildVersion = "N/A"
var buildDate = "N/A"
var buildCommit = "N/A"

func main() {
	printBuildInfo()

	cfg := config.Load()

	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal(err)
	}
	defer logger.Sync()

	r := newRouter(logger)

	var pool *pgxpool.Pool
	if cfg.DatabaseDSN != "" {
		var err error
		pool, err = pgxpool.New(context.Background(), cfg.DatabaseDSN)
		if err != nil {
			log.Fatal("failed to open database pool:", err)
		}
		defer pool.Close()

		sqlDB := stdlib.OpenDBFromPool(pool)
		defer sqlDB.Close()
		if err := runMigrations(sqlDB); err != nil {
			log.Fatal("failed to run migrations:", err)
		}
	}

	var repo repository.URLRepository
	switch {
	case cfg.DatabaseDSN != "" && pool != nil:
		repo = repository.NewPostgresRepository(pool)
	case cfg.FileStoragePath != "":
		fileRepo, err := repository.NewFileRepository(cfg.FileStoragePath)
		if err != nil {
			log.Fatal("failed to open file storage:", err)
		}
		repo = fileRepo
	default:
		repo = repository.NewMemoryRepository()
	}

	auditor, auditObservers, err := newAuditNotifier(cfg)
	if err != nil {
		log.Fatal("failed to configure audit:", err)
	}
	defer auditObservers.closeFileObserver()

	shortener := service.NewShortener(repo, cfg.BaseURL)
	h := handler.NewShortenerHandler(shortener, auditor)

	r.Get("/ping", handler.PingDB(pool))
	r.Post("/", h.PostShorten)
	r.Post("/api/shorten", h.PostShortenJSON)
	r.Post("/api/shorten/batch", h.PostShortenBatch)
	r.Get("/api/user/urls", h.GetUserURLs)
	r.Delete("/api/user/urls", h.DeleteUserURLs)
	r.Get("/{id}", h.GetRedirect)

	log.Printf("Server starting at %s", cfg.ServerAddr)

	srv := &http.Server{
		Addr:    cfg.ServerAddr,
		Handler: r,
	}

	go func() {
		if err := listenAndServe(srv, cfg.EnableHTTPS); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server shutdown: %v", err)
	}
}

func printBuildInfo() {
	fmt.Printf("Build version: %s\n", buildVersion)
	fmt.Printf("Build date: %s\n", buildDate)
	fmt.Printf("Build commit: %s\n", buildCommit)
}

func newRouter(logger *zap.Logger) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.RequestLogger(logger))
	r.Use(middleware.GzipMiddleware)
	r.Use(middleware.AuthMiddleware)
	return r
}

type auditObservers struct {
	file *audit.FileObserver
}

func (o *auditObservers) closeFileObserver() {
	if o == nil || o.file == nil {
		return
	}
	if err := o.file.Close(); err != nil {
		log.Printf("failed to close audit file observer: %v", err)
	}
}

func newAuditNotifier(cfg *config.Config) (*audit.Notifier, *auditObservers, error) {
	var observers []audit.Observer
	auditObservers := &auditObservers{}

	if cfg.AuditFile != "" {
		fileObserver, err := audit.NewFileObserver(cfg.AuditFile)
		if err != nil {
			return nil, nil, err
		}
		observers = append(observers, fileObserver)
		auditObservers.file = fileObserver
	}

	if cfg.AuditURL != "" {
		remoteObserver, err := audit.NewRemoteObserver(cfg.AuditURL)
		if err != nil {
			return nil, nil, err
		}
		observers = append(observers, remoteObserver)
	}

	return audit.NewNotifier(observers...), auditObservers, nil
}

func runMigrations(db *sql.DB) error {
	source, err := iofs.New(dbmigrations.FS, ".")
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
	defer func() {
		sourceErr, databaseErr := m.Close()
		if sourceErr != nil {
			log.Printf("failed to close migration source: %v", sourceErr)
		}
		if databaseErr != nil {
			log.Printf("failed to close migration database: %v", databaseErr)
		}
	}()
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}
