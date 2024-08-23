//nolint:gocyclo // Only used in tests
package indexer

import (
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/ethpandaops/tracoor/pkg/compression"
	pindexer "github.com/ethpandaops/tracoor/pkg/proto/tracoor/indexer"
	"github.com/ethpandaops/tracoor/pkg/store"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func createRandomBeaconBadBlockRequest() *pindexer.CreateBeaconBadBlockRequest {
	return &pindexer.CreateBeaconBadBlockRequest{
		Node:                 wrapperspb.String(generateRandomString(5)),
		Slot:                 wrapperspb.UInt64(uint64(generateRandomInt64())),
		Epoch:                wrapperspb.UInt64(uint64(generateRandomInt64())),
		BlockRoot:            wrapperspb.String(generateRandomString(32)),
		FetchedAt:            timestamppb.Now(),
		BeaconImplementation: wrapperspb.String(generateRandomString(15)),
		NodeVersion:          wrapperspb.String(generateRandomString(8)),
		Location:             wrapperspb.String(generateRandomString(10)),
		Network:              wrapperspb.String(generateRandomString(5)),
	}
}

func TestIndexerBeaconBadBlockCount(t *testing.T) {
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
		_, err := index.CreateBeaconBadBlock(ctx, createRandomBeaconBadBlockRequest())
		if err != nil {
			t.Fatalf("failed to create beacon state: %v", err)
		}

		rsp, err := index.CountBeaconBadBlock(ctx, &pindexer.CountBeaconBadBlockRequest{})
		if err != nil {
			t.Fatalf("failed to count beacon state: %v", err)
		}

		if rsp.Count.Value != uint64(1) {
			t.Fatalf("expected 1 beacon state, got %d", rsp.Count.Value)
		}
	})
}

func TestIndexerBeaconBadBlockDownloading(t *testing.T) {
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

		compressor := compression.NewCompressor()

		compressedData, err := compressor.Compress(&data, compression.Gzip)
		if err != nil {
			t.Fatalf("failed to compress data: %v", err)
		}

		location, err := index.Store().SaveBeaconBadBlock(ctx, &store.SaveParams{
			Data:            &compressedData,
			Location:        "data.json",
			ContentEncoding: compression.Gzip.ContentEncoding,
		})
		if err != nil {
			t.Fatalf("failed to save beacon state: %v", err)
		}

		req := createRandomBeaconBadBlockRequest()
		req.Location = wrapperspb.String(location)

		resp, err := index.CreateBeaconBadBlock(ctx, req)
		if err != nil {
			t.Fatalf("failed to create beacon state: %v", err)
		}

		// List it
		rsp, err := index.ListBeaconBadBlock(ctx, &pindexer.ListBeaconBadBlockRequest{Id: resp.Id.Value})
		if err != nil {
			t.Fatalf("failed to get beacon state: %v", err)
		}

		url, err := index.Store().GetBeaconBadBlockURL(ctx, &store.GetURLParams{
			Location:        rsp.BeaconBadBlocks[0].Location.GetValue(),
			Expiry:          60,
			ContentEncoding: rsp.BeaconBadBlocks[0].ContentEncoding.GetValue(),
		})
		if err != nil {
			t.Fatalf("failed to get beacon state URL: %v", err)
		}

		if url == "" {
			t.Fatalf("expected URL to not be empty")
		}

		// Download it via http
		//nolint:gosec // This is a test
		if resp, err := http.Get(url); err != nil {
			t.Fatalf("failed to download beacon state: %v", err)
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

func TestIndexerBeaconBadBlock(t *testing.T) {
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
		_, err := index.CreateBeaconBadBlock(ctx, createRandomBeaconBadBlockRequest())
		if err != nil {
			t.Fatalf("failed to create beacon state: %v", err)
		}
	})

	t.Run("Creating returns a valid ID", func(t *testing.T) {
		rsp, err := index.CreateBeaconBadBlock(ctx, createRandomBeaconBadBlockRequest())
		if err != nil {
			t.Fatalf("failed to create beacon state: %v", err)
		}

		if rsp.Id == nil {
			t.Fatalf("expected ID to not be empty")
		}
	})

	t.Run("Handles duplicates", func(t *testing.T) {
		req := createRandomBeaconBadBlockRequest()

		rsp, err := index.CreateBeaconBadBlock(ctx, req)
		if err != nil {
			t.Fatalf("failed to create beacon state: %v", err)
		}

		if rsp.Id == nil {
			t.Fatalf("expected ID to not be empty")
		}

		_, err = index.CreateBeaconBadBlock(ctx, req)
		if err != nil && err.Error() != "beacon state already exists" {
			t.Fatal("expected error to be 'beacon state already exists'")
		}
	})

	t.Run("Basic Listing", func(t *testing.T) {
		req := createRandomBeaconBadBlockRequest()

		resp, err := index.CreateBeaconBadBlock(ctx, req)
		if err != nil {
			t.Fatalf("failed to create beacon state: %v", err)
		}

		rsp, err := index.ListBeaconBadBlock(ctx, &pindexer.ListBeaconBadBlockRequest{Id: resp.Id.Value})
		if err != nil {
			t.Fatalf("failed to get beacon state: %v", err)
		}

		if len(rsp.BeaconBadBlocks) != 1 {
			t.Fatalf("expected 1 beacon state, got %d", len(rsp.BeaconBadBlocks))
		}
	})

	t.Run("Can list by filters", func(t *testing.T) {
		req := createRandomBeaconBadBlockRequest()

		resp, err := index.CreateBeaconBadBlock(ctx, req)
		if err != nil {
			t.Fatalf("failed to create beacon state: %v", err)
		}

		t.Run("by Node", func(t *testing.T) {
			filterReq := &pindexer.ListBeaconBadBlockRequest{
				Node: req.Node.Value,
			}

			rsp, err := index.ListBeaconBadBlock(ctx, filterReq)
			if err != nil {
				t.Fatalf("failed to list beacon states by node: %v", err)
			}

			if len(rsp.BeaconBadBlocks) == 0 {
				t.Fatal("expected at least one beacon state filtered by node")
			}
		})

		t.Run("by Slot", func(t *testing.T) {
			filterReq := &pindexer.ListBeaconBadBlockRequest{
				Slot: req.Slot.Value,
			}

			rsp, err := index.ListBeaconBadBlock(ctx, filterReq)
			if err != nil {
				t.Fatalf("failed to list beacon states by slot: %v", err)
			}

			if len(rsp.BeaconBadBlocks) == 0 {
				t.Fatal("expected at least one beacon state filtered by slot")
			}
		})

		t.Run("by Epoch", func(t *testing.T) {
			filterReq := &pindexer.ListBeaconBadBlockRequest{
				Epoch: req.Epoch.Value,
			}

			rsp, err := index.ListBeaconBadBlock(ctx, filterReq)
			if err != nil {
				t.Fatalf("failed to list beacon states by epoch: %v", err)
			}

			if len(rsp.BeaconBadBlocks) == 0 {
				t.Fatal("expected at least one beacon state filtered by epoch")
			}
		})

		t.Run("by BlockRoot", func(t *testing.T) {
			filterReq := &pindexer.ListBeaconBadBlockRequest{
				BlockRoot: req.BlockRoot.Value,
			}

			rsp, err := index.ListBeaconBadBlock(ctx, filterReq)
			if err != nil {
				t.Fatalf("failed to list beacon states by state root: %v", err)
			}

			if len(rsp.BeaconBadBlocks) == 0 {
				t.Fatal("expected at least one beacon state filtered by state root")
			}
		})

		t.Run("by Network", func(t *testing.T) {
			filterReq := &pindexer.ListBeaconBadBlockRequest{
				Network: req.Network.Value,
			}

			rsp, err := index.ListBeaconBadBlock(ctx, filterReq)
			if err != nil {
				t.Fatalf("failed to list beacon states by network: %v", err)
			}

			if len(rsp.BeaconBadBlocks) == 0 {
				t.Fatal("expected at least one beacon state filtered by network")
			}
		})

		t.Run("by BeaconImplementation", func(t *testing.T) {
			filterReq := &pindexer.ListBeaconBadBlockRequest{
				BeaconImplementation: req.BeaconImplementation.Value,
			}

			rsp, err := index.ListBeaconBadBlock(ctx, filterReq)
			if err != nil {
				t.Fatalf("failed to list beacon states by beacon implementation: %v", err)
			}

			if len(rsp.BeaconBadBlocks) == 0 {
				t.Fatal("expected at least one beacon state filtered by beacon implementation")
			}
		})

		t.Run("by NodeVersion", func(t *testing.T) {
			filterReq := &pindexer.ListBeaconBadBlockRequest{
				NodeVersion: req.NodeVersion.Value,
			}

			rsp, err := index.ListBeaconBadBlock(ctx, filterReq)
			if err != nil {
				t.Fatalf("failed to list beacon states by node version: %v", err)
			}

			if len(rsp.BeaconBadBlocks) == 0 {
				t.Fatal("expected at least one beacon state filtered by node version")
			}
		})

		t.Run("by Location", func(t *testing.T) {
			filterReq := &pindexer.ListBeaconBadBlockRequest{
				Location: req.Location.Value,
			}

			rsp, err := index.ListBeaconBadBlock(ctx, filterReq)
			if err != nil {
				t.Fatalf("failed to list beacon states by location: %v", err)
			}

			if len(rsp.BeaconBadBlocks) == 0 {
				t.Fatal("expected at least one beacon state filtered by location")
			}
		})

		t.Run("by Before", func(t *testing.T) {
			filterReq := &pindexer.ListBeaconBadBlockRequest{
				Before: timestamppb.New(req.FetchedAt.AsTime().Add(time.Minute)),
			}

			rsp, err := index.ListBeaconBadBlock(ctx, filterReq)
			if err != nil {
				t.Fatalf("failed to list beacon states by fetched at: %v", err)
			}

			if len(rsp.BeaconBadBlocks) == 0 {
				t.Fatal("expected at least one beacon state filtered by fetched at")
			}
		})

		t.Run("by After", func(t *testing.T) {
			filterReq := &pindexer.ListBeaconBadBlockRequest{
				After: timestamppb.New(req.FetchedAt.AsTime().Add(-time.Minute)),
			}

			rsp, err := index.ListBeaconBadBlock(ctx, filterReq)
			if err != nil {
				t.Fatalf("failed to list beacon states by fetched at: %v", err)
			}

			if len(rsp.BeaconBadBlocks) == 0 {
				t.Fatal("expected at least one beacon state filtered by fetched at")
			}
		})

		t.Run("by ID", func(t *testing.T) {
			filterReq := &pindexer.ListBeaconBadBlockRequest{
				Id: resp.Id.Value,
			}

			rsp, err := index.ListBeaconBadBlock(ctx, filterReq)
			if err != nil {
				t.Fatalf("failed to list beacon states by ID: %v", err)
			}

			if len(rsp.BeaconBadBlocks) == 0 {
				t.Fatal("expected at least one beacon state filtered by ID")
			}
		})

		t.Run("by BlockRoot", func(t *testing.T) {
			filterReq := &pindexer.ListBeaconBadBlockRequest{
				BlockRoot: req.BlockRoot.Value,
			}

			rsp, err := index.ListBeaconBadBlock(ctx, filterReq)
			if err != nil {
				t.Fatalf("failed to list beacon states by state root: %v", err)
			}

			if len(rsp.BeaconBadBlocks) == 0 {
				t.Fatal("expected at least one beacon state filtered by state root")
			}
		})
	})
}
