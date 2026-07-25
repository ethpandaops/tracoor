package indexer

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	pindexer "github.com/ethpandaops/tracoor/pkg/proto/tracoor/indexer"
	"github.com/ethpandaops/tracoor/pkg/server/ethereum"
	"github.com/ethpandaops/tracoor/pkg/server/persistence"
	"github.com/ethpandaops/tracoor/pkg/store"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// newLocationOwnershipTestIndexer builds a real Indexer backed by a
// file-backed SQLite database and an FS store, avoiding the Docker/Minio
// dependency that NewMockIndexer requires.
func newLocationOwnershipTestIndexer(t *testing.T) *Indexer {
	t.Helper()

	ctx := context.Background()

	dbFile, err := os.CreateTemp("", "location_ownership_*.db")
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

	db, err := persistence.NewIndexer("location-ownership-test", logrus.New(), persistence.Config{
		DSN:        fmt.Sprintf("file:%s?parseTime=True", dbPath),
		DriverName: "sqlite",
	}, persistence.DefaultOptions().SetMetricsEnabled(false))
	if err != nil {
		t.Fatalf("failed to create persistence indexer: %v", err)
	}
	if err := db.Start(ctx); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	basePath, err := os.MkdirTemp("", "location_ownership_fs")
	if err != nil {
		t.Fatalf("failed to create temp fs dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(basePath) })

	st, err := store.NewFSStore("location-ownership-test", logrus.New(), &store.FSStoreConfig{BasePath: basePath}, &store.Options{})
	if err != nil {
		t.Fatalf("failed to create FS store: %v", err)
	}

	idx, err := NewIndexer(ctx, logrus.New(), &Config{}, db, st, &ethereum.Config{})
	if err != nil {
		t.Fatalf("failed to create indexer: %v", err)
	}

	return idx
}

func TestCreateBeaconState_RejectsLocationOwnedByADifferentRecord(t *testing.T) {
	idx := newLocationOwnershipTestIndexer(t)
	ctx := context.Background()

	location := "beacon_state/shared-location.ssz"
	data := []byte("original owner's data")

	if _, err := idx.Store().SaveBeaconState(ctx, &store.SaveParams{Data: &data, Location: location}); err != nil {
		t.Fatalf("failed to pre-upload blob: %v", err)
	}

	// The legitimate record.
	if _, err := idx.CreateBeaconState(ctx, &pindexer.CreateBeaconStateRequest{
		Node:                 wrapperspb.String("owner-node"),
		Slot:                 wrapperspb.UInt64(100),
		Epoch:                wrapperspb.UInt64(3),
		StateRoot:            wrapperspb.String("0xowner"),
		FetchedAt:            timestamppb.New(time.Now()),
		BeaconImplementation: wrapperspb.String("teku"),
		NodeVersion:          wrapperspb.String("1.0.0"),
		Location:             wrapperspb.String(location),
		Network:              wrapperspb.String("mainnet"),
	}); err != nil {
		t.Fatalf("legitimate create failed: %v", err)
	}

	// A second record with a completely different identity, but the same
	// location, must be rejected.
	_, err := idx.CreateBeaconState(ctx, &pindexer.CreateBeaconStateRequest{
		Node:                 wrapperspb.String("someone-elses-node"),
		Slot:                 wrapperspb.UInt64(1),
		Epoch:                wrapperspb.UInt64(0),
		StateRoot:            wrapperspb.String("0xdifferent"),
		FetchedAt:            timestamppb.New(time.Now()),
		BeaconImplementation: wrapperspb.String("teku"),
		NodeVersion:          wrapperspb.String("1.0.0"),
		Location:             wrapperspb.String(location),
		Network:              wrapperspb.String("mainnet"),
	})
	if err == nil {
		t.Fatal("expected the colliding create to be rejected, got no error")
	}

	countRsp, err := idx.CountBeaconState(ctx, &pindexer.CountBeaconStateRequest{})
	if err != nil {
		t.Fatalf("failed to count: %v", err)
	}
	if countRsp.Count.Value != 1 {
		t.Fatalf("expected exactly 1 row after the rejected collision attempt, got %d", countRsp.Count.Value)
	}
}

func TestCreateBeaconState_AllowsRetryOfTheSameRecord(t *testing.T) {
	idx := newLocationOwnershipTestIndexer(t)
	ctx := context.Background()

	location := "beacon_state/retry.ssz"
	data := []byte("data")

	if _, err := idx.Store().SaveBeaconState(ctx, &store.SaveParams{Data: &data, Location: location}); err != nil {
		t.Fatalf("failed to pre-upload blob: %v", err)
	}

	req := &pindexer.CreateBeaconStateRequest{
		Node:                 wrapperspb.String("retrying-node"),
		Slot:                 wrapperspb.UInt64(50),
		Epoch:                wrapperspb.UInt64(1),
		StateRoot:            wrapperspb.String("0xretry"),
		FetchedAt:            timestamppb.New(time.Now()),
		BeaconImplementation: wrapperspb.String("teku"),
		NodeVersion:          wrapperspb.String("1.0.0"),
		Location:             wrapperspb.String(location),
		Network:              wrapperspb.String("mainnet"),
	}

	if _, err := idx.CreateBeaconState(ctx, req); err != nil {
		t.Fatalf("first create failed: %v", err)
	}

	// A retry of the exact same record (for example, an agent resubmitting
	// after a timeout) must still be recognized as AlreadyExists, not
	// treated as a location collision.
	_, err := idx.CreateBeaconState(ctx, req)
	if err == nil {
		t.Fatal("expected AlreadyExists for a same-identity retry, got no error")
	}

	countRsp, err := idx.CountBeaconState(ctx, &pindexer.CountBeaconStateRequest{})
	if err != nil {
		t.Fatalf("failed to count: %v", err)
	}
	if countRsp.Count.Value != 1 {
		t.Fatalf("expected exactly 1 row, got %d", countRsp.Count.Value)
	}
}

func TestCreateExecutionBadBlock_RejectsLocationOwnedByADifferentRecord(t *testing.T) {
	idx := newLocationOwnershipTestIndexer(t)
	ctx := context.Background()

	location := "execution_bad_block/shared-location.json"

	if _, err := idx.CreateExecutionBadBlock(ctx, &pindexer.CreateExecutionBadBlockRequest{
		Node:                    wrapperspb.String("owner-node"),
		BlockHash:               wrapperspb.String("0xowner"),
		BlockNumber:             wrapperspb.Int64(1),
		FetchedAt:               timestamppb.New(time.Now()),
		Location:                wrapperspb.String(location),
		ContentEncoding:         wrapperspb.String("gzip"),
		Network:                 wrapperspb.String("mainnet"),
		ExecutionImplementation: wrapperspb.String("geth"),
		NodeVersion:             wrapperspb.String("1.0.0"),
	}); err != nil {
		t.Fatalf("legitimate create failed: %v", err)
	}

	// Before this fix, CreateExecutionBadBlock had no dedup or ownership
	// check at all, so this collision attempt would have silently
	// succeeded.
	_, err := idx.CreateExecutionBadBlock(ctx, &pindexer.CreateExecutionBadBlockRequest{
		Node:                    wrapperspb.String("attacker-node"),
		BlockHash:               wrapperspb.String("0xdifferent"),
		BlockNumber:             wrapperspb.Int64(2),
		FetchedAt:               timestamppb.New(time.Now()),
		Location:                wrapperspb.String(location),
		ContentEncoding:         wrapperspb.String("gzip"),
		Network:                 wrapperspb.String("mainnet"),
		ExecutionImplementation: wrapperspb.String("geth"),
		NodeVersion:             wrapperspb.String("1.0.0"),
	})
	if err == nil {
		t.Fatal("expected the colliding create to be rejected, got no error")
	}

	countRsp, err := idx.CountExecutionBadBlock(ctx, &pindexer.CountExecutionBadBlockRequest{})
	if err != nil {
		t.Fatalf("failed to count: %v", err)
	}
	if countRsp.Count.Value != 1 {
		t.Fatalf("expected exactly 1 row after the rejected collision attempt, got %d", countRsp.Count.Value)
	}
}
