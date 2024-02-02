package services

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/0xsequence/ethkit/ethrpc"
	backoff "github.com/cenkalti/backoff/v4"
	"github.com/ethpandaops/tracoor/pkg/networks"
	"github.com/go-co-op/gocron"
	"github.com/sirupsen/logrus"
)

type MetadataService struct {
	rpc *ethrpc.Provider
	log logrus.FieldLogger

	Network *networks.Network

	onReadyCallbacks []func(context.Context) error

	nodeVersion string

	mu sync.Mutex
}

func NewMetadataService(log logrus.FieldLogger, rpc *ethrpc.Provider) MetadataService {
	return MetadataService{
		rpc:              rpc,
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

func (m *MetadataService) Ready(ctx context.Context) error {
	if m.nodeVersion == "" {
		return errors.New("node version is not available")
	}

	return nil
}

func (m *MetadataService) Web3ClientVersion(ctx context.Context) (string, error) {
	var version string
	call := ethrpc.NewCallBuilder[string]("web3_clientVersion", nil)
	_, err := m.rpc.Do(ctx, call.Into(&version))
	if err != nil {
		return "", err
	}

	return version, nil
}

func (m *MetadataService) RefreshAll(ctx context.Context) error {
	version, err := m.Web3ClientVersion(ctx)
	if err != nil {
		return err
	}

	m.nodeVersion = string(version)

	return nil
}

func (m *MetadataService) Client(ctx context.Context) string {
	return string(ClientFromString(m.nodeVersion))
}

func (m *MetadataService) ClientVersion() string {
	return m.nodeVersion
}
