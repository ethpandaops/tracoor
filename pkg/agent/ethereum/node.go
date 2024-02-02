package ethereum

import (
	"context"

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

	executionReady bool
	beaconReady    bool
}

func NewNode(ctx context.Context, log logrus.FieldLogger, config *Config, node string) *Node {
	return &Node{
		log:       log.WithField("module", "agent/ethereum/node"),
		beacon:    beacon.NewNode(ctx, log, node, config.OverrideNetworkName, config.Beacon),
		execution: execution.NewNode(log, config.Execution),
	}
}

func (n *Node) Execution() *execution.Node {
	return n.execution
}

func (n *Node) Beacon() *beacon.Node {
	return n.beacon
}

func (n *Node) Start(ctx context.Context) error {
	n.beacon.OnReady(ctx, func(ctx context.Context) error {
		n.beaconReady = true

		n.checkReadyPublish(ctx)

		return nil
	})

	n.execution.OnReady(ctx, func(ctx context.Context) error {
		n.executionReady = true

		n.checkReadyPublish(ctx)

		return nil
	})

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return n.beacon.Start(gCtx)
	})

	g.Go(func() error {
		return n.execution.Start(gCtx)
	})

	return nil
}

func (n *Node) OnReady(_ context.Context, callback func(ctx context.Context) error) {
	n.onReadyCallbacks = append(n.onReadyCallbacks, callback)
}

func (n *Node) checkReadyPublish(ctx context.Context) {
	if n.beaconReady && n.executionReady {
		for _, callback := range n.onReadyCallbacks {
			if err := callback(ctx); err != nil {
				n.log.WithError(err).Error("error executing on_ready callback")
			}
		}
	}
}
