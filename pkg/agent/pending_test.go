package agent

import (
	"context"
	"sync"
	"testing"

	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/stretchr/testify/require"
)

// newQueueTestAgent builds an agent with just enough wiring to exercise the
// enqueue and drain paths, and no ethereum node behind them.
func newQueueTestAgent(name string, size int) *agent {
	s := newTestAgent(name)

	s.pending = newPendingItems()
	s.beaconStateQueue = make(chan *BeaconStateRequest, size)
	s.beaconBlockQueue = make(chan *BeaconBlockRequest, size)
	s.executionBlockTraceQueue = make(chan *ExecutionBlockTraceRequest, size)
	s.executionBadBlockQueue = make(chan *ExecutionBadBlockRequest, size)

	return s
}

func TestEnqueueCollapsesDuplicates(t *testing.T) {
	s := newQueueTestAgent("dedup", 16)

	ctx := context.Background()

	for i := 0; i < 5; i++ {
		s.enqueueBeaconState(ctx, phase0.Slot(7))
	}

	require.Len(t, s.beaconStateQueue, 1, "a slot already queued must not be queued again")
	require.Equal(t, float64(4), droppedCount(s, BeaconStateQueue, dropReasonDuplicate))
}

func TestEnqueueKeepsDistinctIdentifiersApart(t *testing.T) {
	s := newQueueTestAgent("distinct", 16)

	ctx := context.Background()

	s.enqueueBeaconState(ctx, phase0.Slot(1))
	s.enqueueBeaconState(ctx, phase0.Slot(2))

	require.Len(t, s.beaconStateQueue, 2)
	require.Zero(t, droppedCount(s, BeaconStateQueue, dropReasonDuplicate))
}

func TestEnqueueKeepsKindsApart(t *testing.T) {
	s := newQueueTestAgent("kinds", 16)

	ctx := context.Background()

	// The same slot is a different item for each artifact, so one must not
	// shadow the other.
	s.enqueueBeaconState(ctx, phase0.Slot(9))
	s.enqueueBeaconBlock(ctx, phase0.Slot(9))

	require.Len(t, s.beaconStateQueue, 1)
	require.Len(t, s.beaconBlockQueue, 1)
}

func TestEnqueueAcceptsAnItemAgainOnceItHasBeenReleased(t *testing.T) {
	s := newQueueTestAgent("release", 16)

	ctx := context.Background()

	s.enqueueBeaconState(ctx, phase0.Slot(3))

	request := <-s.beaconStateQueue
	s.releaseQueueItem(request.key)

	s.enqueueBeaconState(ctx, phase0.Slot(3))

	require.Len(t, s.beaconStateQueue, 1, "a finished item can be queued again")
}

func TestEnqueueReleasesTheClaimWhenTheQueueIsAbandoned(t *testing.T) {
	// A full queue plus a cancelled context is what a shutdown mid-enqueue
	// looks like: the item never lands, so its claim must not linger.
	s := newQueueTestAgent("abandoned", 1)

	ctx, cancel := context.WithCancel(context.Background())

	s.enqueueBeaconState(ctx, phase0.Slot(4))
	require.Len(t, s.beaconStateQueue, 1)

	cancel()

	s.enqueueBeaconState(ctx, phase0.Slot(5))

	require.Len(t, s.beaconStateQueue, 1)
	require.False(t, s.pending.claim(pendingKey(BeaconStateQueue, "4")), "the queued item still holds its claim")
	require.True(t, s.pending.claim(pendingKey(BeaconStateQueue, "5")), "the abandoned item must not hold one")
}

func TestEnqueueExecutionBadBlockHasASingleOutstandingItem(t *testing.T) {
	s := newQueueTestAgent("bad-blocks", 16)

	ctx := context.Background()

	s.enqueueExecutionBadBlock(ctx)
	s.enqueueExecutionBadBlock(ctx)
	s.enqueueExecutionBadBlock(ctx)

	require.Len(t, s.executionBadBlockQueue, 1)
	require.Equal(t, float64(2), droppedCount(s, ExecutionBadBlockQueue, dropReasonDuplicate))
}

func TestPendingItemsIsSafeUnderConcurrentUse(t *testing.T) {
	s := newQueueTestAgent("concurrent", 512)

	ctx := context.Background()

	var wg sync.WaitGroup

	for producer := 0; producer < 8; producer++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for slot := 0; slot < 64; slot++ {
				s.enqueueExecutionBlockTrace(ctx, slotIdentifier(phase0.Slot(slot)))
			}
		}()
	}

	wg.Wait()

	require.Len(t, s.executionBlockTraceQueue, 64, "every distinct block must be queued exactly once")
}
