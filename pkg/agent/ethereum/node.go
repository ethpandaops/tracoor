package ethereum

import (
	"context"
	"errors"
	"sync"

	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/tracoor/pkg/agent/ethereum/beacon"
	"github.com/ethpandaops/tracoor/pkg/agent/ethereum/execution"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

type Node struct {
	log logrus.FieldLogger

	beacon    *beacon.Node
	execution *execution.Node

	onReadyCallbacks []func(ctx context.Context) error

	// readyMu guards the readiness flags and readyFired: beacon and execution
	// readiness land on separate goroutines, and the OnReady callbacks must run
	// exactly once — with whichever readiness arrives last — because they start
	// the agent's worker set.
	readyMu        sync.Mutex
	executionReady bool
	beaconReady    bool
	readyFired     bool

	syncToleranceSlots phase0.Slot

	// failed carries the first problem that leaves this node unusable. Only the
	// first one matters — whoever owns the node stops on it either way — so it
	// is buffered and reported at most once.
	failed     chan error
	failedOnce sync.Once
}

func NewNode(ctx context.Context, log logrus.FieldLogger, config *Config, node string, syncToleranceSlots phase0.Slot) *Node {
	return &Node{
		log:                log.WithField("module", "agent/ethereum/node"),
		beacon:             beacon.NewNode(ctx, log, node, config.OverrideNetworkName, config.Beacon),
		execution:          execution.NewNode(log, config.Execution),
		syncToleranceSlots: syncToleranceSlots,
		failed:             make(chan error, 1),
	}
}

// Failed reports that this node can no longer be used. It is how one sick node
// stops one agent instead of the process it happens to share.
func (n *Node) Failed() <-chan error {
	return n.failed
}

func (n *Node) reportFailure(err error) {
	n.failedOnce.Do(func() {
		n.failed <- err
	})
}

func (n *Node) Execution() *execution.Node {
	return n.execution
}

func (n *Node) Beacon() *beacon.Node {
	return n.beacon
}

func (n *Node) Start(ctx context.Context) error {
	n.beacon.OnReady(ctx, func(ctx context.Context) error {
		n.markBeaconReady(ctx)

		return nil
	})

	n.execution.OnReady(ctx, func(ctx context.Context) error {
		n.markExecutionReady(ctx)

		return nil
	})

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return n.beacon.Start(gCtx)
	})

	g.Go(func() error {
		return n.execution.Start(gCtx)
	})

	// Start does not block, so the group is waited on here: without this the
	// only account of why a node stopped would be discarded. A shutdown is not
	// a failure, so a cancelled context is not reported as one.
	go func() {
		if err := g.Wait(); err != nil && ctx.Err() == nil && !errors.Is(err, context.Canceled) {
			n.log.WithError(err).Error("Ethereum node stopped")

			n.reportFailure(err)
		}
	}()

	return nil
}

func (n *Node) OnReady(_ context.Context, callback func(ctx context.Context) error) {
	n.onReadyCallbacks = append(n.onReadyCallbacks, callback)
}

func (n *Node) markBeaconReady(ctx context.Context) {
	n.readyMu.Lock()
	n.beaconReady = true
	n.readyMu.Unlock()

	n.checkReadyPublish(ctx)
}

func (n *Node) markExecutionReady(ctx context.Context) {
	n.readyMu.Lock()
	n.executionReady = true
	n.readyMu.Unlock()

	n.checkReadyPublish(ctx)
}

// checkReadyPublish fires the OnReady callbacks once both nodes are ready.
// readyFired is claimed under the lock so the callbacks run exactly once, on
// whichever readiness landed last; they run outside it because they are the
// caller's code and may block.
func (n *Node) checkReadyPublish(ctx context.Context) {
	n.readyMu.Lock()

	if !n.beaconReady || !n.executionReady || n.readyFired {
		n.readyMu.Unlock()

		return
	}

	n.readyFired = true

	n.readyMu.Unlock()

	for _, callback := range n.onReadyCallbacks {
		if err := callback(ctx); err != nil {
			n.log.WithError(err).Error("error executing on_ready callback")
		}
	}
}

func (n *Node) ShouldIgnoreEventFromSlot(slot phase0.Slot) (bool, error) {
	wallclock := n.beacon.Metadata().Wallclock()
	if wallclock == nil {
		return false, errors.New("missing wallclock")
	}

	currentSlot := wallclock.Slots().Current()

	if phase0.Slot(currentSlot.Number())-slot > n.syncToleranceSlots {
		return true, nil
	}

	return false, nil
}
