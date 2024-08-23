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
	opts := *bn.
		DefaultOptions().
		DisablePrometheusMetrics()

	if config.BeaconSubscriptions != nil {
		opts.BeaconSubscription = bn.BeaconSubscriptionOptions{
			Enabled: true,
			Topics:  *config.BeaconSubscriptions,
		}
	} else {
		opts.EnableDefaultBeaconSubscription()
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

	errs := make(chan error, 1)

	go func() {
		wg := sync.WaitGroup{}

		for _, service := range b.services {
			wg.Add(1)

			service.OnReady(ctx, func(ctx context.Context) error {
				b.log.WithField("service", service.Name()).Info("Service is ready")

				wg.Done()

				return nil
			})

			b.log.WithField("service", service.Name()).Info("Starting service")

			if err := service.Start(ctx); err != nil {
				errs <- fmt.Errorf("failed to start service: %w", err)
			}

			wg.Wait()
		}

		b.log.Info("All services are ready")

		for _, callback := range b.onReadyCallbacks {
			if err := callback(ctx); err != nil {
				errs <- fmt.Errorf("failed to run on ready callback: %w", err)
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

	return service.(*services.MetadataService)
}

func (b *Node) OnReady(_ context.Context, callback func(ctx context.Context) error) {
	b.onReadyCallbacks = append(b.onReadyCallbacks, callback)
}

func (b *Node) Synced(ctx context.Context) error {
	status := b.beacon.Status()
	if status == nil {
		return errors.New("missing beacon status")
	}

	syncState := status.SyncState()
	if syncState == nil {
		return errors.New("missing beacon node status sync state")
	}

	if syncState.SyncDistance > 3 {
		return errors.New("beacon node is not synced")
	}

	wallclock := b.Metadata().Wallclock()
	if wallclock == nil {
		return errors.New("missing wallclock")
	}

	currentSlot := wallclock.Slots().Current()

	if currentSlot.Number()-uint64(syncState.HeadSlot) > 64 {
		return fmt.Errorf("beacon node is too far behind head, head slot is %d, current slot is %d", syncState.HeadSlot, currentSlot.Number())
	}

	for _, service := range b.services {
		if err := service.Ready(ctx); err != nil {
			return errors.Wrapf(err, "service %s is not ready", service.Name())
		}
	}

	return nil
}
