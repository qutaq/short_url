package repository

import "errors"

var (
	ErrConflict  = errors.New("repository: id already exists")
	ErrURLExists = errors.New("repository: original url already exists")
)
