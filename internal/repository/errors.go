package repository

import "errors"

var (
	ErrConflict = errors.New("repository: id already exists")
)
