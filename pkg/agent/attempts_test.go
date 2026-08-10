package agent

import (
	"context"
	goerrors "errors"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/ethpandaops/beacon/pkg/beacon/api"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func newTestAgent(name string) *agent {
	log := logrus.New()
	log.SetOutput(io.Discard)

	return &agent{
		Config:  &Config{Name: name},
		log:     log,
		metrics: GetMetricsInstance(namespace),
		breaker: newCircuitBreaker(),
	}
}

func droppedCount(s *agent, kind Queue, reason string) float64 {
	return testutil.ToFloat64(s.metrics.itemsDropped.WithLabelValues(string(kind), s.Config.Name, reason))
}

func unsupportedCount(s *agent, kind Queue) float64 {
	return testutil.ToFloat64(s.metrics.artifactUnsupported.WithLabelValues(string(kind), s.Config.Name))
}

func TestAttemptBudgets(t *testing.T) {
	require.Equal(t, 2, attemptBudget(BeaconStateQueue))
	require.Equal(t, 3, attemptBudget(BeaconBlockQueue))
	require.Equal(t, 2, attemptBudget(ExecutionPayloadEnvelopeQueue))
	require.Equal(t, 2, attemptBudget(ExecutionBlockTraceQueue))
	require.Equal(t, defaultAttemptBudget, attemptBudget(BeaconBadBlockQueue))
}

func TestRunQueueItemStopsOnSuccess(t *testing.T) {
	s := newTestAgent("success")

	calls := 0

	s.runQueueItem(context.Background(), BeaconBlockQueue, s.log, func(context.Context) error {
		calls++

		return nil
	})

	require.Equal(t, 1, calls)
	require.Zero(t, droppedCount(s, BeaconBlockQueue, dropReasonAttemptsExhausted))
}

func TestRunQueueItemRetriesUpToBudget(t *testing.T) {
	s := newTestAgent("retries")

	calls := 0

	s.runQueueItem(context.Background(), BeaconBlockQueue, s.log, func(context.Context) error {
		calls++

		if calls < 3 {
			return goerrors.New("transient")
		}

		return nil
	})

	require.Equal(t, attemptBudget(BeaconBlockQueue), calls)
	require.Zero(t, droppedCount(s, BeaconBlockQueue, dropReasonAttemptsExhausted))
}

func TestRunQueueItemCountsExhaustedBudget(t *testing.T) {
	s := newTestAgent("exhausted")

	calls := 0

	s.runQueueItem(context.Background(), BeaconStateQueue, s.log, func(context.Context) error {
		calls++

		return goerrors.New("transient")
	})

	require.Equal(t, attemptBudget(BeaconStateQueue), calls)
	require.Equal(t, float64(1), droppedCount(s, BeaconStateQueue, dropReasonAttemptsExhausted))
	require.Zero(t, unsupportedCount(s, BeaconStateQueue))
}

func TestRunQueueItemSpendsOneBreakerFailurePerItem(t *testing.T) {
	s := newTestAgent("per-item")

	failing := func(context.Context) error { return goerrors.New("transient") }

	for i := 1; i < breakerFailureThreshold; i++ {
		s.runQueueItem(context.Background(), BeaconStateQueue, s.log, failing)
		require.True(t, s.breaker.Allow(BeaconStateQueue))
	}

	s.runQueueItem(context.Background(), BeaconStateQueue, s.log, failing)

	require.False(t, s.breaker.Allow(BeaconStateQueue))
	require.Equal(t, float64(1), unsupportedCount(s, BeaconStateQueue))
	require.Equal(t, float64(breakerFailureThreshold), droppedCount(s, BeaconStateQueue, dropReasonAttemptsExhausted))
}

func TestRunQueueItemStopsOnPermanentFailure(t *testing.T) {
	s := newTestAgent("permanent")

	calls := 0

	s.runQueueItem(context.Background(), ExecutionBlockTraceQueue, s.log, func(context.Context) error {
		calls++

		return &api.HTTPStatusError{StatusCode: http.StatusNotImplemented}
	})

	require.Equal(t, 1, calls)
	require.False(t, s.breaker.Allow(ExecutionBlockTraceQueue))
	require.Equal(t, float64(1), unsupportedCount(s, ExecutionBlockTraceQueue))
	require.Equal(t, float64(1), droppedCount(s, ExecutionBlockTraceQueue, dropReasonUnsupported))
}

func TestRunQueueItemSkipsWhenBreakerIsOpen(t *testing.T) {
	s := newTestAgent("open")

	s.breaker.RecordFailure(ExecutionBlockTraceQueue, failurePermanent, 0)

	calls := 0

	s.runQueueItem(context.Background(), ExecutionBlockTraceQueue, s.log, func(context.Context) error {
		calls++

		return nil
	})

	require.Zero(t, calls)
	require.Equal(t, float64(1), droppedCount(s, ExecutionBlockTraceQueue, dropReasonCircuitOpen))
}

func TestRunQueueItemDoesNotBlameNodeForAbsentArtifact(t *testing.T) {
	s := newTestAgent("absent")

	calls := 0

	for i := 0; i < breakerFailureThreshold+1; i++ {
		s.runQueueItem(context.Background(), ExecutionPayloadEnvelopeQueue, s.log, func(context.Context) error {
			calls++

			return fmt.Errorf("%w: never revealed", errItemNotAvailable)
		})
	}

	require.Equal(t, attemptBudget(ExecutionPayloadEnvelopeQueue)*(breakerFailureThreshold+1), calls)
	require.True(t, s.breaker.Allow(ExecutionPayloadEnvelopeQueue))
	require.Zero(t, unsupportedCount(s, ExecutionPayloadEnvelopeQueue))
	require.Equal(t, float64(breakerFailureThreshold+1), droppedCount(s, ExecutionPayloadEnvelopeQueue, dropReasonNotAvailable))
}

func TestRunQueueItemDropsStaleItemsWithoutRetrying(t *testing.T) {
	s := newTestAgent("stale")

	calls := 0

	for i := 0; i < breakerFailureThreshold+1; i++ {
		s.runQueueItem(context.Background(), ExecutionBlockTraceQueue, s.log, func(context.Context) error {
			calls++

			return fmt.Errorf("%w: execution block 1", errItemStale)
		})
	}

	require.Equal(t, breakerFailureThreshold+1, calls)
	require.True(t, s.breaker.Allow(ExecutionBlockTraceQueue))
	require.Equal(t, float64(breakerFailureThreshold+1), droppedCount(s, ExecutionBlockTraceQueue, dropReasonStale))
}

func TestRunQueueItemDropsDivergentItemsWithoutBlamingTheNode(t *testing.T) {
	s := newTestAgent("divergent")

	calls := 0

	for i := 0; i < breakerFailureThreshold+1; i++ {
		s.runQueueItem(context.Background(), BeaconStateQueue, s.log, func(context.Context) error {
			calls++

			return fmt.Errorf("%w: slot 1", errPayloadDivergent)
		})
	}

	require.Equal(t, breakerFailureThreshold+1, calls, "the divergence was recorded already, so there is nothing to retry")
	require.True(t, s.breaker.Allow(BeaconStateQueue), "a node that answers differently is still answering")
	require.Zero(t, unsupportedCount(s, BeaconStateQueue))
	require.Equal(t, float64(breakerFailureThreshold+1), droppedCount(s, BeaconStateQueue, dropReasonDivergent))
}

func TestRunQueueItemStopsOnCancelledContext(t *testing.T) {
	s := newTestAgent("cancelled")

	ctx, cancel := context.WithCancel(context.Background())

	calls := 0

	s.runQueueItem(ctx, BeaconBlockQueue, s.log, func(context.Context) error {
		calls++

		cancel()

		return context.Canceled
	})

	require.Equal(t, 1, calls)
	require.Zero(t, droppedCount(s, BeaconBlockQueue, dropReasonAttemptsExhausted))
	require.True(t, s.breaker.Allow(BeaconBlockQueue))
}

func TestRunQueueItemDoesNotBlameTheNodeForAStoreOutage(t *testing.T) {
	s := newTestAgent("store-outage")

	calls := 0

	// A store that will not take the payload is down for every node at once,
	// so pausing the nodes over it would silence the whole fleet.
	failing := func(context.Context) error {
		calls++

		return fmt.Errorf("%w: bucket is unreachable", errStoreUnavailable)
	}

	for i := 0; i < breakerFailureThreshold+1; i++ {
		s.runQueueItem(context.Background(), BeaconStateQueue, s.log, failing)
	}

	require.Equal(t, breakerFailureThreshold+1, calls, "the node is not read again for a failure that was not its own")
	require.True(t, s.breaker.Allow(BeaconStateQueue), "a healthy node must keep serving through a store outage")
	require.Zero(t, unsupportedCount(s, BeaconStateQueue))
	require.Equal(t, float64(breakerFailureThreshold+1), droppedCount(s, BeaconStateQueue, dropReasonStoreUnavailable))
	require.Zero(t, droppedCount(s, BeaconStateQueue, dropReasonAttemptsExhausted))
}
