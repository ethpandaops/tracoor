package services

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	backoff "github.com/cenkalti/backoff/v4"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethpandaops/tracoor/pkg/networks"
	"github.com/go-co-op/gocron"
	"github.com/sirupsen/logrus"
)

type MetadataService struct {
	eth *ethclient.Client
	log logrus.FieldLogger

	Network *networks.Network

	onReadyCallbacks []func(context.Context) error

	nodeVersion string

	synced bool

	mu sync.Mutex
}

func NewMetadataService(log logrus.FieldLogger, eth *ethclient.Client) MetadataService {
	return MetadataService{
		eth:              eth,
		log:              log.WithField("module", "agent/ethereum/execution/metadata"),
		Network:          &networks.Network{Name: networks.NetworkNameNone},
		onReadyCallbacks: []func(context.Context) error{},
		mu:               sync.Mutex{},
	}
}

func (m *MetadataService) Start(ctx context.Context) error {
	go func() {
		operation := func() error {
			if err := m.RefreshAll(ctx); err != nil {
				return err
			}

			if err := m.Ready(ctx); err != nil {
				return err
			}

			return nil
		}

		if err := backoff.Retry(operation, backoff.NewExponentialBackOff()); err != nil {
			m.log.WithError(err).Warn("Failed to refresh metadata")
		}

		for _, cb := range m.onReadyCallbacks {
			if err := cb(ctx); err != nil {
				m.log.WithError(err).Warn("Failed to execute onReady callback")
			}
		}
	}()

	s := gocron.NewScheduler(time.Local)

	if _, err := s.Every("5m").Do(func() {
		// Create a new context with timeout for each execution
		refreshCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		_ = m.RefreshAll(refreshCtx)
	}); err != nil {
		return err
	}

	if _, err := s.Every("15s").Do(func() {
		// Create a new context with timeout for each execution
		syncCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := m.updateSyncStatus(syncCtx); err != nil {
			m.log.WithError(err).Warn("Failed to update sync status")
		}
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

func (m *MetadataService) Ready(ctx context.Context) error {
	if m.nodeVersion == "" {
		return errors.New("node version is not available")
	}

	return nil
}

func (m *MetadataService) Web3ClientVersion(ctx context.Context) (string, error) {
	var version string

	if err := m.eth.Client().CallContext(ctx, &version, "web3_clientVersion"); err != nil {
		return "", err
	}

	return version, nil
}

func (m *MetadataService) RefreshAll(ctx context.Context) error {
	version, err := m.Web3ClientVersion(ctx)
	if err != nil {
		return err
	}

	m.nodeVersion = version

	return nil
}

func (m *MetadataService) Client(ctx context.Context) string {
	return string(ClientFromString(m.nodeVersion))
}

func (m *MetadataService) ClientVersion() string {
	return m.nodeVersion
}

func (m *MetadataService) updateSyncStatus(ctx context.Context) error {
	status, err := m.eth.SyncProgress(ctx)
	if err != nil {
		// Check for context cancellation first
		if ctx.Err() != nil {
			// Context was canceled, this is expected during shutdown
			return fmt.Errorf("context canceled: %w", err)
		}

		// Log other errors
		m.log.WithError(err).Debug("Failed to get sync status")

		return fmt.Errorf("failed to get sync status: %w", err)
	}

	// Update sync status based on response
	if status == nil {
		m.synced = true
	} else {
		m.synced = false
	}

	return nil
}

func (m *MetadataService) IsSynced() bool {
	return m.synced
}
