package repository

import "sync"

type MemoryRepository struct {
	mu      sync.RWMutex
	data    map[string]string
	reverse map[string]string // original_url → short_id
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		data:    make(map[string]string),
		reverse: make(map[string]string),
	}
}

func (r *MemoryRepository) Save(id, url string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.reverse[url]; exists {
		return ErrURLExists
	}
	if _, exists := r.data[id]; exists {
		return ErrConflict
	}
	r.data[id] = url
	r.reverse[url] = id
	return nil
}

func (r *MemoryRepository) SaveBatch(entries []BatchEntry) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, e := range entries {
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
	}
	return nil
}

func (r *MemoryRepository) Get(id string) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	url, ok := r.data[id]
	return url, ok
}

func (r *MemoryRepository) GetByOriginalURL(url string) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	id, ok := r.reverse[url]
	return id, ok
}
