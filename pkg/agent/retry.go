package agent

import (
	"context"
	"time"

	"github.com/ethpandaops/ethwallclock"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/sirupsen/logrus"
)

// envelopeRevealSlots ends the block path's wait at the end of the next slot; later reveals arrive via the payload event.
const envelopeRevealSlots = 2

// backoff spaces out retries of a failure expected to clear with time, giving up after limit retries.
type backoff struct {
	base  time.Duration
	max   time.Duration
	limit int
}

var (
	// defaultRevealBackoff paces the wait for an unrevealed payload; the reveal deadline usually ends it first.
	defaultRevealBackoff = backoff{base: time.Second, max: 4 * time.Second, limit: 10}

	// defaultOutageBackoff paces retries through a store or indexer outage, about 90s in total.
	defaultOutageBackoff = backoff{base: 2 * time.Second, max: 30 * time.Second, limit: 6}
)

// after reports the delay before the next retry, or false once the retries are spent.
func (b backoff) after(retries int) (time.Duration, bool) {
	if retries >= b.limit {
		return 0, false
	}

	return min(b.base<<retries, b.max), true
}

// retryPolicy is what runQueueItem needs to set a failed item aside instead of dropping it.
type retryPolicy struct {
	// retries is how many times the item has already been set aside.
	retries int

	// revealBy is when an unavailable artifact stops being worth waiting for; zero means absence is final.
	revealBy time.Time
}

// untilRevealed reports the delay before looking for an unrevealed artifact again, or false once its window closed.
func (p retryPolicy) untilRevealed(pace backoff, now time.Time) (time.Duration, bool) {
	remaining := p.revealBy.Sub(now)
	if p.revealBy.IsZero() || remaining <= 0 {
		return 0, false
	}

	delay, ok := pace.after(p.retries)
	if !ok {
		return 0, false
	}

	return min(delay, remaining), true
}

// revealDeadline is when the block path stops looking for the payload of the block at slot.
func revealDeadline(wallclock *ethwallclock.EthereumBeaconChain, slot phase0.Slot) time.Time {
	if wallclock == nil {
		return time.Time{}
	}

	end := wallclock.Slots().FromNumber(uint64(slot) + envelopeRevealSlots)

	return end.TimeWindow().Start()
}

func (s *agent) retryItem(kind Queue, logCtx logrus.FieldLogger, reason string, err error, delay time.Duration) time.Duration {
	s.metrics.IncrementItemRetried(kind, s.Config.Name, reason)

	logCtx.
		WithError(err).
		WithField("reason", reason).
		WithField("retry_in", delay.String()).
		Debug("Retrying queue item later")

	return delay
}

// waitFor sleeps for d, reporting false if ctx ended first.
func waitFor(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

// retryLater resubmits an item after delay, on a tracked worker that shutdown cancels rather than waits out.
func retryLater[T queued](ctx context.Context, s *agent, kind Queue, queue chan<- T, item T, delay time.Duration) {
	s.workers.Go(func() {
		if !waitFor(ctx, delay) {
			return
		}

		item.markRetry()

		resubmit(ctx, s, kind, queue, item)
	})
}

// resubmit requeues an item under a fresh claim; a collision means another copy is already doing the work.
func resubmit[T queued](ctx context.Context, s *agent, kind Queue, queue chan<- T, item T) {
	if !s.allowArtifact(kind) || !s.claimQueueItem(kind, item.claimKey(), false) {
		return
	}

	if !enqueue(ctx, queue, item) {
		s.releaseQueueItem(item.claimKey())
	}
}
