package repository

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
	Save(id, url, userID string) error
	SaveBatch(entries []BatchEntry) error

	Get(id string) (url string, ok bool)
	GetByOriginalURL(url string) (id string, ok bool)
	GetURLsByUser(userID string) ([]URLPair, error)
}
