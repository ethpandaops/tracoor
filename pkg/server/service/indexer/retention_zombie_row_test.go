package indexer

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/ethpandaops/beacon/pkg/human"
	"github.com/ethpandaops/tracoor/pkg/server/ethereum"
	"github.com/ethpandaops/tracoor/pkg/server/persistence"
	"github.com/ethpandaops/tracoor/pkg/store"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func newRetentionTestIndexer(t *testing.T) *Indexer {
	t.Helper()

	ctx := context.Background()

	dbFile, err := os.CreateTemp("", "retention_zombie_row_*.db")
	require.NoError(t, err)
	dbPath := dbFile.Name()
	dbFile.Close()
	os.Remove(dbPath)

	t.Cleanup(func() {
		os.Remove(dbPath)
		os.Remove(dbPath + "-wal")
		os.Remove(dbPath + "-shm")
	})

	db, err := persistence.NewIndexer("retention-zombie-row-test", logrus.New(), persistence.Config{
		DSN:        fmt.Sprintf("file:%s?parseTime=True", dbPath),
		DriverName: "sqlite",
	}, persistence.DefaultOptions().SetMetricsEnabled(false))
	require.NoError(t, err)
	require.NoError(t, db.Start(ctx))

	basePath, err := os.MkdirTemp("", "retention_zombie_row_fs")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(basePath) })

	st, err := store.NewFSStore("retention-zombie-row-test", logrus.New(), &store.FSStoreConfig{BasePath: basePath}, &store.Options{})
	require.NoError(t, err)

	conf := &Config{
		Retention: RetentionConfig{
			BeaconStates: human.Duration{Duration: time.Minute},
		},
	}

	idx, err := NewIndexer(ctx, logrus.New(), conf, db, st, &ethereum.Config{})
	require.NoError(t, err)

	return idx
}

// TestPurgeOldBeaconStates_DeletesDBRowWhenFileAlreadyMissing is a
// regression test for NM-W1-003. Before the fix, a beacon state row whose
// underlying file was already gone from disk (manual cleanup, a crash
// mid-delete, or one of a pair of duplicate rows sharing a location where a
// sibling delete already removed it) could never be purged: FSStore's
// DeleteBeaconState returned a raw *PathError that didn't satisfy
// errors.Is(err, store.ErrNotFound), so the retention loop treated it as a
// real failure, skipped the database delete, and retried forever.
func TestPurgeOldBeaconStates_DeletesDBRowWhenFileAlreadyMissing(t *testing.T) {
	idx := newRetentionTestIndexer(t)
	ctx := context.Background()

	// FetchedAt always arrives from a protobuf timestamp's AsTime(), which is
	// always UTC-located; matching that here rather than using the server's
	// local time.Now() keeps this test representative of production data.
	oldFetchedAt := time.Now().UTC().Add(-time.Hour)

	require.NoError(t, idx.db.InsertBeaconState(ctx, &persistence.BeaconState{
		ID:                   "zombie-state",
		Node:                 "some-node",
		Network:              "mainnet",
		Slot:                 100,
		Epoch:                3,
		StateRoot:            "0xzombie",
		FetchedAt:            oldFetchedAt,
		BeaconImplementation: "teku",
		NodeVersion:          "1.0.0",
		// This location was never written to the FS store, standing in for
		// a file that's already gone by the time retention gets to it.
		Location: "beacon_state/already_gone.ssz",
	}))

	require.NoError(t, idx.purgeOldBeaconStates(ctx))

	count, err := idx.db.CountBeaconState(ctx, &persistence.BeaconStateFilter{})
	require.NoError(t, err)
	require.Equal(t, int64(0), count, "expected the database row to be purged even though its file was already missing")
}

// TestPurgeOldBeaconStates_LeavesRecentStatesAlone is a baseline check that
// the fix didn't change which rows are eligible for purging.
func TestPurgeOldBeaconStates_LeavesRecentStatesAlone(t *testing.T) {
	idx := newRetentionTestIndexer(t)
	ctx := context.Background()

	require.NoError(t, idx.db.InsertBeaconState(ctx, &persistence.BeaconState{
		ID:                   "recent-state",
		Node:                 "some-node",
		Network:              "mainnet",
		Slot:                 200,
		Epoch:                6,
		StateRoot:            "0xrecent",
		FetchedAt:            time.Now().UTC(),
		BeaconImplementation: "teku",
		NodeVersion:          "1.0.0",
		Location:             "beacon_state/recent.ssz",
	}))

	require.NoError(t, idx.purgeOldBeaconStates(ctx))

	count, err := idx.db.CountBeaconState(ctx, &persistence.BeaconStateFilter{})
	require.NoError(t, err)
	require.Equal(t, int64(1), count, "a state fetched within the retention window must not be purged")
}
