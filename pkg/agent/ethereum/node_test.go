package ethereum

import (
	"context"
	"io"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func newReadyTestNode() *Node {
	log := logrus.New()
	log.SetOutput(io.Discard)

	return &Node{log: log}
}

// TestOnReadyFiresExactlyOnceWhenBothReadinessesRace drives the beacon and
// execution readiness from separate goroutines, the way they arrive live. The
// callbacks start the agent's worker set, so firing twice doubles the workers
// on one queue and never firing leaves a zombie agent; run under -race this
// also exercises the flag accesses themselves.
func TestOnReadyFiresExactlyOnceWhenBothReadinessesRace(t *testing.T) {
	for i := 0; i < 100; i++ {
		n := newReadyTestNode()

		var fired atomic.Int32

		n.OnReady(context.Background(), func(context.Context) error {
			fired.Add(1)

			return nil
		})

		var wg sync.WaitGroup

		wg.Add(2)

		go func() {
			defer wg.Done()

			n.markBeaconReady(context.Background())
		}()

		go func() {
			defer wg.Done()

			n.markExecutionReady(context.Background())
		}()

		wg.Wait()

		require.Equal(t, int32(1), fired.Load(), "the callbacks run exactly once, on whichever readiness lands last")
	}
}

func TestOnReadyDoesNotFireAgainOnARepeatedReadiness(t *testing.T) {
	n := newReadyTestNode()

	var fired atomic.Int32

	n.OnReady(context.Background(), func(context.Context) error {
		fired.Add(1)

		return nil
	})

	n.markBeaconReady(context.Background())
	require.Zero(t, fired.Load(), "one node ready is not both")

	n.markExecutionReady(context.Background())
	require.Equal(t, int32(1), fired.Load())

	// The beacon library re-fires readiness on reconnects; the worker set must
	// not be started again for it.
	n.markBeaconReady(context.Background())
	n.markExecutionReady(context.Background())
	require.Equal(t, int32(1), fired.Load())
}
