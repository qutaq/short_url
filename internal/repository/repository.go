package repository

type BatchEntry struct {
	ID  string
	URL string
}
type URLRepository interface {
	Save(id, url string) error
	SaveBatch(entries []BatchEntry) error

	Get(id string) (url string, ok bool)
	GetByOriginalURL(url string) (id string, ok bool)
}
