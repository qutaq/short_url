package repository

import "context"

// BatchEntry описывает одну запись URL для сохранения в пакетной операции.
type BatchEntry struct {
	// ID содержит сгенерированный идентификатор короткой ссылки.
	ID string
	// URL содержит исходный URL.
	URL string
	// UserID содержит идентификатор владельца записи URL.
	UserID string
}

// Stats содержит агрегированную статистику хранилища URL.
type Stats struct {
	// URLs содержит общее количество сокращённых URL.
	URLs int
	// Users содержит количество уникальных пользователей.
	Users int
}

// URLPair содержит сохранённый идентификатор короткой ссылки и исходный URL.
type URLPair struct {
	// ShortID содержит сохранённый идентификатор короткой ссылки.
	ShortID string
	// OriginalURL содержит URL, связанный с ShortID.
	OriginalURL string
}

// URLRepository описывает операции хранилища, необходимые сервису сокращения URL.
type URLRepository interface {
	// Save сохраняет одну запись короткой ссылки.
	Save(ctx context.Context, id, url, userID string) error
	// SaveBatch сохраняет несколько записей коротких ссылок атомарно, если хранилище это поддерживает.
	SaveBatch(ctx context.Context, entries []BatchEntry) error
	// Get возвращает исходный URL по идентификатору короткой ссылки.
	Get(ctx context.Context, id string) (url string, err error)
	// GetByOriginalURL ищет существующий идентификатор короткой ссылки по исходному URL.
	GetByOriginalURL(ctx context.Context, url string) (id string, ok bool, err error)
	// GetURLsByUser возвращает все записи URL, принадлежащие пользователю.
	GetURLsByUser(ctx context.Context, userID string) ([]URLPair, error)
	// DeleteUserURLs помечает URL, принадлежащие пользователю, как удалённые.
	DeleteUserURLs(ctx context.Context, shortIDs []string, userID string) error
	// GetStats возвращает количество сокращённых URL и пользователей в хранилище.
	GetStats(ctx context.Context) (Stats, error)
}
