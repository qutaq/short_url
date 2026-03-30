package repository

import "errors"

var (
	ErrConflict  = errors.New("repository: id already exists")
	ErrURLExists = errors.New("repository: original url already exists")
	ErrNotFound  = errors.New("repository: url not found")
	ErrDeleted   = errors.New("repository: url is deleted")
)
