package agent

import (
	"context"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

// A histogram series only exists once something has been observed into it, and
// every test here uses an agent name of its own, so a new series appearing is
// proof that the item was measured.

func TestRunWorkerMeasuresAnItemTheHandlerGaveUpOn(t *testing.T) {
	s := newQueueTestAgent("worker-drop", 4)

	waits := testutil.CollectAndCount(s.metrics.queueWaitTime)
	processing := testutil.CollectAndCount(s.metrics.queueItemProcessingTime)

	ctx, cancel := context.WithCancel(context.Background())

	s.beaconStateQueue <- &BeaconStateRequest{Slot: 3, queueItem: newQueueItem(pendingKey(BeaconStateQueue, "3"))}

	runWorker(ctx, s, BeaconStateQueue, s.beaconStateQueue, nil, func(*BeaconStateRequest) {
		// A handler that returns without doing the work is the case the timing
		// used to miss entirely.
		cancel()
	})

	require.Equal(t, waits+1, testutil.CollectAndCount(s.metrics.queueWaitTime))
	require.Equal(t, processing+1, testutil.CollectAndCount(s.metrics.queueItemProcessingTime))
}

func TestRunWorkerReleasesTheClaimWhateverHappens(t *testing.T) {
	s := newQueueTestAgent("worker-release", 4)

	ctx, cancel := context.WithCancel(context.Background())

	key := pendingKey(BeaconStateQueue, "9")
	require.True(t, s.pending.claim(key, false))

	s.beaconStateQueue <- &BeaconStateRequest{Slot: 9, queueItem: newQueueItem(key)}

	runWorker(ctx, s, BeaconStateQueue, s.beaconStateQueue, nil, func(*BeaconStateRequest) {
		cancel()
	})

	require.True(t, s.pending.claim(key, false), "an item the worker gave up on must still release its claim")
}
