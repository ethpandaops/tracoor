package indexer

import (
	"context"
	"io"
	"net/http"
	"testing"

	pindexer "github.com/ethpandaops/tracoor/pkg/proto/tracoor/indexer"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func createRandomExecutionBadBlockRequest() *pindexer.CreateExecutionBadBlockRequest {
	return &pindexer.CreateExecutionBadBlockRequest{
		Node:                    wrapperspb.String(generateRandomString(5)),
		FetchedAt:               timestamppb.Now(),
		BlockHash:               wrapperspb.String(generateRandomString(64)),
		BlockNumber:             wrapperspb.Int64(generateRandomInt64()),
		Location:                wrapperspb.String(generateRandomString(10)),
		Network:                 wrapperspb.String(generateRandomString(5)),
		ExecutionImplementation: wrapperspb.String(generateRandomString(15)),
		NodeVersion:             wrapperspb.String(generateRandomString(8)),
		BlockExtraData:          wrapperspb.String(generateRandomString(20)),
	}
}

func TestIndexerExecutionBadBlockCount(t *testing.T) {
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

	t.Run("Counting", func(t *testing.T) {
		_, err := index.CreateExecutionBadBlock(ctx, createRandomExecutionBadBlockRequest())
		if err != nil {
			t.Fatalf("failed to create execution bad block: %v", err)
		}

		rsp, err := index.CountExecutionBadBlock(ctx, &pindexer.CountExecutionBadBlockRequest{})
		if err != nil {
			t.Fatalf("failed to count execution bad block: %v", err)
		}

		if rsp.Count.Value != uint64(1) {
			t.Fatalf("expected 1 execution bad block, got %d", rsp.Count.Value)
		}
	})
}

func TestIndexerExecutionBadBlock(t *testing.T) {
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

	t.Run("Creating", func(t *testing.T) {
		_, err := index.CreateExecutionBadBlock(ctx, createRandomExecutionBadBlockRequest())
		if err != nil {
			t.Fatalf("failed to create execution bad block: %v", err)
		}
	})

	t.Run("Creating returns a valid ID", func(t *testing.T) {
		rsp, err := index.CreateExecutionBadBlock(ctx, createRandomExecutionBadBlockRequest())
		if err != nil {
			t.Fatalf("failed to create execution bad block: %v", err)
		}

		if rsp.Id == nil {
			t.Fatalf("expected ID to not be empty")
		}
	})

	t.Run("Handles duplicates", func(t *testing.T) {
		req := createRandomExecutionBadBlockRequest()
		rsp, err := index.CreateExecutionBadBlock(ctx, req)
		if err != nil {
			t.Fatalf("failed to create execution bad block: %v", err)
		}

		if rsp.Id == nil {
			t.Fatalf("expected ID to not be empty")
		}

		_, err = index.CreateExecutionBadBlock(ctx, req)
		if err != nil && err.Error() != "execution bad block already exists" {
			t.Fatal("expected error to be 'execution bad block already exists'")
		}
	})

	t.Run("Basic Listing", func(t *testing.T) {
		req := createRandomExecutionBadBlockRequest()

		resp, err := index.CreateExecutionBadBlock(ctx, req)
		if err != nil {
			t.Fatalf("failed to create execution bad block: %v", err)
		}

		rsp, err := index.ListExecutionBadBlock(ctx, &pindexer.ListExecutionBadBlockRequest{Id: resp.Id.Value})
		if err != nil {
			t.Fatalf("failed to get execution bad block: %v", err)
		}

		if len(rsp.ExecutionBadBlocks) != 1 {
			t.Fatalf("expected 1 execution bad block, got %d", len(rsp.ExecutionBadBlocks))
		}
	})

	t.Run("Can list by filters", func(t *testing.T) {
		req := createRandomExecutionBadBlockRequest()

		_, err := index.CreateExecutionBadBlock(ctx, req)
		if err != nil {
			t.Fatalf("failed to create execution bad block: %v", err)
		}

		t.Run("by Node", func(t *testing.T) {
			filterReq := &pindexer.ListExecutionBadBlockRequest{
				Node: req.Node.Value,
			}

			rsp, err := index.ListExecutionBadBlock(ctx, filterReq)
			if err != nil {
				t.Fatalf("failed to list execution bad blocks by node: %v", err)
			}

			if len(rsp.ExecutionBadBlocks) == 0 {
				t.Fatal("expected at least one execution bad block filtered by node")
			}
		})

		t.Run("by BlockNumber", func(t *testing.T) {
			filterReq := &pindexer.ListExecutionBadBlockRequest{
				BlockNumber: req.BlockNumber.Value,
			}

			rsp, err := index.ListExecutionBadBlock(ctx, filterReq)
			if err != nil {
				t.Fatalf("failed to list execution bad blocks by block number: %v", err)
			}

			if len(rsp.ExecutionBadBlocks) == 0 {
				t.Fatal("expected at least one execution bad block filtered by block number")
			}
		})

		t.Run("by BlockHash", func(t *testing.T) {
			filterReq := &pindexer.ListExecutionBadBlockRequest{
				BlockHash: req.BlockHash.Value,
			}

			rsp, err := index.ListExecutionBadBlock(ctx, filterReq)
			if err != nil {
				t.Fatalf("failed to list execution bad blocks by block hash: %v", err)
			}

			if len(rsp.ExecutionBadBlocks) == 0 {
				t.Fatal("expected at least one execution bad block filtered by block hash")
			}
		})

		t.Run("by Location", func(t *testing.T) {
			filterReq := &pindexer.ListExecutionBadBlockRequest{
				Location: req.Location.Value,
			}

			rsp, err := index.ListExecutionBadBlock(ctx, filterReq)
			if err != nil {
				t.Fatalf("failed to list execution bad blocks by location: %v", err)
			}

			if len(rsp.ExecutionBadBlocks) == 0 {
				t.Fatal("expected at least one execution bad block filtered by location")
			}
		})

		t.Run("by Network", func(t *testing.T) {
			filterReq := &pindexer.ListExecutionBadBlockRequest{
				Network: req.Network.Value,
			}

			rsp, err := index.ListExecutionBadBlock(ctx, filterReq)
			if err != nil {
				t.Fatalf("failed to list execution bad blocks by network: %v", err)
			}

			if len(rsp.ExecutionBadBlocks) == 0 {
				t.Fatal("expected at least one execution bad block filtered by network")
			}
		})

		t.Run("by ExecutionImplementation", func(t *testing.T) {
			filterReq := &pindexer.ListExecutionBadBlockRequest{
				ExecutionImplementation: req.ExecutionImplementation.Value,
			}

			rsp, err := index.ListExecutionBadBlock(ctx, filterReq)
			if err != nil {
				t.Fatalf("failed to list execution bad blocks by execution implementation: %v", err)
			}

			if len(rsp.ExecutionBadBlocks) == 0 {
				t.Fatal("expected at least one execution bad block filtered by execution implementation")
			}
		})

		t.Run("by NodeVersion", func(t *testing.T) {
			filterReq := &pindexer.ListExecutionBadBlockRequest{
				NodeVersion: req.NodeVersion.Value,
			}

			rsp, err := index.ListExecutionBadBlock(ctx, filterReq)
			if err != nil {
				t.Fatalf("failed to list execution bad blocks by node version: %v", err)
			}

			if len(rsp.ExecutionBadBlocks) == 0 {
				t.Fatal("expected at least one execution bad block filtered by node version")
			}
		})
	})
}

func TestIndexerExecutionBadBlockDownloading(t *testing.T) {
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

	t.Run("Downloading", func(t *testing.T) {
		data := []byte("{\"key\": \"value\"}")

		location, err := index.Store().SaveExecutionBadBlock(ctx, &data, "data.json")
		if err != nil {
			t.Fatalf("failed to save execution bad block: %v", err)
		}

		req := createRandomExecutionBadBlockRequest()
		req.Location = wrapperspb.String(location)

		resp, err := index.CreateExecutionBadBlock(ctx, req)
		if err != nil {
			t.Fatalf("failed to create execution bad block: %v", err)
		}

		// List it
		rsp, err := index.ListExecutionBadBlock(ctx, &pindexer.ListExecutionBadBlockRequest{Id: resp.Id.Value})
		if err != nil {
			t.Fatalf("failed to get execution bad block: %v", err)
		}

		url, err := index.Store().GetExecutionBadBlockURL(ctx, rsp.ExecutionBadBlocks[0].Location.GetValue(), 60)
		if err != nil {
			t.Fatalf("failed to get execution bad block URL: %v", err)
		}

		if url == "" {
			t.Fatalf("expected URL to not be empty")
		}

		// Download it via http
		//nolint:gosec // This is a test
		if resp, err := http.Get(url); err != nil {
			t.Fatalf("failed to download execution bad block: %v", err)
		} else {
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Fatalf("expected status code to be 200, got %d", resp.StatusCode)
			}

			// Check the body contents
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("failed to read response body: %v", err)
			}

			if len(body) == 0 {
				t.Fatalf("expected body to not be empty")
			}
		}
	})
}
