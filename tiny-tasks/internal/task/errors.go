package task

import "errors"

var (
	ErrInvalidTitle    = errors.New("title must be at least 3 characters")
	ErrNoFieldsToPatch = errors.New("provide at least one field: title or completed")
)
