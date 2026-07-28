package agent

import (
	"context"
	"sync"

	"golang.org/x/sync/semaphore"
)

// defaultMaxConcurrentFetches bounds how many fetches may be in flight at once
// across every agent in the process.
//
// Every fetch path reads a whole response into memory before compressing and
// storing it, so peak usage is this limit multiplied by the largest response.
// A single shared budget is deliberate: per-path budgets bound each path but
// not the total, which is how 10 states + 10 block traces + 10 bad blocks came
// to run concurrently and OOM a 6GB limit.
//
// Sizing from a devnet with 47 agents: block traces are the largest at ~168MB,
// then states at ~74MB and bad blocks at ~13MB, so 10 caps the worst case near
// 1.7GB regardless of which paths are active.
const defaultMaxConcurrentFetches = 10

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

var globalFetchLimiter fetchLimiter

// acquireFetchSlot blocks until a fetch slot is free or ctx is cancelled. The
// returned release function must be called once the fetched payload and
// anything derived from it are no longer referenced.
func (s *agent) acquireFetchSlot(ctx context.Context) (func(), error) {
	return globalFetchLimiter.acquire(
		ctx,
		s.Config.Ethereum.GetMaxConcurrentFetches(),
		defaultMaxConcurrentFetches,
	)
}
