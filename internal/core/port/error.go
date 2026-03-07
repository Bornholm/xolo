package port

import "errors"

var (
	ErrNotFound = errors.New("not found")
	ErrCanceled = errors.New("canceled")
)
