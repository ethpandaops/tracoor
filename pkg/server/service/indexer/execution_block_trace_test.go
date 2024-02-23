//nolint:gosec // This is a test
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

func createRandomExecutionBlockTraceRequest() *pindexer.CreateExecutionBlockTraceRequest {
	return &pindexer.CreateExecutionBlockTraceRequest{
		Node:                    wrapperspb.String(generateRandomString(5)),
		FetchedAt:               timestamppb.Now(),
		BlockHash:               wrapperspb.String(generateRandomString(64)),
		BlockNumber:             wrapperspb.Int64(generateRandomInt64()),
		Location:                wrapperspb.String(generateRandomString(10)),
		Network:                 wrapperspb.String(generateRandomString(5)),
		ExecutionImplementation: wrapperspb.String(generateRandomString(15)),
		NodeVersion:             wrapperspb.String(generateRandomString(8)),
	}
}

func TestIndexerExecutionBlockTraceCount(t *testing.T) {
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
		_, err := index.CreateExecutionBlockTrace(ctx, createRandomExecutionBlockTraceRequest())
		if err != nil {
			t.Fatalf("failed to create execution block trace: %v", err)
		}

		rsp, err := index.CountExecutionBlockTrace(ctx, &pindexer.CountExecutionBlockTraceRequest{})
		if err != nil {
			t.Fatalf("failed to count execution block trace: %v", err)
		}

		if rsp.Count.Value != uint64(1) {
			t.Fatalf("expected 1 execution block trace, got %d", rsp.Count.Value)
		}
	})
}

func TestIndexerExecutionBlockTrace(t *testing.T) {
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
		_, err := index.CreateExecutionBlockTrace(ctx, createRandomExecutionBlockTraceRequest())
		if err != nil {
			t.Fatalf("failed to create execution block trace: %v", err)
		}
	})

	t.Run("Creating returns a valid ID", func(t *testing.T) {
		rsp, err := index.CreateExecutionBlockTrace(ctx, createRandomExecutionBlockTraceRequest())
		if err != nil {
			t.Fatalf("failed to create execution block trace: %v", err)
		}

		if rsp.Id == nil {
			t.Fatalf("expected ID to not be empty")
		}
	})

	t.Run("Handles duplicates", func(t *testing.T) {
		req := createRandomExecutionBlockTraceRequest()
		rsp, err := index.CreateExecutionBlockTrace(ctx, req)
		if err != nil {
			t.Fatalf("failed to create execution block trace: %v", err)
		}

		if rsp.Id == nil {
			t.Fatalf("expected ID to not be empty")
		}

		_, err = index.CreateExecutionBlockTrace(ctx, req)
		if err != nil && err.Error() != "execution block trace already exists" {
			t.Fatal("expected error to be 'execution block trace already exists'")
		}
	})

	t.Run("Basic Listing", func(t *testing.T) {
		req := createRandomExecutionBlockTraceRequest()

		resp, err := index.CreateExecutionBlockTrace(ctx, req)
		if err != nil {
			t.Fatalf("failed to create execution block trace: %v", err)
		}

		rsp, err := index.ListExecutionBlockTrace(ctx, &pindexer.ListExecutionBlockTraceRequest{Id: resp.Id.Value})
		if err != nil {
			t.Fatalf("failed to get execution block trace: %v", err)
		}

		if len(rsp.ExecutionBlockTraces) != 1 {
			t.Fatalf("expected 1 execution block trace, got %d", len(rsp.ExecutionBlockTraces))
		}
	})

	t.Run("Can list by filters", func(t *testing.T) {
		req := createRandomExecutionBlockTraceRequest()

		_, err := index.CreateExecutionBlockTrace(ctx, req)
		if err != nil {
			t.Fatalf("failed to create execution block trace: %v", err)
		}

		t.Run("by Node", func(t *testing.T) {
			filterReq := &pindexer.ListExecutionBlockTraceRequest{
				Node: req.Node.Value,
			}

			rsp, err := index.ListExecutionBlockTrace(ctx, filterReq)
			if err != nil {
				t.Fatalf("failed to list execution block traces by node: %v", err)
			}

			if len(rsp.ExecutionBlockTraces) == 0 {
				t.Fatal("expected at least one execution block trace filtered by node")
			}
		})

		t.Run("by BlockNumber", func(t *testing.T) {
			filterReq := &pindexer.ListExecutionBlockTraceRequest{
				BlockNumber: req.BlockNumber.Value,
			}

			rsp, err := index.ListExecutionBlockTrace(ctx, filterReq)
			if err != nil {
				t.Fatalf("failed to list execution block traces by block number: %v", err)
			}

			if len(rsp.ExecutionBlockTraces) == 0 {
				t.Fatal("expected at least one execution block trace filtered by block number")
			}
		})

		t.Run("by BlockHash", func(t *testing.T) {
			filterReq := &pindexer.ListExecutionBlockTraceRequest{
				BlockHash: req.BlockHash.Value,
			}

			rsp, err := index.ListExecutionBlockTrace(ctx, filterReq)
			if err != nil {
				t.Fatalf("failed to list execution block traces by block hash: %v", err)
			}

			if len(rsp.ExecutionBlockTraces) == 0 {
				t.Fatal("expected at least one execution block trace filtered by block hash")
			}
		})

		t.Run("by Location", func(t *testing.T) {
			filterReq := &pindexer.ListExecutionBlockTraceRequest{
				Location: req.Location.Value,
			}

			rsp, err := index.ListExecutionBlockTrace(ctx, filterReq)
			if err != nil {
				t.Fatalf("failed to list execution block traces by location: %v", err)
			}

			if len(rsp.ExecutionBlockTraces) == 0 {
				t.Fatal("expected at least one execution block trace filtered by location")
			}
		})

		t.Run("by Network", func(t *testing.T) {
			filterReq := &pindexer.ListExecutionBlockTraceRequest{
				Network: req.Network.Value,
			}

			rsp, err := index.ListExecutionBlockTrace(ctx, filterReq)
			if err != nil {
				t.Fatalf("failed to list execution block traces by network: %v", err)
			}

			if len(rsp.ExecutionBlockTraces) == 0 {
				t.Fatal("expected at least one execution block trace filtered by network")
			}
		})

		t.Run("by ExecutionImplementation", func(t *testing.T) {
			filterReq := &pindexer.ListExecutionBlockTraceRequest{
				ExecutionImplementation: req.ExecutionImplementation.Value,
			}

			rsp, err := index.ListExecutionBlockTrace(ctx, filterReq)
			if err != nil {
				t.Fatalf("failed to list execution block traces by execution implementation: %v", err)
			}

			if len(rsp.ExecutionBlockTraces) == 0 {
				t.Fatal("expected at least one execution block trace filtered by execution implementation")
			}
		})

		t.Run("by NodeVersion", func(t *testing.T) {
			filterReq := &pindexer.ListExecutionBlockTraceRequest{
				NodeVersion: req.NodeVersion.Value,
			}

			rsp, err := index.ListExecutionBlockTrace(ctx, filterReq)
			if err != nil {
				t.Fatalf("failed to list execution block traces by node version: %v", err)
			}

			if len(rsp.ExecutionBlockTraces) == 0 {
				t.Fatal("expected at least one execution block trace filtered by node version")
			}
		})
	})
}

func TestIndexerExecutionBlockTraceDownloading(t *testing.T) {
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

		location, err := index.Store().SaveExecutionBlockTrace(ctx, &data, "data.json")
		if err != nil {
			t.Fatalf("failed to save execution block trace: %v", err)
		}

		req := createRandomExecutionBlockTraceRequest()
		req.Location = wrapperspb.String(location)

		resp, err := index.CreateExecutionBlockTrace(ctx, req)
		if err != nil {
			t.Fatalf("failed to create execution block trace: %v", err)
		}

		// List it
		rsp, err := index.ListExecutionBlockTrace(ctx, &pindexer.ListExecutionBlockTraceRequest{Id: resp.Id.Value})
		if err != nil {
			t.Fatalf("failed to get execution block trace: %v", err)
		}

		url, err := index.Store().GetExecutionBlockTraceURL(ctx, rsp.ExecutionBlockTraces[0].Location.GetValue(), 60)
		if err != nil {
			t.Fatalf("failed to get execution block trace URL: %v", err)
		}

		if url == "" {
			t.Fatalf("expected URL to not be empty")
		}

		// Download it via http
		if resp, err := http.Get(url); err != nil {
			t.Fatalf("failed to download execution block trace: %v", err)
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
