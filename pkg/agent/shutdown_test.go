package agent

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWorkerGroupDrainsWithinTimeout(t *testing.T) {
	var (
		workers  workerGroup
		finished atomic.Bool
	)

	workers.Go(func() {
		time.Sleep(10 * time.Millisecond)

		finished.Store(true)
	})

	require.True(t, workers.Wait(time.Second))
	require.True(t, finished.Load())
}

func TestWorkerGroupAbandonsAfterTimeout(t *testing.T) {
	var workers workerGroup

	release := make(chan struct{})

	workers.Go(func() { <-release })

	start := time.Now()

	require.False(t, workers.Wait(20*time.Millisecond))
	require.Less(t, time.Since(start), time.Second)

	close(release)

	require.True(t, workers.Wait(time.Second))
}

func TestWorkerGroupReturnsImmediatelyWhenIdle(t *testing.T) {
	var workers workerGroup

	require.True(t, workers.Wait(time.Second))
}

func TestWorkerGroupRefusesWorkersOnceDraining(t *testing.T) {
	var (
		workers workerGroup
		started atomic.Bool
	)

	require.True(t, workers.Wait(time.Second))

	workers.Go(func() { started.Store(true) })

	require.True(t, workers.Wait(20*time.Millisecond))
	require.False(t, started.Load())
}

func TestAgentWorkersAreWaitedOn(t *testing.T) {
	s := newTestAgent("track")

	release := make(chan struct{})

	s.workers.Go(func() { <-release })

	require.False(t, s.workers.Wait(20*time.Millisecond))

	close(release)

	require.True(t, s.workers.Wait(time.Second))
}

func TestDrainQueueStopsOnCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	queue := make(chan int, 4)
	queue <- 1

	var (
		handled  atomic.Int64
		returned = make(chan struct{})
	)

	go func() {
		defer close(returned)

		drainQueue(ctx, queue, func(int) {
			handled.Add(1)

			cancel()
		})
	}()

	select {
	case <-returned:
	case <-time.After(time.Second):
		t.Fatal("drainQueue did not return after its context was cancelled")
	}

	// The backlog left behind is abandoned rather than drained.
	queue <- 2

	require.Equal(t, int64(1), handled.Load())
}

func TestDrainQueueStopsWhenQueueCloses(t *testing.T) {
	queue := make(chan int, 2)

	queue <- 1

	close(queue)

	handled := 0

	drainQueue(context.Background(), queue, func(int) {
		handled++
	})

	require.Equal(t, 1, handled)
}

func TestEnqueueGivesUpWhenCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// A full queue with no reader: without the cancellation case this blocks
	// forever and holds the shutdown open.
	queue := make(chan int, 1)
	queue <- 1

	done := make(chan struct{})

	go func() {
		defer close(done)

		enqueue(ctx, queue, 2)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("enqueue blocked on a full queue after its context was cancelled")
	}
}

func TestConfigShutdownTimeout(t *testing.T) {
	require.Equal(t, 30*time.Second, (&Config{ShutdownTimeoutSeconds: 30}).ShutdownTimeout())
	require.Equal(t, defaultShutdownTimeout, (&Config{}).ShutdownTimeout())
	require.Equal(t, defaultShutdownTimeout, (&Config{ShutdownTimeoutSeconds: -1}).ShutdownTimeout())
}
