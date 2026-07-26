package indexer

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/tracoor/pkg/server/persistence"
	"github.com/ethpandaops/tracoor/pkg/store"
	"github.com/sirupsen/logrus"
)

// newFileBackedPermanentStore builds a real PermanentStore backed by a
// file-backed SQLite database and an FS store, avoiding the Docker/Minio
// dependency the rest of this package's permanent store tests require. It
// returns the store, the underlying store.Store, and the raw database path
// so callers can inject a real, targeted database failure.
func newFileBackedPermanentStore(t *testing.T) (*PermanentStore, store.Store, string) {
	t.Helper()

	ctx := context.Background()

	dbFile, err := os.CreateTemp("", "permanent_store_cache_*.db")
	if err != nil {
		t.Fatalf("failed to create temp db file: %v", err)
	}
	dbPath := dbFile.Name()
	dbFile.Close()
	os.Remove(dbPath)

	t.Cleanup(func() {
		os.Remove(dbPath)
		os.Remove(dbPath + "-wal")
		os.Remove(dbPath + "-shm")
	})

	db, err := persistence.NewIndexer("permanent-store-cache-test", logrus.New(), persistence.Config{
		DSN:        fmt.Sprintf("file:%s?parseTime=True", dbPath),
		DriverName: "sqlite",
	}, persistence.DefaultOptions().SetMetricsEnabled(false))
	if err != nil {
		t.Fatalf("failed to create persistence indexer: %v", err)
	}
	if err := db.Start(ctx); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	basePath, err := os.MkdirTemp("", "permanent_store_cache_fs")
	if err != nil {
		t.Fatalf("failed to create temp fs dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(basePath) })

	st, err := store.NewFSStore("permanent-store-cache-test", logrus.New(), &store.FSStoreConfig{BasePath: basePath}, &store.Options{})
	if err != nil {
		t.Fatalf("failed to create FS store: %v", err)
	}

	permanentStore, err := NewPermanentStore(logrus.New(), st, db, "cache-test-node", &PermanentStoreConfig{
		Blocks: BlockConfig{Enabled: true},
	})
	if err != nil {
		t.Fatalf("failed to create permanent store: %v", err)
	}

	return permanentStore, st, dbPath
}

// dropPermanentBlocksTable forces every future recordPermanentBlock call to
// fail with a real, clean SQL error, without disturbing the distributed_locks
// table AcquireLock/ReleaseLock depend on. The connection used for the DDL
// is opened, used, and closed immediately so it never contends with the
// permanent store's own connection pool.
func dropPermanentBlocksTable(t *testing.T, dbPath string) {
	t.Helper()

	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open raw connection: %v", err)
	}
	defer conn.Close()

	if _, err := conn.ExecContext(context.Background(), "DROP TABLE permanent_blocks"); err != nil {
		t.Fatalf("failed to drop permanent_blocks table: %v", err)
	}
}

// TestProcessBlock_DoesNotCacheWhenRecordingFails is a regression test: a
// real (not mocked) database failure while recording a block must not be
// treated as success. Before this fix, processBlock logged the failure but
// still cached the block as done and returned nil, so a later retry (for
// example retention's confirm-before-delete check) would short-circuit on
// the cache without ever noticing the block was never actually durable.
func TestProcessBlock_DoesNotCacheWhenRecordingFails(t *testing.T) {
	permanentStore, st, dbPath := newFileBackedPermanentStore(t)
	ctx := context.Background()

	blockData := []byte("block bytes")
	blockLocation := "beacon_block/cache-poison-test.ssz"

	if _, err := st.SaveBeaconBlock(ctx, &store.SaveParams{Data: &blockData, Location: blockLocation}); err != nil {
		t.Fatalf("failed to pre-upload block: %v", err)
	}

	dropPermanentBlocksTable(t, dbPath)

	block := PermanentStoreBlock{
		Location:      blockLocation,
		BlockRoot:     "0xcachepoison",
		Network:       "mainnet",
		Slot:          phase0.Slot(1),
		ProcessedChan: make(chan error, 1),
	}

	// First attempt: the copy succeeds, but recording it fails because the
	// table is gone.
	firstErr := permanentStore.processBlock(ctx, block)
	if firstErr == nil {
		t.Fatal("expected processBlock to return an error when recording fails")
	}

	select {
	case reported := <-block.ProcessedChan:
		if reported == nil {
			t.Fatal("expected a non-nil error to be reported on ProcessedChan")
		}
	default:
		t.Fatal("expected a result to be available on ProcessedChan immediately")
	}

	permanentLocation := permanentStore.GetPermanentLocation(block)

	copied, err := st.Exists(ctx, permanentLocation)
	if err != nil || !copied {
		t.Fatalf("expected the blob to have been copied despite the record failure, exists=%v err=%v", copied, err)
	}

	// Second attempt, same block: if the cache had been poisoned, this
	// would hit the cache-hit shortcut and return nil without even trying
	// to record again. It must instead try again and fail again, since the
	// table is still gone.
	block2 := PermanentStoreBlock{
		Location:      blockLocation,
		BlockRoot:     "0xcachepoison",
		Network:       "mainnet",
		Slot:          phase0.Slot(1),
		ProcessedChan: make(chan error, 1),
	}

	secondErr := permanentStore.processBlock(ctx, block2)
	if secondErr == nil {
		t.Fatal("expected the second processBlock call to also fail, cache was incorrectly poisoned by the first failure")
	}
}

// TestProcessBlock_RecordsSuccessfullyAfterATransientFailureClears shows the
// positive side of the same fix: once the underlying problem is gone, a
// retried attempt for the same block succeeds and is cached normally.
func TestProcessBlock_RecordsSuccessfullyAfterATransientFailureClears(t *testing.T) {
	permanentStore, st, dbPath := newFileBackedPermanentStore(t)
	ctx := context.Background()

	blockData := []byte("block bytes")
	blockLocation := "beacon_block/recovers.ssz"

	if _, err := st.SaveBeaconBlock(ctx, &store.SaveParams{Data: &blockData, Location: blockLocation}); err != nil {
		t.Fatalf("failed to pre-upload block: %v", err)
	}

	dropPermanentBlocksTable(t, dbPath)

	block := PermanentStoreBlock{
		Location:      blockLocation,
		BlockRoot:     "0xrecovers",
		Network:       "mainnet",
		Slot:          phase0.Slot(1),
		ProcessedChan: make(chan error, 1),
	}

	if err := permanentStore.processBlock(ctx, block); err == nil {
		t.Fatal("expected the first attempt to fail while the table is missing")
	}

	// Recreate the table, standing in for whatever transient problem caused
	// the original failure being resolved (a DB coming back, disk space
	// freed up, and so on).
	db, err := persistence.NewIndexer("permanent-store-cache-test-recover", logrus.New(), persistence.Config{
		DSN:        fmt.Sprintf("file:%s?parseTime=True", dbPath),
		DriverName: "sqlite",
	}, persistence.DefaultOptions().SetMetricsEnabled(false))
	if err != nil {
		t.Fatalf("failed to reconnect: %v", err)
	}
	if err := db.Start(ctx); err != nil {
		t.Fatalf("failed to re-migrate: %v", err)
	}

	block2 := PermanentStoreBlock{
		Location:      blockLocation,
		BlockRoot:     "0xrecovers",
		Network:       "mainnet",
		Slot:          phase0.Slot(1),
		ProcessedChan: make(chan error, 1),
	}

	if err := permanentStore.processBlock(ctx, block2); err != nil {
		t.Fatalf("expected the retried attempt to succeed once the table exists again, got: %v", err)
	}

	permanentBlock, err := db.GetPermanentBlockByBlockRoot(ctx, block2.BlockRoot, block2.Network)
	if err != nil {
		t.Fatalf("expected the block to now be recorded in the database: %v", err)
	}
	if permanentBlock == nil {
		t.Fatal("expected a non-nil permanent block record")
	}
}
