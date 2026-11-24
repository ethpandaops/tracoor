package indexer

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/creasty/defaults"
	"github.com/ethpandaops/beacon/pkg/human"
	"github.com/ethpandaops/tracoor/pkg/proto/tracoor/indexer"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"gopkg.in/yaml.v3"
)

func countLoader(calls *atomic.Int64, count uint64) func(context.Context, *indexer.CountBeaconStateRequest) (*indexer.CountBeaconStateResponse, error) {
	return func(context.Context, *indexer.CountBeaconStateRequest) (*indexer.CountBeaconStateResponse, error) {
		calls.Add(1)

		return &indexer.CountBeaconStateResponse{Count: wrapperspb.UInt64(count)}, nil
	}
}

// waiters reports how many callers wait on the load in flight for req.
func waiters(t *testing.T, c *readCache, req proto.Message) int {
	t.Helper()

	key, err := readCacheKey(req)
	require.NoError(t, err)

	c.mu.Lock()
	defer c.mu.Unlock()

	if l, ok := c.loads[key]; ok {
		return l.waiters
	}

	return 0
}

func TestReadCacheServesRepeatsFromCache(t *testing.T) {
	c := newReadCache(time.Minute, 16, nil)

	var calls atomic.Int64

	req := &indexer.CountBeaconStateRequest{Network: "mainnet"}

	for range 3 {
		resp, err := cachedRead(context.Background(), c, req, countLoader(&calls, 7))
		require.NoError(t, err)
		require.Equal(t, uint64(7), resp.GetCount().GetValue())
	}

	require.Equal(t, int64(1), calls.Load())
}

func TestReadCacheKeysOnRequestAndType(t *testing.T) {
	c := newReadCache(time.Minute, 16, nil)

	var calls atomic.Int64

	ctx := context.Background()

	_, err := cachedRead(ctx, c, &indexer.CountBeaconStateRequest{Network: "mainnet"}, countLoader(&calls, 1))
	require.NoError(t, err)

	// A different filter value is a different answer.
	resp, err := cachedRead(ctx, c, &indexer.CountBeaconStateRequest{Network: "hoodi"}, countLoader(&calls, 2))
	require.NoError(t, err)
	require.Equal(t, uint64(2), resp.GetCount().GetValue())

	// Another method's request with the same field numbers and values encodes to the same
	// bytes, and must not collide either.
	block, err := cachedRead(ctx, c, &indexer.CountBeaconBlockRequest{Network: "mainnet"},
		func(context.Context, *indexer.CountBeaconBlockRequest) (*indexer.CountBeaconBlockResponse, error) {
			calls.Add(1)

			return &indexer.CountBeaconBlockResponse{Count: wrapperspb.UInt64(3)}, nil
		})
	require.NoError(t, err)
	require.Equal(t, uint64(3), block.GetCount().GetValue())

	require.Equal(t, int64(3), calls.Load())
}

func TestReadCacheKeyIgnoresFieldOrderAndRepeats(t *testing.T) {
	key := func(fields ...indexer.ListUniqueBeaconStateValuesRequest_Field) string {
		k, err := readCacheKey(&indexer.ListUniqueBeaconStateValuesRequest{Fields: fields, Network: "mainnet"})
		require.NoError(t, err)

		return k
	}

	node := indexer.ListUniqueBeaconStateValuesRequest_NODE
	version := indexer.ListUniqueBeaconStateValuesRequest_NODE_VERSION
	slot := indexer.ListUniqueBeaconStateValuesRequest_SLOT

	require.Equal(t, key(node, version), key(version, node))
	require.Equal(t, key(node, version), key(version, node, node, version))
	require.NotEqual(t, key(node, version), key(node, version, slot))

	// Normalizing the key leaves the caller's request alone.
	req := &indexer.ListUniqueBeaconStateValuesRequest{Fields: []indexer.ListUniqueBeaconStateValuesRequest_Field{version, node, node}}
	_, err := readCacheKey(req)
	require.NoError(t, err)
	require.Equal(t, []indexer.ListUniqueBeaconStateValuesRequest_Field{version, node, node}, req.GetFields())
}

func TestReadCacheExpires(t *testing.T) {
	c := newReadCache(20*time.Millisecond, 16, nil)

	var calls atomic.Int64

	req := &indexer.CountBeaconStateRequest{Network: "mainnet"}

	_, err := cachedRead(context.Background(), c, req, countLoader(&calls, 1))
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		_, err := cachedRead(context.Background(), c, req, countLoader(&calls, 1))

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
		_, err := cachedRead(context.Background(), c, req, failing)
		require.Error(t, err)
	}

	require.Equal(t, int64(2), calls.Load())
}

func TestReadCacheSharesConcurrentLoads(t *testing.T) {
	metrics := NewMetrics(metricsNamespace)
	c := newReadCache(time.Minute, 16, metrics)

	method := "CountBeaconState"
	counter := func(result string) float64 {
		return testutil.ToFloat64(metrics.readCache.WithLabelValues(method, result))
	}
	missBefore, sharedBefore, hitBefore := counter(readCacheMiss), counter(readCacheShared), counter(readCacheHit)

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

			resp, err := cachedRead(context.Background(), c, req, slow)
			if err == nil {
				results <- resp.GetCount().GetValue()
			}
		}()
	}

	// Every caller joins the one load before it is allowed to finish.
	require.Eventually(t, func() bool { return waiters(t, c, req) == callers }, time.Second, time.Millisecond)
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

	_, err := cachedRead(context.Background(), c, req, slow)
	require.NoError(t, err)

	// One caller started the load, the rest shared it, and the last one hit the cache.
	require.InDelta(t, 1, counter(readCacheMiss)-missBefore, 0)
	require.InDelta(t, callers-1, counter(readCacheShared)-sharedBefore, 0)
	require.InDelta(t, 1, counter(readCacheHit)-hitBefore, 0)
}

func TestReadCacheCallerCancelKeepsSharedLoad(t *testing.T) {
	c := newReadCache(time.Minute, 16, nil)

	var calls atomic.Int64

	release := make(chan struct{})

	slow := func(ctx context.Context, _ *indexer.CountBeaconStateRequest) (*indexer.CountBeaconStateResponse, error) {
		calls.Add(1)

		select {
		case <-release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}

		return &indexer.CountBeaconStateResponse{Count: wrapperspb.UInt64(5)}, nil
	}

	req := &indexer.CountBeaconStateRequest{Network: "mainnet"}

	starterCtx, cancel := context.WithCancel(context.Background())

	starter := make(chan error, 1)

	go func() {
		_, err := cachedRead(starterCtx, c, req, slow)
		starter <- err
	}()

	require.Eventually(t, func() bool { return waiters(t, c, req) == 1 }, time.Second, time.Millisecond)

	joiner := make(chan uint64, 1)

	go func() {
		resp, err := cachedRead(context.Background(), c, req, slow)
		if err == nil {
			joiner <- resp.GetCount().GetValue()
		}

		close(joiner)
	}()

	require.Eventually(t, func() bool { return waiters(t, c, req) == 2 }, time.Second, time.Millisecond)

	// The caller that started the load gives up; the one still waiting keeps it alive.
	cancel()
	require.ErrorIs(t, <-starter, context.Canceled)

	close(release)
	require.Equal(t, uint64(5), <-joiner)

	resp, err := cachedRead(context.Background(), c, req, slow)
	require.NoError(t, err)
	require.Equal(t, uint64(5), resp.GetCount().GetValue())
	require.Equal(t, int64(1), calls.Load())
}

func TestReadCacheLastCallerCancelAbortsLoad(t *testing.T) {
	c := newReadCache(time.Minute, 16, nil)

	var calls atomic.Int64

	aborted := make(chan struct{})

	blocking := func(ctx context.Context, _ *indexer.CountBeaconStateRequest) (*indexer.CountBeaconStateResponse, error) {
		calls.Add(1)
		<-ctx.Done()
		close(aborted)

		return nil, ctx.Err()
	}

	req := &indexer.CountBeaconStateRequest{Network: "mainnet"}

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)

	go func() {
		_, err := cachedRead(ctx, c, req, blocking)
		done <- err
	}()

	require.Eventually(t, func() bool { return calls.Load() == 1 }, time.Second, time.Millisecond)

	// With nobody left waiting the query is cancelled, as an uncached request's would be.
	cancel()
	require.ErrorIs(t, <-done, context.Canceled)

	select {
	case <-aborted:
	case <-time.After(time.Second):
		t.Fatal("load kept running after its last caller left")
	}

	// The next caller starts a fresh load rather than joining the cancelled one.
	var fresh atomic.Int64

	resp, err := cachedRead(context.Background(), c, req, countLoader(&fresh, 4))
	require.NoError(t, err)
	require.Equal(t, uint64(4), resp.GetCount().GetValue())
	require.Equal(t, int64(1), fresh.Load())
}

func TestReadCacheReturnsClones(t *testing.T) {
	c := newReadCache(time.Minute, 16, nil)

	load := func(context.Context, *indexer.ListUniqueBeaconStateValuesRequest) (*indexer.ListUniqueBeaconStateValuesResponse, error) {
		return &indexer.ListUniqueBeaconStateValuesResponse{NodeVersion: []string{"v1"}}, nil
	}

	req := &indexer.ListUniqueBeaconStateValuesRequest{Network: "mainnet"}

	first, err := cachedRead(context.Background(), c, req, load)
	require.NoError(t, err)

	first.NodeVersion[0] = "mutated"

	second, err := cachedRead(context.Background(), c, req, load)
	require.NoError(t, err)
	require.Equal(t, []string{"v1"}, second.GetNodeVersion())
}

func TestReadCacheDisabled(t *testing.T) {
	require.Nil(t, newReadCache(0, 16, nil))
	require.Nil(t, newReadCache(time.Minute, 0, nil))

	var calls atomic.Int64

	req := &indexer.CountBeaconStateRequest{Network: "mainnet"}

	for range 2 {
		_, err := cachedRead(context.Background(), nil, req, countLoader(&calls, 1))
		require.NoError(t, err)
	}

	require.Equal(t, int64(2), calls.Load())
}

// TestReadCacheDisabledSurvivesDefaults loads the config the way the server does: defaults,
// then the YAML, then defaults again in CreateGRPCServices. The second pass turns an explicit
// zero TTL back into the default, which is why the off switch is a flag.
func TestReadCacheDisabledSurvivesDefaults(t *testing.T) {
	conf := &Config{}
	require.NoError(t, defaults.Set(conf))
	require.NoError(t, yaml.Unmarshal([]byte("readCache:\n  disabled: true\n  ttl: 0s\n"), conf))
	require.NoError(t, defaults.Set(conf))

	require.True(t, conf.ReadCache.Disabled)

	index, cleanup, err := NewMockIndexer(context.Background(), conf)
	require.NoError(t, err)

	t.Cleanup(func() { require.NoError(t, cleanup()) })

	require.Nil(t, index.reads)
}

// TestIndexerServesCountsAndValuesFromCache goes through the real handlers, so a wrapper
// wired to the wrong loader or left uncached shows up here.
func TestIndexerServesCountsAndValuesFromCache(t *testing.T) {
	ctx := context.Background()

	conf := &Config{ReadCache: ReadCacheConfig{TTL: human.Duration{Duration: time.Minute}, MaxEntries: 16}}

	index, cleanup, err := NewMockIndexer(ctx, conf)
	require.NoError(t, err)

	t.Cleanup(func() { require.NoError(t, cleanup()) })

	require.NotNil(t, index.reads)

	first := createRandomBeaconStateRequest()
	network := first.GetNetwork().GetValue()

	_, err = createBeaconState(ctx, index, first)
	require.NoError(t, err)

	count := &indexer.CountBeaconStateRequest{Network: network}

	rsp, err := index.CountBeaconState(ctx, count)
	require.NoError(t, err)
	require.Equal(t, uint64(1), rsp.GetCount().GetValue())

	values, err := index.ListUniqueBeaconStateValues(ctx, &indexer.ListUniqueBeaconStateValuesRequest{
		Fields: []indexer.ListUniqueBeaconStateValuesRequest_Field{
			indexer.ListUniqueBeaconStateValuesRequest_NODE,
			indexer.ListUniqueBeaconStateValuesRequest_NODE_VERSION,
		},
		Network: network,
	})
	require.NoError(t, err)
	require.Equal(t, []string{first.GetNode().GetValue()}, values.GetNode())

	second := createRandomBeaconStateRequest()
	second.Network = wrapperspb.String(network)

	_, err = createBeaconState(ctx, index, second)
	require.NoError(t, err)

	// Both answers come from the cache until the TTL runs out, including the dropdown asked
	// with its fields reordered and repeated.
	rsp, err = index.CountBeaconState(ctx, count)
	require.NoError(t, err)
	require.Equal(t, uint64(1), rsp.GetCount().GetValue())

	values, err = index.ListUniqueBeaconStateValues(ctx, &indexer.ListUniqueBeaconStateValuesRequest{
		Fields: []indexer.ListUniqueBeaconStateValuesRequest_Field{
			indexer.ListUniqueBeaconStateValuesRequest_NODE_VERSION,
			indexer.ListUniqueBeaconStateValuesRequest_NODE,
			indexer.ListUniqueBeaconStateValuesRequest_NODE,
		},
		Network: network,
	})
	require.NoError(t, err)
	require.Equal(t, []string{first.GetNode().GetValue()}, values.GetNode())

	// A different question is answered fresh, and sees the new row.
	rsp, err = index.CountBeaconState(ctx, &indexer.CountBeaconStateRequest{Network: network, Node: second.GetNode().GetValue()})
	require.NoError(t, err)
	require.Equal(t, uint64(1), rsp.GetCount().GetValue())
}
