package repository

import (
	"encoding/json"
	"os"
	"strconv"
	"sync"
)

type urlRecord struct {
	UUID        string `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

type FileRepository struct {
	mu       sync.RWMutex
	data     map[string]string
	filePath string
	nextUUID int
}

func NewFileRepository(filePath string) (*FileRepository, error) {
	r := &FileRepository{
		data:     make(map[string]string),
		filePath: filePath,
		nextUUID: 1,
	}
	if err := r.load(); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *FileRepository) Save(id, url string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.data[id]; exists {
		return ErrConflict
	}
	r.data[id] = url
	r.nextUUID++

	return r.flush()
}

func (r *FileRepository) Get(id string) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	url, ok := r.data[id]
	return url, ok
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

	var records []urlRecord
	if err := json.Unmarshal(raw, &records); err != nil {
		return err
	}

	for _, rec := range records {
		r.data[rec.ShortURL] = rec.OriginalURL
		if n, err := strconv.Atoi(rec.UUID); err == nil && n >= r.nextUUID {
			r.nextUUID = n + 1
		}
	}
	return nil
}

func (r *FileRepository) flush() error {
	records := make([]urlRecord, 0, len(r.data))
	i := 1
	for shortURL, originalURL := range r.data {
		records = append(records, urlRecord{
			UUID:        strconv.Itoa(i),
			ShortURL:    shortURL,
			OriginalURL: originalURL,
		})
		i++
	}

	raw, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(r.filePath, raw, 0644)
}
