package repository

import "errors"

var (
	// ErrConflict сообщает, что сгенерированный идентификатор короткой ссылки уже существует.
	ErrConflict = errors.New("repository: id already exists")
	// ErrURLExists сообщает, что исходный URL уже сохранён.
	ErrURLExists = errors.New("repository: original url already exists")
	// ErrNotFound сообщает, что идентификатор короткой ссылки не найден.
	ErrNotFound = errors.New("repository: url not found")
	// ErrDeleted сообщает, что короткая ссылка помечена как удалённая.
	ErrDeleted = errors.New("repository: url is deleted")
	// ErrUserIDLen сообщает, что userID слишком длинный для слоя хранения.
	ErrUserIDLen = errors.New("repository: user id exceeds 32 characters")
)
