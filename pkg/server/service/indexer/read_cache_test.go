package indexer

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethpandaops/tracoor/pkg/proto/tracoor/indexer"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func countLoader(calls *atomic.Int64, count uint64) func(context.Context, *indexer.CountBeaconStateRequest) (*indexer.CountBeaconStateResponse, error) {
	return func(context.Context, *indexer.CountBeaconStateRequest) (*indexer.CountBeaconStateResponse, error) {
		calls.Add(1)

		return &indexer.CountBeaconStateResponse{Count: wrapperspb.UInt64(count)}, nil
	}
}

func TestReadCacheServesRepeatsFromCache(t *testing.T) {
	c := newReadCache(time.Minute, 16, nil)

	var calls atomic.Int64

	req := &indexer.CountBeaconStateRequest{Network: "mainnet"}

	for range 3 {
		resp, err := cachedRead(context.Background(), c, "CountBeaconState", req, countLoader(&calls, 7))
		require.NoError(t, err)
		require.Equal(t, uint64(7), resp.GetCount().GetValue())
	}

	require.Equal(t, int64(1), calls.Load())
}

func TestReadCacheKeysOnRequestAndMethod(t *testing.T) {
	c := newReadCache(time.Minute, 16, nil)

	var calls atomic.Int64

	ctx := context.Background()

	_, err := cachedRead(ctx, c, "CountBeaconState", &indexer.CountBeaconStateRequest{Network: "mainnet"}, countLoader(&calls, 1))
	require.NoError(t, err)

	// A different filter value is a different answer.
	resp, err := cachedRead(ctx, c, "CountBeaconState", &indexer.CountBeaconStateRequest{Network: "hoodi"}, countLoader(&calls, 2))
	require.NoError(t, err)
	require.Equal(t, uint64(2), resp.GetCount().GetValue())

	// The same request bytes under another method must not collide either.
	resp, err = cachedRead(ctx, c, "CountBeaconBlock", &indexer.CountBeaconStateRequest{Network: "mainnet"}, countLoader(&calls, 3))
	require.NoError(t, err)
	require.Equal(t, uint64(3), resp.GetCount().GetValue())

	require.Equal(t, int64(3), calls.Load())
}

func TestReadCacheExpires(t *testing.T) {
	c := newReadCache(20*time.Millisecond, 16, nil)

	var calls atomic.Int64

	req := &indexer.CountBeaconStateRequest{Network: "mainnet"}

	_, err := cachedRead(context.Background(), c, "CountBeaconState", req, countLoader(&calls, 1))
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		_, err := cachedRead(context.Background(), c, "CountBeaconState", req, countLoader(&calls, 1))

		return err == nil && calls.Load() == 2
	}, time.Second, 10*time.Millisecond)
}

func TestReadCacheDoesNotCacheErrors(t *testing.T) {
	c := newReadCache(time.Minute, 16, nil)

	var calls atomic.Int64

	failing := func(context.Context, *indexer.CountBeaconStateRequest) (*indexer.CountBeaconStateResponse, error) {
		calls.Add(1)

		return nil, errors.New("database unavailable")
	}

	req := &indexer.CountBeaconStateRequest{Network: "mainnet"}

	for range 2 {
		_, err := cachedRead(context.Background(), c, "CountBeaconState", req, failing)
		require.Error(t, err)
	}

	require.Equal(t, int64(2), calls.Load())
}

func TestReadCacheSharesConcurrentLoads(t *testing.T) {
	c := newReadCache(time.Minute, 16, nil)

	var calls atomic.Int64

	release := make(chan struct{})

	slow := func(context.Context, *indexer.CountBeaconStateRequest) (*indexer.CountBeaconStateResponse, error) {
		calls.Add(1)
		<-release

		return &indexer.CountBeaconStateResponse{Count: wrapperspb.UInt64(9)}, nil
	}

	req := &indexer.CountBeaconStateRequest{Network: "mainnet"}

	const callers = 8

	var wg sync.WaitGroup

	results := make(chan uint64, callers)

	for range callers {
		wg.Add(1)

		go func() {
			defer wg.Done()

			resp, err := cachedRead(context.Background(), c, "CountBeaconState", req, slow)
			if err == nil {
				results <- resp.GetCount().GetValue()
			}
		}()
	}

	// Give every caller time to join the in-flight load before it completes.
	time.Sleep(50 * time.Millisecond)
	close(release)
	wg.Wait()
	close(results)

	got := 0

	for v := range results {
		require.Equal(t, uint64(9), v)

		got++
	}

	require.Equal(t, callers, got)
	require.Equal(t, int64(1), calls.Load())
}

func TestReadCacheCallerCancelDoesNotAbortSharedLoad(t *testing.T) {
	c := newReadCache(time.Minute, 16, nil)

	var calls atomic.Int64

	release := make(chan struct{})

	slow := func(ctx context.Context, _ *indexer.CountBeaconStateRequest) (*indexer.CountBeaconStateResponse, error) {
		calls.Add(1)
		<-release

		// The load's own context must still be live after its starter gave up.
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		return &indexer.CountBeaconStateResponse{Count: wrapperspb.UInt64(5)}, nil
	}

	req := &indexer.CountBeaconStateRequest{Network: "mainnet"}

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)

	go func() {
		_, err := cachedRead(ctx, c, "CountBeaconState", req, slow)
		done <- err
	}()

	require.Eventually(t, func() bool { return calls.Load() == 1 }, time.Second, 5*time.Millisecond)

	cancel()
	require.ErrorIs(t, <-done, context.Canceled)

	close(release)

	// The abandoned load still lands in the cache for the next caller.
	require.Eventually(t, func() bool {
		resp, err := cachedRead(context.Background(), c, "CountBeaconState", req, slow)

		return err == nil && resp.GetCount().GetValue() == 5 && calls.Load() == 1
	}, time.Second, 5*time.Millisecond)
}

func TestReadCacheReturnsClones(t *testing.T) {
	c := newReadCache(time.Minute, 16, nil)

	load := func(context.Context, *indexer.ListUniqueBeaconStateValuesRequest) (*indexer.ListUniqueBeaconStateValuesResponse, error) {
		return &indexer.ListUniqueBeaconStateValuesResponse{NodeVersion: []string{"v1"}}, nil
	}

	req := &indexer.ListUniqueBeaconStateValuesRequest{Network: "mainnet"}

	first, err := cachedRead(context.Background(), c, "ListUniqueBeaconStateValues", req, load)
	require.NoError(t, err)

	first.NodeVersion[0] = "mutated"

	second, err := cachedRead(context.Background(), c, "ListUniqueBeaconStateValues", req, load)
	require.NoError(t, err)
	require.Equal(t, []string{"v1"}, second.GetNodeVersion())
}

func TestReadCacheDisabled(t *testing.T) {
	require.Nil(t, newReadCache(0, 16, nil))
	require.Nil(t, newReadCache(time.Minute, 0, nil))

	var calls atomic.Int64

	req := &indexer.CountBeaconStateRequest{Network: "mainnet"}

	for range 2 {
		_, err := cachedRead(context.Background(), nil, "CountBeaconState", req, countLoader(&calls, 1))
		require.NoError(t, err)
	}

	require.Equal(t, int64(2), calls.Load())
}
