package store

import "errors"

var (
	ErrNotFound      = errors.New("not found")
	ErrAlreadyStored = errors.New("already stored")
	ErRInvalid       = errors.New("invalid")
)
