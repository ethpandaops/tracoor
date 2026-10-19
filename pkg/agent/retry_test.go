package agent

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/ethpandaops/beacon/pkg/beacon/state"
	"github.com/ethpandaops/ethwallclock"
	"github.com/ethpandaops/go-eth2-client/spec"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// uniqueName keeps metric series apart across repeated runs of the process-wide metrics.
func uniqueName(name string) string {
	return name + "-" + uuid.NewString()[:8]
}

func TestBackoffDoublesUpToItsCeilingAndLimit(t *testing.T) {
	pace := backoff{base: time.Second, max: 4 * time.Second, limit: 4}

	var delays []time.Duration

	for retries := 0; ; retries++ {
		delay, ok := pace.after(retries)
		if !ok {
			break
		}

		delays = append(delays, delay)
	}

	require.Equal(t, []time.Duration{time.Second, 2 * time.Second, 4 * time.Second, 4 * time.Second}, delays)
}

func TestUntilRevealedStopsAtTheDeadline(t *testing.T) {
	pace := backoff{base: time.Second, max: 4 * time.Second, limit: 10}
	now := time.Now()

	delay, ok := retryPolicy{retries: 3, revealBy: now.Add(10 * time.Second)}.untilRevealed(pace, now)
	require.True(t, ok)
	require.Equal(t, 4*time.Second, delay)

	delay, ok = retryPolicy{retries: 3, revealBy: now.Add(time.Second)}.untilRevealed(pace, now)
	require.True(t, ok)
	require.Equal(t, time.Second, delay, "the last look lands on the deadline rather than past it")

	_, ok = retryPolicy{revealBy: now}.untilRevealed(pace, now)
	require.False(t, ok, "a closed window is not waited on")

	_, ok = retryPolicy{}.untilRevealed(pace, now)
	require.False(t, ok, "without a deadline absence is final")

	_, ok = retryPolicy{retries: pace.limit, revealBy: now.Add(time.Hour)}.untilRevealed(pace, now)
	require.False(t, ok, "the retries are bounded even when the deadline is not")
}

func TestRevealDeadlineIsTheEndOfTheNextSlot(t *testing.T) {
	genesis := time.Unix(1_700_000_000, 0)

	// Not stopped: Stop blocks until the current epoch ends.
	wallclock := ethwallclock.NewEthereumBeaconChain(genesis, 12*time.Second, 32)

	require.Equal(t, genesis.Add(12*12*time.Second), revealDeadline(wallclock, 10))
	require.True(t, revealDeadline(nil, 10).IsZero())
}

func TestEnvelopeForkScheduled(t *testing.T) {
	specWith := func(forks ...*state.ForkEpoch) *state.Spec {
		return &state.Spec{ForkEpochs: forks}
	}

	fulu := &state.ForkEpoch{Name: spec.DataVersionFulu, Epoch: 10}

	require.False(t, envelopeForkScheduled(nil))
	require.False(t, envelopeForkScheduled(specWith(fulu)), "a node that predates gloas names no envelope fork")
	require.False(t, envelopeForkScheduled(specWith(fulu, &state.ForkEpoch{Name: spec.DataVersionGloas, Epoch: farFutureEpoch})))
	require.True(t, envelopeForkScheduled(specWith(fulu, &state.ForkEpoch{Name: spec.DataVersionGloas, Epoch: 353024})))
	require.True(t, envelopeForkScheduled(specWith(&state.ForkEpoch{Name: spec.DataVersionHeze, Epoch: 0})))
}

func TestRunQueueItemSetsAnUnrevealedEnvelopeAside(t *testing.T) {
	s := newTestAgent(uniqueName("unrevealed"))

	calls := 0

	delay := s.runQueueItem(context.Background(), ExecutionPayloadEnvelopeQueue, s.log,
		retryPolicy{revealBy: time.Now().Add(time.Minute)},
		func(context.Context) error {
			calls++

			return fmt.Errorf("%w: not revealed yet", errItemNotAvailable)
		},
	)

	require.Equal(t, s.revealBackoff.base, delay)
	require.Equal(t, 1, calls, "a second look straight away would land in the same moment")
	require.Equal(t, float64(1), retriedCount(s, ExecutionPayloadEnvelopeQueue, dropReasonNotAvailable))
	require.Zero(t, droppedCount(s, ExecutionPayloadEnvelopeQueue, dropReasonNotAvailable))
	require.True(t, s.breaker.Allow(ExecutionPayloadEnvelopeQueue))
}

func TestRunQueueItemDropsAnEnvelopeOnceItsRevealWindowCloses(t *testing.T) {
	s := newTestAgent(uniqueName("window-closed"))

	calls := 0

	delay := s.runQueueItem(context.Background(), ExecutionPayloadEnvelopeQueue, s.log,
		retryPolicy{retries: 3, revealBy: time.Now().Add(-time.Second)},
		func(context.Context) error {
			calls++

			return fmt.Errorf("%w: withheld", errItemNotAvailable)
		},
	)

	require.Zero(t, delay)
	require.Equal(t, 1, calls)
	require.Equal(t, float64(1), droppedCount(s, ExecutionPayloadEnvelopeQueue, dropReasonNotAvailable))
	require.True(t, s.breaker.Allow(ExecutionPayloadEnvelopeQueue), "a withheld payload is not the node's fault")
}

func TestRunQueueItemSetsItemsAsideThroughAnOutage(t *testing.T) {
	for _, tc := range []struct {
		name   string
		err    error
		reason string
	}{
		{name: "store", err: errStoreUnavailable, reason: dropReasonStoreUnavailable},
		{name: "indexer", err: errIndexerUnavailable, reason: dropReasonIndexerUnavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := newTestAgent(uniqueName("outage-" + tc.name))

			calls := 0

			delay := s.runQueueItem(context.Background(), BeaconBlockQueue, s.log, retryPolicy{retries: 1}, func(context.Context) error {
				calls++

				return fmt.Errorf("%w: 429 reduce your concurrent request rate", tc.err)
			})

			require.Equal(t, 2*s.outageBackoff.base, delay)
			require.Equal(t, 1, calls, "the node is not read again until the outage has had time to clear")
			require.Equal(t, float64(1), retriedCount(s, BeaconBlockQueue, tc.reason))
			require.Zero(t, droppedCount(s, BeaconBlockQueue, tc.reason))
			require.True(t, s.breaker.Allow(BeaconBlockQueue))
		})
	}
}

// fakeFetch returns one scripted outcome per attempt, repeating the last, and records each attempt's retries.
type fakeFetch struct {
	mu      sync.Mutex
	script  []error
	calls   int
	retries []int
	done    chan struct{}
}

func newFakeFetch(script ...error) *fakeFetch {
	return &fakeFetch{script: script, done: make(chan struct{})}
}

func (f *fakeFetch) attempt(retries int) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	err := f.script[min(f.calls, len(f.script)-1)]
	f.calls++
	f.retries = append(f.retries, retries)

	if err == nil {
		close(f.done)
	}

	return err
}

func (f *fakeFetch) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.calls
}

func (f *fakeFetch) seen() []int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return append([]int(nil), f.retries...)
}

func (f *fakeFetch) waitForSuccess(t *testing.T) {
	t.Helper()

	select {
	case <-f.done:
	case <-time.After(5 * time.Second):
		t.Fatalf("item never succeeded after %d attempts", f.count())
	}
}

func claimHeld(s *agent, key string) bool {
	s.pending.mu.Lock()
	defer s.pending.mu.Unlock()

	_, held := s.pending.items[key]

	return held
}

// startWorker drains one queue as the agent does until the test ends.
func startWorker[T queued](
	t *testing.T,
	s *agent,
	kind Queue,
	queue chan T,
	requeue func(ctx context.Context, item T),
	handle func(ctx context.Context, item T) time.Duration,
) context.Context {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	stopped := make(chan struct{})

	var requeueItem func(item T)
	if requeue != nil {
		requeueItem = func(item T) { requeue(ctx, item) }
	}

	go func() {
		defer close(stopped)

		runWorker(ctx, s, kind, queue, requeueItem, func(item T) time.Duration { return handle(ctx, item) })
	}()

	t.Cleanup(func() {
		cancel()
		<-stopped
		require.True(t, s.workers.Wait(time.Second), "no retry may outlive the agent")
	})

	return ctx
}

// envelopeHandler mirrors the envelope queue's handler with a fixed reveal deadline.
func envelopeHandler(s *agent, revealBy time.Time, fetch *fakeFetch) func(context.Context, *ExecutionPayloadEnvelopeRequest) time.Duration {
	return func(ctx context.Context, req *ExecutionPayloadEnvelopeRequest) time.Duration {
		retry := retryPolicy{retries: req.retries, revealBy: revealBy}

		return s.runQueueItem(ctx, ExecutionPayloadEnvelopeQueue, s.log, retry, func(context.Context) error {
			return fetch.attempt(req.retries)
		})
	}
}

func blockHandler(s *agent, fetch *fakeFetch) func(context.Context, *BeaconBlockRequest) time.Duration {
	return func(ctx context.Context, req *BeaconBlockRequest) time.Duration {
		return s.runQueueItem(ctx, BeaconBlockQueue, s.log, retryPolicy{retries: req.retries}, func(context.Context) error {
			return fetch.attempt(req.retries)
		})
	}
}

func TestWorkerCapturesALateRevealedEnvelope(t *testing.T) {
	s := newQueueTestAgent(uniqueName("late-reveal"), 4)
	s.revealBackoff = backoff{base: 5 * time.Millisecond, max: 20 * time.Millisecond, limit: 10}

	notYet := fmt.Errorf("%w: not revealed yet", errItemNotAvailable)
	fetch := newFakeFetch(notYet, notYet, notYet, nil)

	ctx := startWorker(t, s, ExecutionPayloadEnvelopeQueue, s.executionPayloadEnvelopeQueue, nil,
		envelopeHandler(s, time.Now().Add(5*time.Second), fetch))

	s.enqueueExecutionPayloadEnvelope(ctx, phase0.Slot(100))

	fetch.waitForSuccess(t)

	require.Equal(t, []int{0, 1, 2, 3}, fetch.seen(), "each look is a retry of the same item")
	require.Equal(t, float64(3), retriedCount(s, ExecutionPayloadEnvelopeQueue, dropReasonNotAvailable))
	require.Zero(t, droppedCount(s, ExecutionPayloadEnvelopeQueue, dropReasonNotAvailable))
}

func TestWorkerGivesUpOnAWithheldEnvelopeOnceItsWindowCloses(t *testing.T) {
	s := newQueueTestAgent(uniqueName("withheld"), 4)
	s.revealBackoff = backoff{base: 5 * time.Millisecond, max: 10 * time.Millisecond, limit: 100}

	fetch := newFakeFetch(fmt.Errorf("%w: withheld", errItemNotAvailable))

	ctx := startWorker(t, s, ExecutionPayloadEnvelopeQueue, s.executionPayloadEnvelopeQueue, nil,
		envelopeHandler(s, time.Now().Add(80*time.Millisecond), fetch))

	s.enqueueExecutionPayloadEnvelope(ctx, phase0.Slot(101))

	require.Eventually(t, func() bool {
		return droppedCount(s, ExecutionPayloadEnvelopeQueue, dropReasonNotAvailable) == 1
	}, 5*time.Second, 5*time.Millisecond)

	attempts := fetch.count()

	// Longer than any retry delay: nothing may still be scheduled.
	time.Sleep(50 * time.Millisecond)

	require.Equal(t, attempts, fetch.count(), "a closed window ends the item for good")
	require.GreaterOrEqual(t, attempts, 3, "the window was spent looking, not waiting idle")
	require.Equal(t, float64(attempts-1), retriedCount(s, ExecutionPayloadEnvelopeQueue, dropReasonNotAvailable))
	require.True(t, s.breaker.Allow(ExecutionPayloadEnvelopeQueue))
	require.True(t, s.pending.claim(pendingKey(ExecutionPayloadEnvelopeQueue, "101"), false), "the dropped item holds no claim")
}

func TestWorkerRequeuesAnItemThroughAStoreOutage(t *testing.T) {
	s := newQueueTestAgent(uniqueName("store-requeue"), 4)
	s.outageBackoff = backoff{base: 5 * time.Millisecond, max: 20 * time.Millisecond, limit: 6}

	outage := fmt.Errorf("%w: 429 reduce your concurrent request rate", errStoreUnavailable)
	fetch := newFakeFetch(outage, outage, nil)

	ctx := startWorker(t, s, BeaconBlockQueue, s.beaconBlockQueue, nil, blockHandler(s, fetch))

	s.enqueueBeaconBlock(ctx, phase0.Slot(200))

	fetch.waitForSuccess(t)

	require.Equal(t, []int{0, 1, 2}, fetch.seen())
	require.Equal(t, float64(2), retriedCount(s, BeaconBlockQueue, dropReasonStoreUnavailable))
	require.Zero(t, droppedCount(s, BeaconBlockQueue, dropReasonStoreUnavailable))
	require.True(t, s.breaker.Allow(BeaconBlockQueue))
}

func TestWorkerDropsAnItemOnceAnOutageOutlastsItsRetries(t *testing.T) {
	s := newQueueTestAgent(uniqueName("store-exhausted"), 4)
	s.outageBackoff = backoff{base: time.Millisecond, max: 2 * time.Millisecond, limit: 3}

	fetch := newFakeFetch(fmt.Errorf("%w: bucket is unreachable", errStoreUnavailable))

	ctx := startWorker(t, s, BeaconBlockQueue, s.beaconBlockQueue, nil, blockHandler(s, fetch))

	s.enqueueBeaconBlock(ctx, phase0.Slot(201))

	require.Eventually(t, func() bool {
		return droppedCount(s, BeaconBlockQueue, dropReasonStoreUnavailable) == 1
	}, 5*time.Second, time.Millisecond)

	require.Equal(t, []int{0, 1, 2, 3}, fetch.seen(), "one attempt, then one per retry")
	require.Equal(t, float64(3), retriedCount(s, BeaconBlockQueue, dropReasonStoreUnavailable))
	require.True(t, s.breaker.Allow(BeaconBlockQueue), "an outage is never held against the node")
}

func TestWorkerRunsAFreshRequestWhileAnItemWaitsToRetry(t *testing.T) {
	s := newQueueTestAgent(uniqueName("fresh-request"), 4)
	s.revealBackoff = backoff{base: time.Hour, max: time.Hour, limit: 10}

	fetch := newFakeFetch(fmt.Errorf("%w: not revealed yet", errItemNotAvailable), nil)

	ctx := startWorker(t, s, ExecutionPayloadEnvelopeQueue, s.executionPayloadEnvelopeQueue, nil,
		envelopeHandler(s, time.Now().Add(2*time.Hour), fetch))

	s.enqueueExecutionPayloadEnvelope(ctx, phase0.Slot(300))

	key := pendingKey(ExecutionPayloadEnvelopeQueue, "300")

	require.Eventually(t, func() bool {
		return retriedCount(s, ExecutionPayloadEnvelopeQueue, dropReasonNotAvailable) == 1 && !claimHeld(s, key)
	}, 5*time.Second, time.Millisecond, "the item waits for its retry without holding its claim")

	// The payload event arrives while the first copy waits out its hour.
	s.enqueueExecutionPayloadEnvelope(ctx, phase0.Slot(300))

	fetch.waitForSuccess(t)

	require.Equal(t, 2, fetch.count())
	require.Zero(t, droppedCount(s, ExecutionPayloadEnvelopeQueue, dropReasonRedoRequested))
	require.Zero(t, droppedCount(s, ExecutionPayloadEnvelopeQueue, dropReasonDuplicate))
}

func TestWorkerRerunsARedoInsteadOfWaitingToRetry(t *testing.T) {
	s := newQueueTestAgent(uniqueName("redo-over-retry"), 4)
	s.revealBackoff = backoff{base: time.Hour, max: time.Hour, limit: 10}

	fetch := newFakeFetch(fmt.Errorf("%w: not revealed yet", errItemNotAvailable), nil)
	handle := envelopeHandler(s, time.Now().Add(2*time.Hour), fetch)

	requeue := func(ctx context.Context, item *ExecutionPayloadEnvelopeRequest) {
		s.enqueueExecutionPayloadEnvelope(ctx, item.Slot)
	}

	var announced sync.Once

	ctx := startWorker(t, s, ExecutionPayloadEnvelopeQueue, s.executionPayloadEnvelopeQueue, requeue,
		func(ctx context.Context, req *ExecutionPayloadEnvelopeRequest) time.Duration {
			// The node announces the payload while the first look is in flight.
			announced.Do(func() { s.enqueueExecutionPayloadEnvelope(ctx, req.Slot) })

			return handle(ctx, req)
		})

	s.enqueueExecutionPayloadEnvelope(ctx, phase0.Slot(301))

	fetch.waitForSuccess(t)

	require.Equal(t, []int{0, 0}, fetch.seen(), "the redo runs fresh and at once, not after the hour-long retry")
	require.Equal(t, float64(1), droppedCount(s, ExecutionPayloadEnvelopeQueue, dropReasonRedoRequested))
}
