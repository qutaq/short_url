package service

import (
	"crypto/rand"
	"errors"
	"fmt"

	"github.com/qutaq/short_url/internal/repository"
)

const (
	shortIDLen     = 8
	maxSaveRetries = 5
)

var (
	ErrNotFound     = errors.New("url not found")
	ErrInvalidInput = errors.New("invalid input")
	ErrURLConflict  = errors.New("original url already exists")
)

type Shortener struct {
	repo    repository.URLRepository
	baseURL string
}
type BatchInput struct {
	CorrelationID string
	OriginalURL   string
}

type BatchOutput struct {
	CorrelationID string
	ShortURL      string
}

func NewShortener(repo repository.URLRepository, baseURL string) *Shortener {
	return &Shortener{repo: repo, baseURL: baseURL}
}

func (s *Shortener) Shorten(url string) (string, error) {
	if url == "" {
		return "", ErrInvalidInput
	}
	id, err := s.generateUniqueID()
	if err != nil {
		return "", err
	}
	for range maxSaveRetries {
		err = s.repo.Save(id, url)
		if err == nil {
			return s.baseURL + "/" + id, nil
		}
		if errors.Is(err, repository.ErrURLExists) {
			return s.findExistingShortURL(url)
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

func (s *Shortener) findExistingShortURL(url string) (string, error) {
	existingID, ok := s.repo.GetByOriginalURL(url)
	if !ok {
		return "", fmt.Errorf("original url conflict but record not found")
	}
	return s.baseURL + "/" + existingID, ErrURLConflict
}

func (s *Shortener) ShortenBatch(items []BatchInput) ([]BatchOutput, error) {
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
		entries[i] = repository.BatchEntry{ID: id, URL: item.OriginalURL}
		results[i] = BatchOutput{
			CorrelationID: item.CorrelationID,
			ShortURL:      s.baseURL + "/" + id,
		}
	}

	for attempt := range maxSaveRetries {
		err := s.repo.SaveBatch(entries)
		if err == nil {
			return results, nil
		}
		if errors.Is(err, repository.ErrURLExists) {
			return s.shortenBatchOneByOne(entries, results)
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
			entries[i] = repository.BatchEntry{ID: id, URL: item.OriginalURL}
			results[i].ShortURL = s.baseURL + "/" + id
		}
	}
	return nil, fmt.Errorf("failed to save batch after %d retries: %w", maxSaveRetries, repository.ErrConflict)
}

func (s *Shortener) shortenBatchOneByOne(entries []repository.BatchEntry, results []BatchOutput) ([]BatchOutput, error) {
	for i := range entries {
		saved := false
		for attempt := 0; attempt < maxSaveRetries && !saved; attempt++ {
			err := s.repo.Save(entries[i].ID, entries[i].URL)
			if err == nil {
				results[i].ShortURL = s.baseURL + "/" + entries[i].ID
				saved = true
				break
			}
			if errors.Is(err, repository.ErrURLExists) {
				existingID, ok := s.repo.GetByOriginalURL(entries[i].URL)
				if !ok {
					return nil, fmt.Errorf("original url exists but record not found")
				}
				results[i].ShortURL = s.baseURL + "/" + existingID
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

func (s *Shortener) GetOriginal(id string) (string, error) {
	url, ok := s.repo.Get(id)
	if !ok {
		return "", ErrNotFound
	}
	return url, nil
}

func (s *Shortener) generateUniqueID() (string, error) {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, shortIDLen)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate id: %w", err)
	}
	for i := range b {
		b[i] = charset[int(b[i])%len(charset)]
	}
	return string(b), nil
}
