package indexer

import (
	"context"
	"fmt"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethpandaops/tracoor/pkg/server/persistence"
	"github.com/ethpandaops/tracoor/pkg/store"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

// delayedCopyStore wraps a real store.Store and blocks inside Copy for a
// fixed duration before delegating, standing in for a slow permanent-copy
// write that's still in flight when Stop is called.
type delayedCopyStore struct {
	store.Store
	delay     time.Duration
	copyStart chan struct{}
	copyDone  atomic.Bool
}

func (d *delayedCopyStore) Copy(ctx context.Context, params *store.CopyParams) error {
	close(d.copyStart)
	time.Sleep(d.delay)

	err := d.Store.Copy(ctx, params)
	d.copyDone.Store(true)

	return err
}

func newStopTestPermanentStore(t *testing.T, delay time.Duration) (*PermanentStore, *delayedCopyStore, *persistence.Indexer) {
	t.Helper()

	ctx := context.Background()

	dbFile, err := os.CreateTemp("", "permanent_store_stop_*.db")
	require.NoError(t, err)
	dbPath := dbFile.Name()
	dbFile.Close()
	os.Remove(dbPath)

	t.Cleanup(func() {
		os.Remove(dbPath)
		os.Remove(dbPath + "-wal")
		os.Remove(dbPath + "-shm")
	})

	db, err := persistence.NewIndexer("permanent-store-stop-test", logrus.New(), persistence.Config{
		DSN:        fmt.Sprintf("file:%s?parseTime=True", dbPath),
		DriverName: "sqlite",
	}, persistence.DefaultOptions().SetMetricsEnabled(false))
	require.NoError(t, err)
	require.NoError(t, db.Start(ctx))

	basePath, err := os.MkdirTemp("", "permanent_store_stop_fs")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(basePath) })

	fsStore, err := store.NewFSStore("permanent-store-stop-test", logrus.New(), &store.FSStoreConfig{BasePath: basePath}, &store.Options{})
	require.NoError(t, err)

	wrapped := &delayedCopyStore{Store: fsStore, delay: delay, copyStart: make(chan struct{})}

	data := []byte("block-data")
	_, err = fsStore.SaveBeaconBlock(ctx, &store.SaveParams{Data: &data, Location: "beacon_block/source.ssz"})
	require.NoError(t, err)

	ps, err := NewPermanentStore(logrus.New(), wrapped, db, "stop-test-node", &PermanentStoreConfig{
		Blocks: BlockConfig{Enabled: true},
	})
	require.NoError(t, err)
	require.NoError(t, ps.Start(ctx))

	return ps, wrapped, db
}

// TestStop_WaitsForInFlightBlockToFinishProcessing is a regression test for
// NM-W2-002: Stop used to return as soon as the queue channel was drained,
// which happens the instant a worker goroutine RECEIVES a block, not when it
// finishes processing it. A caller that tore things down right after Stop
// returned (closing the DB pool, exiting the process) could interrupt a
// still-running store.Copy or database write. Stop must now block until the
// block has actually finished, not just been dequeued.
func TestStop_WaitsForInFlightBlockToFinishProcessing(t *testing.T) {
	delay := 500 * time.Millisecond
	ps, wrapped, db := newStopTestPermanentStore(t, delay)
	ctx := context.Background()

	ps.QueueBlock(PermanentStoreBlock{
		Location:  "beacon_block/source.ssz",
		BlockRoot: "0xstoptest",
		Network:   "mainnet",
		Slot:      1,
	})

	select {
	case <-wrapped.copyStart:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for the block's Copy to start")
	}

	stopStart := time.Now()
	require.NoError(t, ps.Stop(context.Background()))
	stopElapsed := time.Since(stopStart)

	if stopElapsed < delay {
		t.Fatalf("Stop returned after %s, before the in-flight copy's %s delay elapsed -- it did not wait for processing to finish", stopElapsed, delay)
	}

	if !wrapped.copyDone.Load() {
		t.Fatal("Stop returned before the in-flight Copy call completed")
	}

	permanentBlock, err := db.GetPermanentBlockByBlockRoot(ctx, "0xstoptest", "mainnet")
	require.NoError(t, err)
	require.NotNil(t, permanentBlock, "expected the block to be durably recorded by the time Stop returns")
}

// TestStop_ReturnsPromptlyWhenNothingIsQueued confirms the fix didn't turn
// Stop into something that always waits -- an idle store must still stop
// immediately.
func TestStop_ReturnsPromptlyWhenNothingIsQueued(t *testing.T) {
	ps, _, _ := newStopTestPermanentStore(t, 0)

	start := time.Now()
	require.NoError(t, ps.Stop(context.Background()))
	elapsed := time.Since(start)

	if elapsed > 100*time.Millisecond {
		t.Fatalf("Stop took %s with nothing queued, expected it to return promptly", elapsed)
	}
}

// TestStop_RespectsContextCancellation confirms Stop still returns ctx.Err()
// rather than hanging forever if the in-flight work outlives the deadline
// the caller gave it.
func TestStop_RespectsContextCancellation(t *testing.T) {
	ps, wrapped, _ := newStopTestPermanentStore(t, 2*time.Second)

	ps.QueueBlock(PermanentStoreBlock{
		Location:  "beacon_block/source.ssz",
		BlockRoot: "0xctxtest",
		Network:   "mainnet",
		Slot:      1,
	})

	select {
	case <-wrapped.copyStart:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for the block's Copy to start")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := ps.Stop(ctx)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

// TestQueueBlock_RejectedAfterStopDoesNotPanicWaitGroup guards the mutex
// added around the stopped flag and wg.Add: without it, a QueueBlock call
// racing Stop could call wg.Add(1) concurrently with (or after) wg.Wait
// returning, which is a documented sync.WaitGroup misuse.
func TestQueueBlock_RejectedAfterStopDoesNotPanicWaitGroup(t *testing.T) {
	ps, _, _ := newStopTestPermanentStore(t, 0)

	require.NoError(t, ps.Stop(context.Background()))

	done := make(chan struct{})

	go func() {
		defer close(done)

		ps.QueueBlock(PermanentStoreBlock{
			Location:  "beacon_block/source.ssz",
			BlockRoot: "0xafterstop",
			Network:   "mainnet",
			Slot:      1,
		})
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("QueueBlock did not return after Stop")
	}
}
