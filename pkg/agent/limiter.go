package agent

import (
	"context"
	"sync"

	"golang.org/x/sync/semaphore"
)

// Fetch concurrency defaults. Both paths read a whole response into memory, so
// peak usage scales with how many are in flight rather than with the response
// size alone.
//
// In `single` mode one process runs an agent per node and every agent reacts to
// the same block event, so without a bound the peak scales with the node count:
// on a devnet with ~100MB states, 47 agents peaked near 6GB and were OOM killed.
// Bounding in-flight fetches makes peak memory a function of these limits rather
// than of how many nodes are configured.
const (
	defaultMaxConcurrentBeaconStateFetches = 10
	// Bad block responses carry every bad block the node still holds, so they can
	// be large and are decoded rather than streamed. Kept lower than the state
	// limit because the decoded slice is retained for the whole indexing pass.
	defaultMaxConcurrentExecutionBadBlockFetches = 4
)

// fetchLimiter is a lazily sized process-wide concurrency budget.
//
// The budget is package-level so that agents constructed independently still
// share it; in `single` mode they are separate agent instances in the same
// process. The first caller fixes the size, so mixed per-agent limits are not
// supported - the value from whichever agent starts first wins.
type fetchLimiter struct {
	once sync.Once
	sem  *semaphore.Weighted
}

// acquire blocks until a slot is free or ctx is cancelled, returning the release
// function. def is used when limit is unset.
func (l *fetchLimiter) acquire(ctx context.Context, limit, def int) (func(), error) {
	l.once.Do(func() {
		if limit <= 0 {
			limit = def
		}

		l.sem = semaphore.NewWeighted(int64(limit))
	})

	if err := l.sem.Acquire(ctx, 1); err != nil {
		return nil, err
	}

	return func() { l.sem.Release(1) }, nil
}

var (
	beaconStateFetchLimiter       fetchLimiter
	executionBadBlockFetchLimiter fetchLimiter
)

// acquireBeaconStateFetch bounds concurrent beacon state fetches. The returned
// release function must be called once the state and its compressed copy are no
// longer referenced.
func (s *agent) acquireBeaconStateFetch(ctx context.Context) (func(), error) {
	return beaconStateFetchLimiter.acquire(
		ctx,
		s.Config.Ethereum.GetMaxConcurrentBeaconStateFetches(),
		defaultMaxConcurrentBeaconStateFetches,
	)
}

// acquireExecutionBadBlockFetch bounds concurrent bad block fetches. The
// returned release function must be called once the decoded blocks are no longer
// referenced, which is after indexing rather than after the fetch.
func (s *agent) acquireExecutionBadBlockFetch(ctx context.Context) (func(), error) {
	return executionBadBlockFetchLimiter.acquire(
		ctx,
		s.Config.Ethereum.GetMaxConcurrentExecutionBadBlockFetches(),
		defaultMaxConcurrentExecutionBadBlockFetches,
	)
}
