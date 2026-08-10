package persistence

import "strings"

// IsUniqueConstraintError reports whether err was caused by a unique
// constraint violation, checking for the error signatures produced by both
// database backends this package supports (SQLite and Postgres). Neither
// gorm nor either underlying driver exposes a single driver-agnostic type
// for this, so this checks for the stable, well-known substrings each
// database actually produces in its error message rather than depending on
// either driver's internal error types.
func IsUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}

	msg := err.Error()

	return strings.Contains(msg, "UNIQUE constraint failed") || // SQLite
		strings.Contains(msg, "duplicate key value violates unique constraint") // Postgres
}
