package repository

import "context"

type BatchEntry struct {
	ID     string
	URL    string
	UserID string
}

type URLPair struct {
	ShortID     string
	OriginalURL string
}

type URLRepository interface {
	Save(ctx context.Context, id, url, userID string) error
	SaveBatch(ctx context.Context, entries []BatchEntry) error
	Get(ctx context.Context, id string) (url string, err error)
	GetByOriginalURL(ctx context.Context, url string) (id string, ok bool, err error)
	GetURLsByUser(ctx context.Context, userID string) ([]URLPair, error)
	DeleteUserURLs(ctx context.Context, shortIDs []string, userID string) error
}
