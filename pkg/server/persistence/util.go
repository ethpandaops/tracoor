package persistence

import (
	"math/rand"
	"time"
)

// utcBound normalises a timestamp before it is bound into a query or written to a row.
//
// The SQLite driver stores a time.Time as text carrying that value's own UTC offset and
// compares those strings byte for byte, so two instants written in different zones do not
// order by instant. Every timestamp this package stores is UTC, so every bound compared
// against one has to be UTC too — otherwise a cutoff built from the process clock is compared
// against stored values by wall-clock digits and a host in a non-UTC zone expires rows that
// are still inside their window.
func utcBound(t time.Time) time.Time {
	return t.UTC()
}

// utcBoundPtr is utcBound for the nullable columns; a nil timestamp stays nil.
func utcBoundPtr(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}

	normalised := t.UTC()

	return &normalised
}

func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	b := make([]byte, length)

	for i := range b {
		//nolint:gosec // Only used in tests
		b[i] = charset[rand.Intn(len(charset))]
	}

	return string(b)
}

func generateRandomInt64() int64 {
	//nolint:gosec // Only used in tests
	return rand.Int63()
}
