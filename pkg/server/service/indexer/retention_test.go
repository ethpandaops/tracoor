package indexer

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/ethpandaops/beacon/pkg/human"
	pindexer "github.com/ethpandaops/tracoor/pkg/proto/tracoor/indexer"
	"github.com/ethpandaops/tracoor/pkg/server/ethereum"
	"github.com/ethpandaops/tracoor/pkg/server/persistence"
	"github.com/ethpandaops/tracoor/pkg/store"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// newRetentionTestIndexer builds a real Indexer backed by a file-backed
// SQLite database and an FS store, avoiding the Docker/Minio dependency
// that NewMockIndexer requires.
func newRetentionTestIndexer(t *testing.T, conf *Config) *Indexer {
	t.Helper()

	ctx := context.Background()

	dbFile, err := os.CreateTemp("", "retention_test_*.db")
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

	db, err := persistence.NewIndexer("retention-test", logrus.New(), persistence.Config{
		DSN:        fmt.Sprintf("file:%s?parseTime=True", dbPath),
		DriverName: "sqlite",
	}, persistence.DefaultOptions().SetMetricsEnabled(false))
	if err != nil {
		t.Fatalf("failed to create persistence indexer: %v", err)
	}
	if err := db.Start(ctx); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	basePath, err := os.MkdirTemp("", "retention_test_fs")
	if err != nil {
		t.Fatalf("failed to create temp fs dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(basePath) })

	st, err := store.NewFSStore("retention-test", logrus.New(), &store.FSStoreConfig{BasePath: basePath}, &store.Options{})
	if err != nil {
		t.Fatalf("failed to create FS store: %v", err)
	}

	idx, err := NewIndexer(ctx, logrus.New(), conf, db, st, &ethereum.Config{})
	if err != nil {
		t.Fatalf("failed to create indexer: %v", err)
	}

	return idx
}

// TestPurgeOldBeaconBlocks_DoesNotBlockWhenPermanentStoreDisabled is a
// regression test: purgeOldBeaconBlocks used to queue every aged-out block
// with the permanent store and then wait on a channel that was never
// closed whenever the permanent store was disabled (the default
// configuration), hanging the retention loop forever. It must now complete
// within a bounded time and actually purge the block.
func TestPurgeOldBeaconBlocks_DoesNotBlockWhenPermanentStoreDisabled(t *testing.T) {
	idx := newRetentionTestIndexer(t, &Config{
		Retention: RetentionConfig{BeaconBlocks: human.Duration{Duration: 1 * time.Minute}},
		// PermanentStore left at its zero value: Blocks.Enabled == false,
		// matching the real documented default.
	})

	ctx := context.Background()

	if idx.permanentStore.IsEnabled() {
		t.Fatal("test setup error: permanent store unexpectedly enabled")
	}

	location := "beacon_block/aged-out.ssz"
	data := []byte("a beacon block aged past its retention window")

	if _, err := idx.Store().SaveBeaconBlock(ctx, &store.SaveParams{Data: &data, Location: location}); err != nil {
		t.Fatalf("failed to pre-upload block: %v", err)
	}

	if _, err := idx.CreateBeaconBlock(ctx, &pindexer.CreateBeaconBlockRequest{
		Node:                 wrapperspb.String("some-node"),
		Slot:                 wrapperspb.UInt64(12345),
		Epoch:                wrapperspb.UInt64(385),
		BlockRoot:            wrapperspb.String("0xagedblock"),
		FetchedAt:            timestamppb.New(time.Now().Add(-2 * time.Minute)),
		BeaconImplementation: wrapperspb.String("teku"),
		NodeVersion:          wrapperspb.String("1.0.0"),
		Location:             wrapperspb.String(location),
		Network:              wrapperspb.String("mainnet"),
	}); err != nil {
		t.Fatalf("failed to create beacon block: %v", err)
	}

	done := make(chan error, 1)

	go func() {
		done <- idx.purgeOldBeaconBlocks(ctx)
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("purgeOldBeaconBlocks returned an error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("purgeOldBeaconBlocks did not return within 5 seconds -- it is blocked, the deadlock regressed")
	}

	countRsp, err := idx.CountBeaconBlock(ctx, &pindexer.CountBeaconBlockRequest{})
	if err != nil {
		t.Fatalf("failed to count: %v", err)
	}
	if countRsp.Count.Value != 0 {
		t.Fatalf("expected the aged-out block to be purged, but %d rows remain", countRsp.Count.Value)
	}
}

// TestPurgeOldBeaconBlocks_RespectsContextCancellation covers the shutdown
// path specifically. The permanent store is enabled but its worker
// goroutines are never started, so a queued block is accepted into the
// channel but never actually processed (standing in for workers being busy
// or otherwise not keeping up) -- ProcessedChan is never closed on its own.
// The only way purgeOldBeaconBlocks can still return promptly is by
// noticing the context was cancelled.
func TestPurgeOldBeaconBlocks_RespectsContextCancellation(t *testing.T) {
	idx := newRetentionTestIndexer(t, &Config{
		Retention:      RetentionConfig{BeaconBlocks: human.Duration{Duration: 1 * time.Minute}},
		PermanentStore: PermanentStoreConfig{Blocks: BlockConfig{Enabled: true}},
	})

	bgCtx := context.Background()

	location := "beacon_block/aged-out.ssz"
	data := []byte("data")

	if _, err := idx.Store().SaveBeaconBlock(bgCtx, &store.SaveParams{Data: &data, Location: location}); err != nil {
		t.Fatalf("failed to pre-upload block: %v", err)
	}

	if _, err := idx.CreateBeaconBlock(bgCtx, &pindexer.CreateBeaconBlockRequest{
		Node:                 wrapperspb.String("some-node"),
		Slot:                 wrapperspb.UInt64(1),
		Epoch:                wrapperspb.UInt64(0),
		BlockRoot:            wrapperspb.String("0xagedblock"),
		FetchedAt:            timestamppb.New(time.Now().Add(-2 * time.Minute)),
		BeaconImplementation: wrapperspb.String("teku"),
		NodeVersion:          wrapperspb.String("1.0.0"),
		Location:             wrapperspb.String(location),
		Network:              wrapperspb.String("mainnet"),
	}); err != nil {
		t.Fatalf("failed to create beacon block: %v", err)
	}

	// Deliberately not calling idx.permanentStore.Start(): nothing ever
	// consumes from the queue, so QueueBlock delivers the block
	// successfully but it is never handed to processBlock, and
	// ProcessedChan is never closed by that path.
	ctx, cancel := context.WithCancel(bgCtx)

	done := make(chan error, 1)

	go func() {
		done <- idx.purgeOldBeaconBlocks(ctx)
	}()

	// Give the goroutine a moment to reach the blocking wait, then cancel.
	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected purgeOldBeaconBlocks to return the cancellation error")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("purgeOldBeaconBlocks did not return promptly after context cancellation")
	}
}
