package persistence

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

// The pragmas ride the DSN so that every connection the driver opens gets them, rather than
// whichever one happened to serve the statement.
func TestSQLitePragmasAreAppliedToTheConnection(t *testing.T) {
	indexer, err := NewIndexer("pragma_test", logrus.New(), Config{
		DSN:        "file:" + filepath.Join(t.TempDir(), "tracoor.db"),
		DriverName: "sqlite",
	}, DefaultOptions().SetMetricsEnabled(false))
	require.NoError(t, err)
	require.NoError(t, indexer.Start(context.Background()))

	var journalMode string

	require.NoError(t, indexer.db.Raw("PRAGMA journal_mode").Scan(&journalMode).Error)
	require.Equal(t, "wal", journalMode)

	var busyTimeout int

	require.NoError(t, indexer.db.Raw("PRAGMA busy_timeout").Scan(&busyTimeout).Error)
	require.Equal(t, 10000, busyTimeout)
}

func TestWithSQLitePragmas(t *testing.T) {
	require.Equal(t, "file:x.db?"+sqlitePragmas, withSQLitePragmas("file:x.db"))
	require.Equal(t, "file:x.db?mode=rw&"+sqlitePragmas, withSQLitePragmas("file:x.db?mode=rw"))

	// An operator who has spelled out their own pragmas keeps them verbatim.
	custom := "file:x.db?_pragma=busy_timeout(1)"
	require.Equal(t, custom, withSQLitePragmas(custom))
}
