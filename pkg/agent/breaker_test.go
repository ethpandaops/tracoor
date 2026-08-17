package agent

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/0xsequence/ethkit/ethrpc/jsonrpc"
	"github.com/ethpandaops/beacon/pkg/beacon/api"
	"github.com/stretchr/testify/require"
)

// newTestBreaker returns a breaker whose clock the test drives directly.
func newTestBreaker() (*circuitBreaker, func(time.Duration)) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	breaker := newCircuitBreaker()
	breaker.now = func() time.Time { return now }

	return breaker, func(d time.Duration) { now = now.Add(d) }
}

func TestBreakerTripsAfterConsecutiveFailures(t *testing.T) {
	breaker, _ := newTestBreaker()

	for i := 1; i < breakerFailureThreshold; i++ {
		require.False(t, breaker.RecordFailure(BeaconStateQueue, failureTransient, 0))
		require.True(t, breaker.Allow(BeaconStateQueue))
	}

	require.True(t, breaker.RecordFailure(BeaconStateQueue, failureTransient, 0))
	require.False(t, breaker.Allow(BeaconStateQueue))
}

func TestBreakerIsPerArtifact(t *testing.T) {
	breaker, _ := newTestBreaker()

	for i := 0; i < breakerFailureThreshold; i++ {
		breaker.RecordFailure(ExecutionBlockTraceQueue, failureTransient, 0)
	}

	require.False(t, breaker.Allow(ExecutionBlockTraceQueue))
	require.True(t, breaker.Allow(BeaconStateQueue))
}

func TestBreakerSuccessResetsFailures(t *testing.T) {
	breaker, _ := newTestBreaker()

	breaker.RecordFailure(BeaconStateQueue, failureTransient, 0)
	breaker.RecordFailure(BeaconStateQueue, failureTransient, 0)
	breaker.RecordSuccess(BeaconStateQueue)

	require.False(t, breaker.RecordFailure(BeaconStateQueue, failureTransient, 0))
	require.True(t, breaker.Allow(BeaconStateQueue))
}

func TestBreakerPermanentFailureTripsImmediately(t *testing.T) {
	breaker, _ := newTestBreaker()

	require.True(t, breaker.RecordFailure(ExecutionBlockTraceQueue, failurePermanent, 0))
	require.False(t, breaker.Allow(ExecutionBlockTraceQueue))
}

func TestBreakerHalfOpensAfterCooldown(t *testing.T) {
	breaker, advance := newTestBreaker()

	breaker.RecordFailure(ExecutionBlockTraceQueue, failurePermanent, 0)
	require.False(t, breaker.Allow(ExecutionBlockTraceQueue))

	advance(breakerCooldown - time.Second)
	require.False(t, breaker.Allow(ExecutionBlockTraceQueue))

	advance(time.Second)
	require.True(t, breaker.Allow(ExecutionBlockTraceQueue))
}

func TestBreakerHalfOpenProbeFailureReopens(t *testing.T) {
	breaker, advance := newTestBreaker()

	breaker.RecordFailure(ExecutionBlockTraceQueue, failurePermanent, 0)
	advance(breakerCooldown)

	require.True(t, breaker.Allow(ExecutionBlockTraceQueue))

	// A single failure from the half-open window re-opens immediately rather
	// than starting the consecutive count over.
	require.True(t, breaker.RecordFailure(ExecutionBlockTraceQueue, failureTransient, 0))
	require.False(t, breaker.Allow(ExecutionBlockTraceQueue))
}

func TestBreakerHalfOpenProbeSuccessRecovers(t *testing.T) {
	breaker, advance := newTestBreaker()

	breaker.RecordFailure(ExecutionBlockTraceQueue, failurePermanent, 0)
	advance(breakerCooldown)

	require.True(t, breaker.Allow(ExecutionBlockTraceQueue))

	breaker.RecordSuccess(ExecutionBlockTraceQueue)
	require.True(t, breaker.Allow(ExecutionBlockTraceQueue))

	// Recovery is complete: the full consecutive count is required again.
	require.False(t, breaker.RecordFailure(ExecutionBlockTraceQueue, failureTransient, 0))
	require.True(t, breaker.Allow(ExecutionBlockTraceQueue))
}

func TestBreakerNeverLatchesPermanently(t *testing.T) {
	breaker, advance := newTestBreaker()

	for i := 0; i < 5; i++ {
		require.True(t, breaker.RecordFailure(ExecutionBlockTraceQueue, failurePermanent, 0))
		require.False(t, breaker.Allow(ExecutionBlockTraceQueue))

		advance(breakerCooldown)

		require.True(t, breaker.Allow(ExecutionBlockTraceQueue))
	}
}

func TestBreakerHonoursRetryAfter(t *testing.T) {
	breaker, advance := newTestBreaker()

	breaker.RecordFailure(BeaconStateQueue, failureTransient, 30*time.Second)
	breaker.RecordFailure(BeaconStateQueue, failureTransient, 30*time.Second)
	breaker.RecordFailure(BeaconStateQueue, failureTransient, 30*time.Second)

	advance(29 * time.Second)
	require.False(t, breaker.Allow(BeaconStateQueue))

	// The hint is used in place of the default cooldown, not on top of it.
	advance(time.Second)
	require.True(t, breaker.Allow(BeaconStateQueue))
}

func TestBreakerClampsRetryAfter(t *testing.T) {
	breaker, advance := newTestBreaker()

	breaker.RecordFailure(BeaconStateQueue, failurePermanent, 24*time.Hour)

	advance(breakerMaxCooldown)
	require.True(t, breaker.Allow(BeaconStateQueue))
}

func TestClassifyFailure(t *testing.T) {
	tests := []struct {
		name          string
		kind          Queue
		err           error
		expectedClass failureClass
	}{
		{
			name:          "not implemented is permanent",
			kind:          ExecutionPayloadEnvelopeQueue,
			err:           &api.HTTPStatusError{StatusCode: http.StatusNotImplemented},
			expectedClass: failurePermanent,
		},
		{
			name:          "method not allowed is permanent",
			kind:          BeaconStateQueue,
			err:           &api.HTTPStatusError{StatusCode: http.StatusMethodNotAllowed},
			expectedClass: failurePermanent,
		},
		{
			name:          "not found on the envelope path is permanent",
			kind:          ExecutionPayloadEnvelopeQueue,
			err:           &api.HTTPStatusError{StatusCode: http.StatusNotFound},
			expectedClass: failurePermanent,
		},
		{
			name:          "not found on the state path describes the slot",
			kind:          BeaconStateQueue,
			err:           &api.HTTPStatusError{StatusCode: http.StatusNotFound},
			expectedClass: failureTransient,
		},
		{
			name:          "not found on the block path describes the slot",
			kind:          BeaconBlockQueue,
			err:           &api.HTTPStatusError{StatusCode: http.StatusNotFound},
			expectedClass: failureTransient,
		},
		{
			name:          "server error on the state path is transient",
			kind:          BeaconStateQueue,
			err:           &api.HTTPStatusError{StatusCode: http.StatusInternalServerError},
			expectedClass: failureTransient,
		},
		{
			name:          "server error elsewhere is transient",
			kind:          ExecutionBlockTraceQueue,
			err:           &api.HTTPStatusError{StatusCode: http.StatusBadGateway},
			expectedClass: failureTransient,
		},
		{
			name:          "json-rpc method not found is permanent",
			kind:          ExecutionBlockTraceQueue,
			err:           fmt.Errorf("call failed: %w", &jsonrpc.Error{Code: jsonRPCMethodNotFound, Message: "method not found"}),
			expectedClass: failurePermanent,
		},
		{
			name:          "other json-rpc errors are transient",
			kind:          ExecutionBlockTraceQueue,
			err:           fmt.Errorf("call failed: %w", &jsonrpc.Error{Code: -32000, Message: "boom"}),
			expectedClass: failureTransient,
		},
		{
			name:          "an untyped error is transient",
			kind:          BeaconStateQueue,
			err:           fmt.Errorf("connection reset"),
			expectedClass: failureTransient,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			class, _ := classifyFailure(test.kind, test.err)
			require.Equal(t, test.expectedClass, class)
		})
	}
}

func TestClassifyFailureExtractsRetryAfter(t *testing.T) {
	_, retryAfter := classifyFailure(BeaconStateQueue, &api.HTTPStatusError{
		StatusCode: http.StatusServiceUnavailable,
		RetryAfter: "42",
	})

	require.Equal(t, 42*time.Second, retryAfter)
}

func TestParseRetryAfter(t *testing.T) {
	require.Equal(t, time.Duration(0), parseRetryAfter(""))
	require.Equal(t, time.Duration(0), parseRetryAfter("not-a-header"))
	require.Equal(t, time.Duration(0), parseRetryAfter("0"))
	require.Equal(t, time.Duration(0), parseRetryAfter("-5"))
	require.Equal(t, 15*time.Second, parseRetryAfter("15"))

	// An HTTP-date in the past carries no useful hint.
	require.Equal(t, time.Duration(0), parseRetryAfter(time.Now().Add(-time.Hour).UTC().Format(http.TimeFormat)))
	require.Positive(t, parseRetryAfter(time.Now().Add(time.Hour).UTC().Format(http.TimeFormat)))
}
