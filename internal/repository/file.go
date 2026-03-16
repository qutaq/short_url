package repository

import (
	"bufio"
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

	rec := urlRecord{
		UUID:        strconv.Itoa(r.nextUUID),
		ShortURL:    id,
		OriginalURL: url,
	}
	if err := r.appendRecord(rec); err != nil {
		return err
	}
	r.data[id] = url
	r.nextUUID++
	return nil
}

func (r *FileRepository) SaveBatch(entries []BatchEntry) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, e := range entries {
		if _, exists := r.data[e.ID]; exists {
			return ErrConflict
		}
	}
	for _, e := range entries {
		r.data[e.ID] = e.URL
		r.nextUUID++
	}
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
