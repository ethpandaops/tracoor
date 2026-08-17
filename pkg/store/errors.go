package store

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrAlreadyStored = errors.New("already stored")
	ErrInvalid       = errors.New("invalid")
)

// DeleteManyError reports which locations a bulk delete could not remove. A bulk delete is
// partial by nature: one bad key must not condemn the rest of the batch, and the caller needs
// the failed subset so it can retry or quarantine those locations on their own.
type DeleteManyError struct {
	Failed []string
	Err    error
}

func (e *DeleteManyError) Error() string {
	return fmt.Sprintf("failed to delete %d objects: %v", len(e.Failed), e.Err)
}

func (e *DeleteManyError) Unwrap() error {
	return e.Err
}
