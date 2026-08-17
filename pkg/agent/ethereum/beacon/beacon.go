package beacon

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	bn "github.com/ethpandaops/beacon/pkg/beacon"
	"github.com/ethpandaops/tracoor/pkg/agent/ethereum/beacon/services"
	"github.com/ethpandaops/tracoor/pkg/mime"
	"github.com/go-co-op/gocron"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type Node struct {
	config *Config
	log    logrus.FieldLogger

	beacon bn.Node

	services []services.Service

	onReadyCallbacks []func(ctx context.Context) error
}

func NewNode(ctx context.Context, log logrus.FieldLogger, name, overrideNetworkName string, config *Config) *Node {
	// The node's metrics stay on for the raw response counters they carry:
	// raw_response_leaks_total counts bodies that only the cleanup safety net
	// closed, and a leaked body holds a transport slot for the life of the
	// process, degrading every later fetch against that node. The library owns
	// those counters and only feeds them when its metrics are enabled, so
	// turning them off costs the one signal that makes a missing Close visible.
	opts := *bn.
		DefaultOptions().
		EnablePrometheusMetrics()

	if config.BeaconSubscriptions != nil {
		opts.BeaconSubscription = bn.BeaconSubscriptionOptions{
			Enabled: true,
			Topics:  *config.BeaconSubscriptions,
		}
	} else {
		opts.BeaconSubscription = bn.BeaconSubscriptionOptions{
			Enabled: true,
			Topics:  []string{"block", "chain_reorg"},
		}
	}

	opts.HealthCheck.Interval.Duration = time.Second * 3
	opts.HealthCheck.SuccessfulResponses = 1

	node := bn.NewNode(log, &bn.Config{
		Name:    name,
		Addr:    config.NodeAddress,
		Headers: config.NodeHeaders,
	}, "tracoor_agent", opts)

	metadata := services.NewMetadataService(log, node, overrideNetworkName)

	svcs := []services.Service{
		&metadata,
	}

	return &Node{
		config:   config,
		log:      log.WithField("module", "agent/ethereum/beacon"),
		beacon:   node,
		services: svcs,
	}
}

func (b *Node) GetVersionImmuneBlock(ctx context.Context, blockID string) (*VersionImmuneBlock, error) {
	data, err := b.beacon.FetchRawBlock(ctx, blockID, string(mime.ContentTypeJSON))
	if err != nil {
		return nil, err
	}

	block := &VersionImmuneBlock{}

	if err := json.Unmarshal(data, block); err != nil {
		return nil, err
	}

	return block, nil
}

func (b *Node) Start(ctx context.Context) error {
	s := gocron.NewScheduler(time.Local)

	var (
		// errs carries the first problem that makes this node unusable. It is
		// buffered and sent to once, so reporting a failure never blocks the
		// goroutine that noticed it.
		errs     = make(chan error, 1)
		failed   = make(chan struct{})
		failOnce sync.Once
	)

	fail := func(err error) {
		failOnce.Do(func() {
			errs <- err

			close(failed)
		})
	}

	go func() {
		for _, service := range b.services {
			ready := make(chan struct{})

			var readyOnce sync.Once

			service.OnReady(ctx, func(ctx context.Context) error {
				b.log.WithField("service", service.Name()).Info("Service is ready")

				readyOnce.Do(func() { close(ready) })

				return nil
			})

			service.OnFailure(func(_ context.Context, err error) {
				fail(fmt.Errorf("service %s failed: %w", service.Name(), err))
			})

			b.log.WithField("service", service.Name()).Info("Starting service")

			if err := service.Start(ctx); err != nil {
				fail(fmt.Errorf("failed to start service: %w", err))

				return
			}

			// A service that gave up will never report ready, so the wait ends
			// on its failure as well as on shutdown. Parking here forever would
			// leave the node neither started nor stopped.
			select {
			case <-ready:
			case <-failed:
				return
			case <-ctx.Done():
				return
			}
		}

		b.log.Info("All services are ready")

		for _, callback := range b.onReadyCallbacks {
			if err := callback(ctx); err != nil {
				fail(fmt.Errorf("failed to run on ready callback: %w", err))

				return
			}
		}
	}()

	s.StartAsync()

	if err := b.beacon.Start(ctx); err != nil {
		return err
	}

	select {
	case err := <-errs:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (b *Node) Node() bn.Node {
	return b.beacon
}

func (b *Node) getServiceByName(name services.Name) (services.Service, error) {
	for _, service := range b.services {
		if service.Name() == name {
			return service, nil
		}
	}

	return nil, errors.New("service not found")
}

func (b *Node) Metadata() *services.MetadataService {
	service, err := b.getServiceByName("metadata")
	if err != nil {
		// This should never happen. If it does, good luck.
		return nil
	}

	//nolint:errcheck // casting fine.
	return service.(*services.MetadataService)
}

func (b *Node) OnReady(_ context.Context, callback func(ctx context.Context) error) {
	b.onReadyCallbacks = append(b.onReadyCallbacks, callback)
}
