package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) Save(id, url string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := r.db.ExecContext(ctx,
		`INSERT INTO urls (short_id, original_url) VALUES ($1, $2)
		 ON CONFLICT (original_url) DO NOTHING`, id, url)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return ErrConflict
		}
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrURLExists
	}
	return nil
}

func (r *PostgresRepository) SaveBatch(entries []BatchEntry) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO urls (short_id, original_url) VALUES ($1, $2)
		 ON CONFLICT (original_url) DO NOTHING`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, e := range entries {
		result, err := stmt.ExecContext(ctx, e.ID, e.URL)
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
				return ErrConflict
			}
			return err
		}
		rows, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if rows == 0 {
			return ErrURLExists
		}
	}
	return tx.Commit()
}

func (r *PostgresRepository) Get(id string) (string, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var originalURL string
	err := r.db.QueryRowContext(ctx,
		`SELECT original_url FROM urls WHERE short_id = $1`, id).Scan(&originalURL)
	if err != nil {
		return "", false
	}
	return originalURL, true
}

func (r *PostgresRepository) GetByOriginalURL(url string) (string, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var shortID string
	err := r.db.QueryRowContext(ctx,
		`SELECT short_id FROM urls WHERE original_url = $1`, url).Scan(&shortID)
	if err != nil {
		return "", false
	}
	return shortID, true
}
