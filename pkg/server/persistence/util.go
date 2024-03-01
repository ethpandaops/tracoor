package persistence

import "math/rand"

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
