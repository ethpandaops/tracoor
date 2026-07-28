package agent

import (
	"context"
	"sync"

	"golang.org/x/sync/semaphore"
)

// Both paths read a whole response into memory, so peak usage is the limit
// multiplied by the response size. States are roughly an order of magnitude
// larger than bad block responses.
const (
	defaultMaxConcurrentBeaconStateFetches       = 10
	defaultMaxConcurrentExecutionBadBlockFetches = 10
)

// fetchLimiter is a lazily sized process-wide concurrency budget. It is
// package-level because `single` mode runs an agent per node in one process and
// constructs each independently, so a per-agent budget would bound nothing. The
// first caller fixes the size.
type fetchLimiter struct {
	once sync.Once
	sem  *semaphore.Weighted
}

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

func (s *agent) acquireBeaconStateFetch(ctx context.Context) (func(), error) {
	return beaconStateFetchLimiter.acquire(
		ctx,
		s.Config.Ethereum.GetMaxConcurrentBeaconStateFetches(),
		defaultMaxConcurrentBeaconStateFetches,
	)
}

func (s *agent) acquireExecutionBadBlockFetch(ctx context.Context) (func(), error) {
	return executionBadBlockFetchLimiter.acquire(
		ctx,
		s.Config.Ethereum.GetMaxConcurrentExecutionBadBlockFetches(),
		defaultMaxConcurrentExecutionBadBlockFetches,
	)
}
