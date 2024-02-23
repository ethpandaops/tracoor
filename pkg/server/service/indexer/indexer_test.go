package indexer

import (
	"context"
	"testing"
	"time"

	"github.com/ethpandaops/beacon/pkg/human"
	pindexer "github.com/ethpandaops/tracoor/pkg/proto/tracoor/indexer"
	"google.golang.org/grpc"
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
		if err := cleanup(); err != nil {
			t.Fatalf("failed to cleanup: %v", err)
		}
	}()

	node := "test-node"
	token := "abc"

	// Create a storage handshake token
	if err = index.Store().SaveStorageHandshakeToken(ctx, node, token); err != nil {
		t.Fatalf("failed to save storage handshake token: %v", err)
	}

	_, err = index.GetStorageHandshakeToken(ctx, &pindexer.GetStorageHandshakeTokenRequest{
		Node:  node,
		Token: token,
	})
	if err != nil {
		t.Fatalf("failed to get storage handshake token: %v", err)
	}
}
