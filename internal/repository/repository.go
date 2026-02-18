package repository

type URLRepository interface {
	Save(id, url string) error

	Get(id string) (url string, ok bool)
}
