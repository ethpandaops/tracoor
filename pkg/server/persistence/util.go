package persistence

import (
	"math/rand"
	"time"
)

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

func timestampFormatForDB(t time.Time) string {
	return t.Truncate(time.Second).UTC().Format("2006-01-02 15:04:05.000000+00:00")
}
