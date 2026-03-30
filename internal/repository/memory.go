package repository

import "sync"

type MemoryRepository struct {
	mu       sync.RWMutex
	data     map[string]string
	reverse  map[string]string   // original_url → short_id
	userURLs map[string][]string // user_id → []short_id
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		data:     make(map[string]string),
		reverse:  make(map[string]string),
		userURLs: make(map[string][]string),
	}
}

func (r *MemoryRepository) Save(id, url, userID string) error {
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
	if userID != "" {
		r.userURLs[userID] = append(r.userURLs[userID], id)
	}
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
		if e.UserID != "" {
			r.userURLs[e.UserID] = append(r.userURLs[e.UserID], e.ID)
		}
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

func (r *MemoryRepository) GetURLsByUser(userID string) ([]URLPair, error) {
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
