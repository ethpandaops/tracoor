package services

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/ethpandaops/beacon/pkg/beacon"
	"github.com/ethpandaops/beacon/pkg/beacon/state"
	"github.com/ethpandaops/ethwallclock"
	v1 "github.com/ethpandaops/go-eth2-client/api/v1"
	"github.com/ethpandaops/tracoor/pkg/networks"
	"github.com/go-co-op/gocron"
	"github.com/sirupsen/logrus"
)

type MetadataService struct {
	beacon beacon.Node
	log    logrus.FieldLogger

	Network *networks.Network

	Genesis *v1.Genesis
	Spec    *state.Spec

	wallclock *ethwallclock.EthereumBeaconChain

	onReadyCallbacks []func(context.Context) error

	onFailureCallbacks []FailureHandler

	overrideNetworkName string

	mu sync.Mutex
}

func NewMetadataService(log logrus.FieldLogger, sbeacon beacon.Node, overrideNetworkName string) MetadataService {
	return MetadataService{
		beacon:              sbeacon,
		log:                 log.WithField("module", "agent/ethereum/beacon/metadata"),
		Network:             &networks.Network{Name: networks.NetworkNameNone},
		onReadyCallbacks:    []func(context.Context) error{},
		onFailureCallbacks:  []FailureHandler{},
		mu:                  sync.Mutex{},
		overrideNetworkName: overrideNetworkName,
	}
}

func (m *MetadataService) Start(ctx context.Context) error {
	go func() {
		operation := func() error {
			if !m.beacon.Healthy() {
				m.log.Info("Waiting for beacon node to be healthy")

				if err := m.WaitForHealthyBeaconNode(ctx); err != nil {
					return err
				}
			}

			if err := m.RefreshAll(ctx); err != nil {
				return err
			}

			if err := m.Ready(ctx); err != nil {
				return err
			}

			return nil
		}

		// A node that never gives up its metadata is a broken node, not a
		// broken process: the service reports the failure and stops, and the
		// owner of this node decides what happens next. Nothing here may end
		// the process, which in single mode is also every other node's agent
		// and the server they all index through.
		if err := backoff.Retry(operation, backoff.WithContext(backoff.NewExponentialBackOff(), ctx)); err != nil {
			if ctx.Err() != nil {
				return
			}

			m.log.WithError(err).Error("Gave up waiting for beacon node metadata")

			m.reportFailure(ctx, err)

			return
		}

		for _, cb := range m.onReadyCallbacks {
			if err := cb(ctx); err != nil {
				m.log.WithError(err).Warn("Failed to execute onReady callback")
			}
		}
	}()

	s := gocron.NewScheduler(time.Local)

	if _, err := s.Every("5m").Do(func() {
		_ = m.RefreshAll(ctx)
	}); err != nil {
		return err
	}

	s.StartAsync()

	return nil
}

func (m *MetadataService) Name() Name {
	return "metadata"
}

func (m *MetadataService) Stop(ctx context.Context) error {
	return nil
}

func (m *MetadataService) OnReady(ctx context.Context, cb func(context.Context) error) {
	m.onReadyCallbacks = append(m.onReadyCallbacks, cb)
}

func (m *MetadataService) OnFailure(cb FailureHandler) {
	m.onFailureCallbacks = append(m.onFailureCallbacks, cb)
}

// reportFailure hands the reason this service stopped to whoever registered for
// it. Callbacks are registered before Start, so this is only ever read here.
func (m *MetadataService) reportFailure(ctx context.Context, err error) {
	for _, cb := range m.onFailureCallbacks {
		cb(ctx, err)
	}
}

// WaitForHealthyBeaconNode blocks until the node reports itself healthy. It
// returns an error rather than ending the process: an unhealthy node is one
// node's problem, and the caller is the one that knows what depends on it.
func (m *MetadataService) WaitForHealthyBeaconNode(ctx context.Context) error {
	operation := func() error {
		if !m.beacon.Healthy() {
			return errors.New("beacon node is not healthy")
		}

		return nil
	}

	if err := backoff.Retry(operation, backoff.WithContext(backoff.NewExponentialBackOff(), ctx)); err != nil {
		m.log.WithError(err).Warn("Failed to wait for healthy beacon node")

		return fmt.Errorf("beacon node did not become healthy: %w", err)
	}

	return nil
}

func (m *MetadataService) Ready(ctx context.Context) error {
	if m.Genesis == nil {
		return errors.New("genesis is not available")
	}

	if m.Spec == nil {
		return errors.New("spec is not available")
	}

	if m.NodeVersion(context.Background()) == "" {
		return errors.New("node version is not available")
	}

	if m.Network.Name == networks.NetworkNameNone {
		return errors.New("network name is not available")
	}

	if m.wallclock == nil {
		return errors.New("wallclock is not available")
	}

	return nil
}

func (m *MetadataService) RefreshAll(ctx context.Context) error {
	if err := m.fetchSpec(ctx); err != nil {
		m.log.WithError(err).Error("Failed to fetch spec for refresh")
	}

	if err := m.fetchGenesis(ctx); err != nil {
		m.log.WithError(err).Error("Failed to fetch genesis for refresh")
	}

	if m.Genesis != nil && m.Spec != nil && m.wallclock == nil {
		if newWallclock := ethwallclock.NewEthereumBeaconChain(m.Genesis.GenesisTime, m.Spec.SecondsPerSlot.AsDuration(), uint64(m.Spec.SlotsPerEpoch)); newWallclock != nil {
			m.mu.Lock()

			m.wallclock = newWallclock

			m.mu.Unlock()
		}
	}

	if err := m.DeriveNetwork(ctx); err != nil {
		m.log.WithError(err).Warn("Failed to derive network name for refresh")
	}

	return nil
}

func (m *MetadataService) Wallclock() *ethwallclock.EthereumBeaconChain {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.wallclock
}

func (m *MetadataService) DeriveNetwork(_ context.Context) error {
	if m.Genesis == nil {
		return errors.New("genesis is not available")
	}

	network := networks.DeriveFromGenesisRoot(fmt.Sprintf("%#x", m.Genesis.GenesisValidatorsRoot))

	if m.overrideNetworkName != "" {
		network.Name = networks.NetworkName(m.overrideNetworkName)

		network.ID = m.Spec.DepositChainID
	}

	if network.Name != m.Network.Name {
		m.log.WithFields(logrus.Fields{
			"name": network.Name,
			"id":   network.ID,
		}).Info("Detected ethereum network")
	}

	m.Network = network

	return nil
}

func (m *MetadataService) fetchSpec(_ context.Context) error {
	m.log.Debug("Fetching spec from beacon node")

	spec, err := m.beacon.Spec()
	if err != nil {
		m.log.WithError(err).Error("Failed to fetch spec from beacon node")

		return err
	}

	m.Spec = spec

	m.log.Debug("Successfully fetched spec from beacon node")

	return nil
}

func (m *MetadataService) fetchGenesis(_ context.Context) error {
	m.log.Debug("Fetching genesis from beacon node")

	genesis, err := m.beacon.Genesis()
	if err != nil {
		m.log.WithError(err).Error("Failed to fetch genesis from beacon node")

		return err
	}

	m.Genesis = genesis

	m.log.Debug("Successfully fetched genesis from beacon node")

	return nil
}

func (m *MetadataService) NodeVersion(_ context.Context) string {
	version, _ := m.beacon.NodeVersion()

	return version
}

func (m *MetadataService) Client(ctx context.Context) string {
	return string(ClientFromString(m.NodeVersion(ctx)))
}
