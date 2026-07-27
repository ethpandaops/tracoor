package indexer

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"

	pindexer "github.com/ethpandaops/tracoor/pkg/proto/tracoor/indexer"
	"github.com/ethpandaops/tracoor/pkg/server/ethereum"
	"github.com/ethpandaops/tracoor/pkg/server/persistence"
	"github.com/ethpandaops/tracoor/pkg/store"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func newUniqueConstraintTestIndexer(t *testing.T) *Indexer {
	t.Helper()

	ctx := context.Background()

	dbFile, err := os.CreateTemp("", "unique_constraint_app_*.db")
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

	db, err := persistence.NewIndexer("unique-constraint-test", logrus.New(), persistence.Config{
		DSN:        fmt.Sprintf("file:%s?parseTime=True", dbPath),
		DriverName: "sqlite",
	}, persistence.DefaultOptions().SetMetricsEnabled(false))
	if err != nil {
		t.Fatalf("failed to create persistence indexer: %v", err)
	}
	if err := db.Start(ctx); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	basePath, err := os.MkdirTemp("", "unique_constraint_app_fs")
	if err != nil {
		t.Fatalf("failed to create temp fs dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(basePath) })

	st, err := store.NewFSStore("unique-constraint-test", logrus.New(), &store.FSStoreConfig{BasePath: basePath}, &store.Options{})
	if err != nil {
		t.Fatalf("failed to create FS store: %v", err)
	}

	idx, err := NewIndexer(ctx, logrus.New(), &Config{}, db, st, &ethereum.Config{})
	if err != nil {
		t.Fatalf("failed to create indexer: %v", err)
	}

	return idx
}

// TestCreateBeaconState_ConcurrentIdenticalRequestsProduceExactlyOneRow is a
// regression test for the TOCTOU race the audit found: 40 concurrent,
// identical CreateBeaconState calls used to reliably produce 2-3 duplicate
// rows (confirmed empirically in the original audit). With a real database
// constraint backing the dedup check, exactly one must land, and every
// loser must get a clean AlreadyExists rather than a raw SQL error leaking
// through as an Internal error.
func TestCreateBeaconState_ConcurrentIdenticalRequestsProduceExactlyOneRow(t *testing.T) {
	idx := newUniqueConstraintTestIndexer(t)
	ctx := context.Background()

	location := "beacon_state/concurrent.ssz"
	data := []byte("data")

	if _, err := idx.Store().SaveBeaconState(ctx, &store.SaveParams{Data: &data, Location: location}); err != nil {
		t.Fatalf("failed to pre-upload blob: %v", err)
	}

	makeReq := func() *pindexer.CreateBeaconStateRequest {
		return &pindexer.CreateBeaconStateRequest{
			Node:                 wrapperspb.String("race-node"),
			Slot:                 wrapperspb.UInt64(12345),
			Epoch:                wrapperspb.UInt64(1),
			StateRoot:            wrapperspb.String("0xconcurrent"),
			FetchedAt:            timestamppb.Now(),
			BeaconImplementation: wrapperspb.String("teku"),
			NodeVersion:          wrapperspb.String("1.0.0"),
			Location:             wrapperspb.String(location),
			Network:              wrapperspb.String("mainnet"),
		}
	}

	const concurrency = 40

	var (
		wg           sync.WaitGroup
		start        = make(chan struct{})
		succeeded    int
		alreadyExist int
		unexpected   int
		mu           sync.Mutex
	)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()
			<-start

			_, err := idx.CreateBeaconState(ctx, makeReq())

			mu.Lock()
			defer mu.Unlock()

			switch {
			case err == nil:
				succeeded++
			case status.Code(err) == codes.AlreadyExists:
				alreadyExist++
			default:
				unexpected++
				t.Logf("unexpected error (not a clean AlreadyExists): %v", err)
			}
		}()
	}

	close(start)
	wg.Wait()

	countRsp, err := idx.CountBeaconState(ctx, &pindexer.CountBeaconStateRequest{})
	if err != nil {
		t.Fatalf("failed to count: %v", err)
	}

	t.Logf("concurrency=%d succeeded=%d alreadyExists=%d unexpected=%d dbRows=%d",
		concurrency, succeeded, alreadyExist, unexpected, countRsp.Count.Value)

	if countRsp.Count.Value != 1 {
		t.Fatalf("expected exactly 1 row from %d concurrent identical requests, got %d", concurrency, countRsp.Count.Value)
	}

	if succeeded != 1 {
		t.Fatalf("expected exactly 1 caller to see success, got %d", succeeded)
	}

	if unexpected != 0 {
		t.Fatalf("expected every losing caller to see a clean AlreadyExists, got %d unexpected error(s)", unexpected)
	}

	if alreadyExist != concurrency-1 {
		t.Fatalf("expected %d callers to see AlreadyExists, got %d", concurrency-1, alreadyExist)
	}
}

// TestCreateExecutionBadBlock_ConcurrentIdenticalRequestsProduceExactlyOneRow
// covers the handler that previously had zero protection of any kind (not
// even a racy check) -- confirmed in the original audit with a
// deterministic, non-concurrent PoC (2 sequential identical calls both
// succeeded). This exercises it under real concurrency too.
func TestCreateExecutionBadBlock_ConcurrentIdenticalRequestsProduceExactlyOneRow(t *testing.T) {
	idx := newUniqueConstraintTestIndexer(t)
	ctx := context.Background()

	makeReq := func() *pindexer.CreateExecutionBadBlockRequest {
		return &pindexer.CreateExecutionBadBlockRequest{
			Node:                    wrapperspb.String("race-node"),
			BlockHash:               wrapperspb.String("0xconcurrent"),
			BlockNumber:             wrapperspb.Int64(1),
			FetchedAt:               timestamppb.Now(),
			Location:                wrapperspb.String("execution_bad_block/concurrent.json"),
			ContentEncoding:         wrapperspb.String("gzip"),
			Network:                 wrapperspb.String("mainnet"),
			ExecutionImplementation: wrapperspb.String("geth"),
			NodeVersion:             wrapperspb.String("1.0.0"),
		}
	}

	const concurrency = 40

	var (
		wg           sync.WaitGroup
		start        = make(chan struct{})
		succeeded    int
		alreadyExist int
		unexpected   int
		mu           sync.Mutex
	)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()
			<-start

			_, err := idx.CreateExecutionBadBlock(ctx, makeReq())

			mu.Lock()
			defer mu.Unlock()

			switch {
			case err == nil:
				succeeded++
			case status.Code(err) == codes.AlreadyExists:
				alreadyExist++
			default:
				unexpected++
				t.Logf("unexpected error (not a clean AlreadyExists): %v", err)
			}
		}()
	}

	close(start)
	wg.Wait()

	countRsp, err := idx.CountExecutionBadBlock(ctx, &pindexer.CountExecutionBadBlockRequest{})
	if err != nil {
		t.Fatalf("failed to count: %v", err)
	}

	t.Logf("concurrency=%d succeeded=%d alreadyExists=%d unexpected=%d dbRows=%d",
		concurrency, succeeded, alreadyExist, unexpected, countRsp.Count.Value)

	if countRsp.Count.Value != 1 {
		t.Fatalf("expected exactly 1 row from %d concurrent identical requests, got %d", concurrency, countRsp.Count.Value)
	}

	if unexpected != 0 {
		t.Fatalf("expected every losing caller to see a clean AlreadyExists, got %d unexpected error(s)", unexpected)
	}
}

// TestRecordPermanentBlock_ConcurrentCallsForSameIdentitySucceedExactlyOnce
// covers the NM-W2-001 interaction: two callers racing past the distributed
// lock for the same identity (which the lock's fixed 30s TTL with no
// renewal doesn't fully rule out) must both come away believing the block
// is durably recorded, not have one of them treat "someone else already
// recorded it" as a failure.
func TestRecordPermanentBlock_ConcurrentCallsForSameIdentitySucceedExactlyOnce(t *testing.T) {
	idx := newUniqueConstraintTestIndexer(t)
	ctx := context.Background()

	block := PermanentStoreBlock{
		Location:  "beacon_block/permanent.ssz",
		BlockRoot: "0xpermanentrace",
		Network:   "mainnet",
		Slot:      1,
	}

	const concurrency = 20

	var (
		wg    sync.WaitGroup
		start = make(chan struct{})
		errs  = make([]error, concurrency)
	)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)

		go func(idx2 int) {
			defer wg.Done()
			<-start

			errs[idx2] = idx.permanentStore.recordPermanentBlock(ctx, block)
		}(i)
	}

	close(start)
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("call %d: expected recordPermanentBlock to treat a concurrent duplicate as success, got: %v", i, err)
		}
	}

	permanentBlock, err := idx.db.GetPermanentBlockByBlockRoot(ctx, block.BlockRoot, block.Network)
	if err != nil {
		t.Fatalf("expected exactly one permanent block record to exist: %v", err)
	}
	if permanentBlock == nil {
		t.Fatal("expected a non-nil permanent block record")
	}
}
