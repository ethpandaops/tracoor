package indexer

import (
	"context"
	"testing"
	"time"

	"github.com/ethpandaops/beacon/pkg/human"
	"github.com/ethpandaops/tracoor/pkg/proto/tracoor/indexer"
	"github.com/ethpandaops/tracoor/pkg/server/persistence"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestIndexerStarts(t *testing.T) {
	ctx := context.Background()
	config := Config{
		Retention: RetentionConfig{
			BeaconStates:         human.Duration{Duration: 30 * time.Minute},
			ExecutionBlockTraces: human.Duration{Duration: 30 * time.Minute},
			ExecutionBadBlocks:   human.Duration{Duration: 312480 * time.Minute},
		},
	}

	index, cleanup, err := NewMockIndexer(ctx, &config)
	if err != nil {
		t.Fatalf("failed to create indexer: %v", err)
	}

	defer func() {
		if err := cleanup(); err != nil {
			t.Fatalf("failed to cleanup: %v", err)
		}
	}()

	if err := index.Start(ctx, grpc.NewServer()); err != nil {
		t.Fatalf("failed to start indexer: %v", err)
	}
}

func TestIndexerStops(t *testing.T) {
	ctx := context.Background()
	config := Config{}

	index, cleanup, err := NewMockIndexer(ctx, &config)
	if err != nil {
		t.Fatalf("failed to create indexer: %v", err)
	}

	defer func() {
		if err := cleanup(); err != nil {
			t.Fatalf("failed to cleanup: %v", err)
		}
	}()

	if err := index.Stop(ctx); err != nil {
		t.Fatalf("failed to stop indexer: %v", err)
	}
}

func TestIndexerGetStorageHandshakeToken(t *testing.T) {
	ctx := context.Background()
	config := Config{}

	index, cleanup, err := NewMockIndexer(ctx, &config)
	if err != nil {
		t.Fatalf("failed to create indexer: %v", err)
	}

	defer func() {
		if err = cleanup(); err != nil {
			t.Fatalf("failed to cleanup: %v", err)
		}
	}()

	node := "test-node"
	token := "abc"

	// Create a storage handshake token
	if err = index.Store().SaveStorageHandshakeToken(ctx, node, token); err != nil {
		t.Fatalf("failed to save storage handshake token: %v", err)
	}

	_, err = index.GetStorageHandshakeToken(ctx, &indexer.GetStorageHandshakeTokenRequest{
		Node:  node,
		Token: token,
	})
	if err != nil {
		t.Fatalf("failed to get storage handshake token: %v", err)
	}
}

func TestIndexerBeaconBlockExpiration(t *testing.T) {
	ctx := context.Background()
	config := Config{
		Retention: RetentionConfig{
			BeaconBlocks: human.Duration{Duration: 99999 * time.Minute},
		},
	}

	index, cleanup, err := NewMockIndexer(ctx, &config)
	if err != nil {
		t.Fatalf("failed to create indexer: %v", err)
	}

	defer func() {
		if err := cleanup(); err != nil {
			t.Fatalf("failed to cleanup: %v", err)
		}
	}()

	// Create a beacon block
	block := &indexer.BeaconBlock{
		Id:                   wrapperspb.String(uuid.New().String()),
		Node:                 wrapperspb.String("test-node"),
		Network:              wrapperspb.String("test-network"),
		Slot:                 wrapperspb.UInt64(1),
		Epoch:                wrapperspb.UInt64(1),
		BlockRoot:            wrapperspb.String("test-block-root"),
		NodeVersion:          wrapperspb.String("test-node-version"),
		Location:             wrapperspb.String("test-location"),
		FetchedAt:            timestamppb.Now(),
		BeaconImplementation: wrapperspb.String("test-implementation"),
	}

	if err := index.db.InsertBeaconBlock(ctx, ProtoBeaconBlockToDBBeaconBlock(block)); err != nil {
		t.Fatalf("failed to insert beacon block: %v", err)
	}

	// Check that the beacon block exists
	filter := &persistence.BeaconBlockFilter{}
	filter.AddID(block.Id.GetValue())

	blocks, err := index.db.ListBeaconBlock(ctx, filter, &persistence.PaginationCursor{Limit: 1, Offset: 0})
	if err != nil {
		t.Fatalf("failed to list beacon blocks: %v", err)
	}

	if len(blocks) == 0 {
		t.Fatalf("beacon block was deleted during cleanup")
	}
	// Run the cleanup process
	if err := index.purgeOldBeaconBlocks(ctx); err != nil {
		t.Fatalf("failed to purge old beacon blocks: %v", err)
	}

	// Check that the beacon block hasn't been deleted
	blocks, err = index.db.ListBeaconBlock(ctx, filter, &persistence.PaginationCursor{Limit: 1, Offset: 0})
	if err != nil {
		t.Fatalf("failed to list beacon blocks: %v", err)
	}

	if len(blocks) == 0 {
		t.Fatalf("beacon block was deleted during cleanup")
	}
}
