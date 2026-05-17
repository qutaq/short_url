package repository

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"sync"
)

type urlRecord struct {
	UUID        string `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
	UserID      string `json:"user_id,omitempty"`
}

type FileRepository struct {
	mu       sync.RWMutex
	data     map[string]string
	reverse  map[string]string   // original_url → short_id
	userURLs map[string][]string // user_id → []short_id
	deleted  map[string]bool     // short_id → is_deleted
	owners   map[string]string   // short_id → user_id
	filePath string
	nextUUID int
}

func NewFileRepository(filePath string) (*FileRepository, error) {
	r := &FileRepository{
		data:     make(map[string]string),
		reverse:  make(map[string]string),
		userURLs: make(map[string][]string),
		deleted:  make(map[string]bool),
		owners:   make(map[string]string),
		filePath: filePath,
		nextUUID: 1,
	}
	if err := r.load(); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *FileRepository) Save(ctx context.Context, id, url, userID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := validateUserID(userID); err != nil {
		return err
	}

	if _, exists := r.reverse[url]; exists {
		return ErrURLExists
	}
	if _, exists := r.data[id]; exists {
		return ErrConflict
	}

	rec := urlRecord{
		UUID:        strconv.Itoa(r.nextUUID),
		ShortURL:    id,
		OriginalURL: url,
		UserID:      userID,
	}
	if err := r.appendRecord(rec); err != nil {
		return err
	}
	r.data[id] = url
	r.reverse[url] = id
	r.owners[id] = userID
	if userID != "" {
		r.userURLs[userID] = append(r.userURLs[userID], id)
	}
	r.nextUUID++
	return nil
}

func (r *FileRepository) SaveBatch(ctx context.Context, entries []BatchEntry) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, e := range entries {
		if err := validateUserID(e.UserID); err != nil {
			return err
		}
		if _, exists := r.reverse[e.URL]; exists {
			return ErrURLExists
		}
		if _, exists := r.data[e.ID]; exists {
			return ErrConflict
		}
	}
	for _, e := range entries {
		r.data[e.ID] = e.URL
		r.reverse[e.URL] = e.ID
		r.owners[e.ID] = e.UserID
		if e.UserID != "" {
			r.userURLs[e.UserID] = append(r.userURLs[e.UserID], e.ID)
		}
		r.nextUUID++
	}
	return r.flush()
}

func (r *FileRepository) Get(ctx context.Context, id string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.deleted[id] {
		return "", ErrDeleted
	}
	url, ok := r.data[id]
	if !ok {
		return "", ErrNotFound
	}
	return url, nil
}

func (r *FileRepository) DeleteUserURLs(ctx context.Context, shortIDs []string, userID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, id := range shortIDs {
		if owner, ok := r.owners[id]; ok && owner == userID {
			r.deleted[id] = true
		}
	}
	return nil
}

func (r *FileRepository) GetByOriginalURL(ctx context.Context, url string) (string, bool, error) {
	if err := ctx.Err(); err != nil {
		return "", false, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	id, ok := r.reverse[url]
	return id, ok, nil
}

func (r *FileRepository) GetURLsByUser(ctx context.Context, userID string) ([]URLPair, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	ids := r.userURLs[userID]
	if len(ids) == 0 {
		return nil, nil
	}
	pairs := make([]URLPair, 0, len(ids))
	for _, id := range ids {
		if url, ok := r.data[id]; ok {
			pairs = append(pairs, URLPair{ShortID: id, OriginalURL: url})
		}
	}
	return pairs, nil
}

func (r *FileRepository) load() error {
	raw, err := os.ReadFile(r.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if len(raw) == 0 {
		return nil
	}

	scanner := bufio.NewScanner(strings.NewReader(string(raw)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var rec urlRecord
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			return err
		}
		r.applyRecord(rec)
	}
	return scanner.Err()
}

func (r *FileRepository) applyRecord(rec urlRecord) {
	r.data[rec.ShortURL] = rec.OriginalURL
	r.reverse[rec.OriginalURL] = rec.ShortURL
	if rec.UserID != "" {
		r.userURLs[rec.UserID] = append(r.userURLs[rec.UserID], rec.ShortURL)
	}
	if n, err := strconv.Atoi(rec.UUID); err == nil && n >= r.nextUUID {
		r.nextUUID = n + 1
	}
}

func (r *FileRepository) appendRecord(rec urlRecord) error {
	f, err := os.OpenFile(r.filePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	line, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	_, err = f.Write(append(line, '\n'))
	return err
}

func (r *FileRepository) flush() error {
	f, err := os.Create(r.filePath)
	if err != nil {
		return err
	}
	defer f.Close()
	i := 1
	for shortURL, originalURL := range r.data {
		rec := urlRecord{
			UUID:        strconv.Itoa(i),
			ShortURL:    shortURL,
			OriginalURL: originalURL,
		}
		line, err := json.Marshal(rec)
		if err != nil {
			return err
		}
		if _, err := f.Write(append(line, '\n')); err != nil {
			return err
		}
		i++
	}
	return nil
}
