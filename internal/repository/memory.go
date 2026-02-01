package repository

import "sync"

type MemoryRepository struct {
	mu   sync.RWMutex
	data map[string]string
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		data: make(map[string]string),
	}
}

func (r *MemoryRepository) Save(id, url string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.data[id]; exists {
		return ErrConflict
	}
	r.data[id] = url
	return nil
}

func (r *MemoryRepository) Get(id string) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	url, ok := r.data[id]
	return url, ok
}
