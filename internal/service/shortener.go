package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	"github.com/qutaq/short_url/internal/repository"
)

const (
	shortIDLen     = 8
	maxSaveRetries = 5
)

var (
	// ErrNotFound сообщает, что идентификатор короткой ссылки не найден.
	ErrNotFound = errors.New("url not found")
	// ErrInvalidInput сообщает, что в сервис переданы некорректные данные.
	ErrInvalidInput = errors.New("invalid input")
	// ErrURLConflict сообщает, что исходный URL уже был сокращён.
	ErrURLConflict = errors.New("original url already exists")
	// ErrDeleted сообщает, что короткая ссылка была удалена владельцем.
	ErrDeleted = errors.New("url has been deleted")
)

type deleteTask struct {
	shortID string
	userID  string
}

// Shortener координирует операции сокращения URL через репозиторий.
type Shortener struct {
	repo           repository.URLRepository
	shortURLPrefix string
	delCh          chan deleteTask
}

// BatchInput описывает один URL в пакетном запросе на сокращение.
type BatchInput struct {
	// CorrelationID идентифицирует элемент в клиентском запросе.
	CorrelationID string
	// OriginalURL содержит URL, который нужно сократить.
	OriginalURL string
}

// BatchOutput описывает один сокращённый URL в пакетном ответе.
type BatchOutput struct {
	// CorrelationID повторяет идентификатор соответствующего элемента запроса.
	CorrelationID string
	// ShortURL содержит сгенерированный абсолютный короткий URL.
	ShortURL string
}

// UserURLOutput описывает короткую ссылку, принадлежащую пользователю.
type UserURLOutput struct {
	// ShortURL содержит абсолютный короткий URL.
	ShortURL string
	// OriginalURL содержит исходный URL, связанный с ShortURL.
	OriginalURL string
}

// NewShortener создаёт Shortener, который строит ссылки с базовым URL baseURL
// и сохраняет данные через repo.
func NewShortener(repo repository.URLRepository, baseURL string) *Shortener {
	s := &Shortener{
		repo:           repo,
		shortURLPrefix: baseURL + "/",
		delCh:          make(chan deleteTask, 1024),
	}
	go s.runDeleteWorker()
	return s
}

// Shorten сохраняет url для userID и возвращает короткую ссылку.
// Если исходный URL уже существует, возвращает имеющуюся короткую ссылку и
// ErrURLConflict.
func (s *Shortener) Shorten(ctx context.Context, url, userID string) (string, error) {
	if url == "" {
		return "", ErrInvalidInput
	}
	id, err := s.generateUniqueID()
	if err != nil {
		return "", err
	}
	for range maxSaveRetries {
		err = s.repo.Save(ctx, id, url, userID)
		if err == nil {
			return s.makeShortURL(id), nil
		}
		if errors.Is(err, repository.ErrURLExists) {
			return s.findExistingShortURL(ctx, url)
		}
		if !errors.Is(err, repository.ErrConflict) {
			return "", err
		}
		id, err = s.generateUniqueID()
		if err != nil {
			return "", err
		}
	}
	return "", fmt.Errorf("failed to save after %d retries: %w", maxSaveRetries, repository.ErrConflict)
}

func (s *Shortener) findExistingShortURL(ctx context.Context, url string) (string, error) {
	existingID, ok, err := s.repo.GetByOriginalURL(ctx, url)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("original url conflict but record not found")
	}
	return s.makeShortURL(existingID), ErrURLConflict
}

// ShortenBatch сохраняет несколько URL для userID и возвращает их короткие
// представления, сохраняя порядок входных данных.
func (s *Shortener) ShortenBatch(ctx context.Context, items []BatchInput, userID string) ([]BatchOutput, error) {
	if len(items) == 0 {
		return nil, ErrInvalidInput
	}
	for _, item := range items {
		if item.OriginalURL == "" {
			return nil, ErrInvalidInput
		}
	}

	entries := make([]repository.BatchEntry, len(items))
	results := make([]BatchOutput, len(items))

	for i, item := range items {
		id, err := s.generateUniqueID()
		if err != nil {
			return nil, err
		}
		entries[i] = repository.BatchEntry{ID: id, URL: item.OriginalURL, UserID: userID}
		results[i] = BatchOutput{
			CorrelationID: item.CorrelationID,
			ShortURL:      s.makeShortURL(id),
		}
	}

	for attempt := range maxSaveRetries {
		err := s.repo.SaveBatch(ctx, entries)
		if err == nil {
			return results, nil
		}
		if errors.Is(err, repository.ErrURLExists) {
			return s.shortenBatchOneByOne(ctx, entries, results)
		}
		if !errors.Is(err, repository.ErrConflict) {
			return nil, err
		}
		if attempt == maxSaveRetries-1 {
			break
		}
		for i, item := range items {
			id, err := s.generateUniqueID()
			if err != nil {
				return nil, err
			}
			entries[i] = repository.BatchEntry{ID: id, URL: item.OriginalURL, UserID: userID}
			results[i].ShortURL = s.makeShortURL(id)
		}
	}
	return nil, fmt.Errorf("failed to save batch after %d retries: %w", maxSaveRetries, repository.ErrConflict)
}

func (s *Shortener) shortenBatchOneByOne(ctx context.Context, entries []repository.BatchEntry, results []BatchOutput) ([]BatchOutput, error) {
	for i := range entries {
		saved := false
		for attempt := 0; attempt < maxSaveRetries && !saved; attempt++ {
			err := s.repo.Save(ctx, entries[i].ID, entries[i].URL, entries[i].UserID)
			if err == nil {
				results[i].ShortURL = s.makeShortURL(entries[i].ID)
				saved = true
				break
			}
			if errors.Is(err, repository.ErrURLExists) {
				existingID, ok, errLookup := s.repo.GetByOriginalURL(ctx, entries[i].URL)
				if errLookup != nil {
					return nil, errLookup
				}
				if !ok {
					return nil, fmt.Errorf("original url exists but record not found")
				}
				results[i].ShortURL = s.makeShortURL(existingID)
				saved = true
				break
			}
			if errors.Is(err, repository.ErrConflict) {
				id, idErr := s.generateUniqueID()
				if idErr != nil {
					return nil, idErr
				}
				entries[i].ID = id
				continue
			}
			return nil, err
		}
		if !saved {
			return nil, fmt.Errorf("failed to save batch item after %d retries: %w", maxSaveRetries, repository.ErrConflict)
		}
	}
	return results, nil
}

// GetOriginal возвращает исходный URL по идентификатору короткой ссылки.
func (s *Shortener) GetOriginal(ctx context.Context, id string) (string, error) {
	url, err := s.repo.Get(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrDeleted) {
			return "", ErrDeleted
		}
		return "", ErrNotFound
	}
	return url, nil
}

// DeleteUserURLs асинхронно помечает короткие ссылки пользователя как удалённые.
func (s *Shortener) DeleteUserURLs(shortIDs []string, userID string) {
	inputCh := generateDeleteTasks(shortIDs, userID)
	go func() {
		for task := range inputCh {
			s.delCh <- task
		}
	}()
}

func generateDeleteTasks(shortIDs []string, userID string) <-chan deleteTask {
	ch := make(chan deleteTask)
	go func() {
		defer close(ch)
		for _, id := range shortIDs {
			ch <- deleteTask{shortID: id, userID: userID}
		}
	}()
	return ch
}

func (s *Shortener) runDeleteWorker() {
	const (
		batchSize    = 100
		flushTimeout = 5 * time.Second
	)
	ticker := time.NewTicker(flushTimeout)
	defer ticker.Stop()

	buf := make([]deleteTask, 0, batchSize)

	for {
		select {
		case task := <-s.delCh:
			buf = append(buf, task)
			if len(buf) >= batchSize {
				s.flushDeleteBatch(buf)
				buf = make([]deleteTask, 0, batchSize)
			}
		case <-ticker.C:
			if len(buf) > 0 {
				s.flushDeleteBatch(buf)
				buf = make([]deleteTask, 0, batchSize)
			}
		}
	}
}

func (s *Shortener) flushDeleteBatch(tasks []deleteTask) {
	byUser := make(map[string][]string)
	for _, t := range tasks {
		byUser[t.userID] = append(byUser[t.userID], t.shortID)
	}
	for userID, ids := range byUser {
		_ = s.repo.DeleteUserURLs(context.Background(), ids, userID)
	}
}

// GetUserURLs возвращает все URL, созданные userID, с полными короткими ссылками.
func (s *Shortener) GetUserURLs(ctx context.Context, userID string) ([]UserURLOutput, error) {
	pairs, err := s.repo.GetURLsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(pairs) == 0 {
		return nil, nil
	}
	result := make([]UserURLOutput, len(pairs))
	for i, p := range pairs {
		result[i] = UserURLOutput{
			ShortURL:    s.makeShortURL(p.ShortID),
			OriginalURL: p.OriginalURL,
		}
	}
	return result, nil
}

func (s *Shortener) makeShortURL(id string) string {
	return s.shortURLPrefix + id
}

func (s *Shortener) generateUniqueID() (string, error) {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	var b [shortIDLen]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generate id: %w", err)
	}
	for i := range b {
		b[i] = charset[int(b[i])%len(charset)]
	}
	return string(b[:]), nil
}
