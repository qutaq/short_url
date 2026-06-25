package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository хранит записи URL в PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository создаёт репозиторий URL на основе PostgreSQL.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// Save сохраняет одну запись короткой ссылки в PostgreSQL.
func (r *PostgresRepository) Save(ctx context.Context, id, url, userID string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := validateUserID(userID); err != nil {
		return err
	}

	tag, err := r.pool.Exec(ctx,
		`INSERT INTO urls (short_id, original_url, user_id) VALUES ($1, $2, $3)
		 ON CONFLICT (original_url) DO NOTHING`, id, url, userID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return ErrConflict
		}
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrURLExists
	}
	return nil
}

// SaveBatch сохраняет несколько записей коротких ссылок в одной транзакции.
func (r *PostgresRepository) SaveBatch(ctx context.Context, entries []BatchEntry) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	for _, e := range entries {
		if err := validateUserID(e.UserID); err != nil {
			return err
		}
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for _, e := range entries {
		tag, err := tx.Exec(ctx,
			`INSERT INTO urls (short_id, original_url, user_id) VALUES ($1, $2, $3)
			 ON CONFLICT (original_url) DO NOTHING`, e.ID, e.URL, e.UserID)
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
				return ErrConflict
			}
			return err
		}
		if tag.RowsAffected() == 0 {
			return ErrURLExists
		}
	}
	return tx.Commit(ctx)
}

// Get возвращает исходный URL по идентификатору короткой ссылки.
func (r *PostgresRepository) Get(ctx context.Context, id string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var originalURL string
	var isDeleted bool
	err := r.pool.QueryRow(ctx,
		`SELECT original_url, is_deleted FROM urls WHERE short_id = $1`, id).Scan(&originalURL, &isDeleted)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", err
	}
	if isDeleted {
		return "", ErrDeleted
	}
	return originalURL, nil
}

// DeleteUserURLs помечает URL, принадлежащие userID, как удалённые.
func (r *PostgresRepository) DeleteUserURLs(ctx context.Context, shortIDs []string, userID string) error {
	if len(shortIDs) == 0 {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	placeholders := make([]string, len(shortIDs))
	args := make([]interface{}, 0, len(shortIDs)+1)
	for i, id := range shortIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args = append(args, id)
	}
	args = append(args, userID)

	query := fmt.Sprintf(
		`UPDATE urls SET is_deleted = TRUE WHERE short_id IN (%s) AND user_id = $%d`,
		strings.Join(placeholders, ", "),
		len(shortIDs)+1,
	)
	_, err := r.pool.Exec(ctx, query, args...)
	return err
}

// GetByOriginalURL ищет идентификатор короткой ссылки по исходному URL.
func (r *PostgresRepository) GetByOriginalURL(ctx context.Context, url string) (string, bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var shortID string
	err := r.pool.QueryRow(ctx,
		`SELECT short_id FROM urls WHERE original_url = $1`, url).Scan(&shortID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", false, nil
		}
		return "", false, err
	}
	return shortID, true, nil
}

// GetURLsByUser возвращает записи URL, созданные userID.
func (r *PostgresRepository) GetURLsByUser(ctx context.Context, userID string) ([]URLPair, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rows, err := r.pool.Query(ctx,
		`SELECT short_id, original_url FROM urls WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pairs []URLPair
	for rows.Next() {
		var p URLPair
		if err := rows.Scan(&p.ShortID, &p.OriginalURL); err != nil {
			return nil, err
		}
		pairs = append(pairs, p)
	}
	return pairs, rows.Err()
}

// GetStats возвращает количество сокращённых URL и пользователей в PostgreSQL.
func (r *PostgresRepository) GetStats(ctx context.Context) (Stats, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var stats Stats
	err := r.pool.QueryRow(ctx, `
		SELECT
			COUNT(*),
			COUNT(DISTINCT user_id) FILTER (WHERE user_id <> '')
		FROM urls`).Scan(&stats.URLs, &stats.Users)
	if err != nil {
		return Stats{}, err
	}
	return stats, nil
}
