package agent

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	//nolint:gosec // only exposed if pprofAddr config is set
	_ "net/http/pprof"

	eth2v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/ethwallclock"
	"github.com/ethpandaops/tracoor/pkg/agent/ethereum"
	"github.com/ethpandaops/tracoor/pkg/agent/indexer"
	"github.com/ethpandaops/tracoor/pkg/networks"
	"github.com/ethpandaops/tracoor/pkg/proto/tracoor"
	pIndexer "github.com/ethpandaops/tracoor/pkg/proto/tracoor/indexer"
	"github.com/ethpandaops/tracoor/pkg/store"
	"github.com/go-co-op/gocron"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

type agent struct {
	Config *Config

	beacon *ethereum.BeaconNode

	log logrus.FieldLogger

	metrics *Metrics

	scheduler *gocron.Scheduler

	indexer *indexer.Client

	store store.Store
}

func New(ctx context.Context, log logrus.FieldLogger, config *Config) (*agent, error) {
	if config == nil {
		return nil, errors.New("config is required")
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	beacon, err := ethereum.NewBeaconNode(ctx, config.Name, &config.Ethereum, log)
	if err != nil {
		return nil, err
	}

	indexerClient, err := indexer.NewClient(config.Indexer, log)
	if err != nil {
		return nil, err
	}

	st, err := store.NewStore("store", log, config.Store.Type, config.Store.Config, store.DefaultOptions())
	if err != nil {
		return nil, err
	}

	return &agent{
		Config:    config,
		beacon:    beacon,
		log:       log,
		metrics:   NewMetrics("tracoor_agent"),
		scheduler: gocron.NewScheduler(time.Local),
		indexer:   indexerClient,
		store:     st,
	}, nil
}

//nolint:gocyclo // Needs refactoring
func (s *agent) Start(ctx context.Context) error {
	if err := s.ServeMetrics(ctx); err != nil {
		return err
	}

	if s.Config.PProfAddr != nil {
		if err := s.ServePProf(ctx); err != nil {
			return err
		}
	}

	s.log.
		WithField("version", tracoor.Full()).
		Info("Starting tracoor in agent mode")

	s.beacon.OnReady(ctx, func(ctx context.Context) error {
		s.log.Info("Internal beacon node is ready, subscribing to events")

		if s.beacon.Metadata().Network.Name == networks.NetworkNameUnknown {
			s.log.Fatal("Unable to determine Ethereum network. Provide an override network name via ethereum.overrideNetworkName")
		}

		s.beacon.Metadata().Wallclock().OnSlotChanged(func(slot ethwallclock.Slot) {
			// Fetch the slot for `slot -1`
			slotNumber := slot.Number() - 1

			if slotNumber < 0 {
				slotNumber = 0
			}

			if err := s.fetchAndIndexBeaconState(ctx, phase0.Slot(slotNumber)); err != nil {
				s.log.WithError(err).Error("Failed to fetch and index beacon state")
			}
		})

		s.beacon.Node().OnChainReOrg(ctx, func(ctx context.Context, chainReorg *eth2v1.ChainReorgEvent) error {
			s.log.WithFields(
				logrus.Fields{
					"old_head_state": chainReorg.OldHeadState,
					"new_head_state": chainReorg.NewHeadState,
					"depth":          chainReorg.Depth,
					"slot":           chainReorg.Slot,
					"old_head_block": chainReorg.OldHeadBlock,
					"new_head_block": chainReorg.NewHeadBlock,
				},
			).Info("Chain reorg detected")

			// Go back and fetch all the new beacon states
			headSlot, _, err := s.beacon.Metadata().Wallclock().Now()
			if err != nil {
				return err
			}

			for slot := chainReorg.Slot; slot < phase0.Slot(headSlot.Number()); slot++ {
				s.log.WithField("slot", slot).Info("Fetching and indexing beacon state from reorg event")

				if err := s.fetchAndIndexBeaconState(ctx, phase0.Slot(slot)); err != nil {
					s.log.WithError(err).Error("Failed to fetch and index beacon state from reorg")
				}
			}

			return nil
		})

		return nil
	})

	if s.Config.Ethereum.OverrideNetworkName != "" {
		s.log.WithField("network", s.Config.Ethereum.OverrideNetworkName).Info("Overriding network name")
	}

	if err := s.performTokenHandshake(ctx); err != nil {
		return err
	}

	if err := s.beacon.Start(ctx); err != nil {
		return err
	}

	cancel := make(chan os.Signal, 1)
	signal.Notify(cancel, syscall.SIGTERM, syscall.SIGINT)

	sig := <-cancel
	s.log.Printf("Caught signal: %v", sig)

	return nil
}

func (s *agent) performTokenHandshake(ctx context.Context) error {
	s.log.Info("Performing token handshake")

	// Perform a token handshake with the indexer to ensure we're connected to the same store.
	// This is important for the indexer to be able to find the beacon states we upload.

	// First check the store. If the token already exists, download it and use it.
	token := uuid.New().String()

	exists, err := s.store.StorageHandshakeTokenExists(ctx, s.Config.Name)
	if err != nil {
		return fmt.Errorf("failed to check if storage handshake token exists: %w", err)
	}

	if exists {
		token, err = s.store.GetStorageHandshakeToken(ctx, s.Config.Name)
		if err != nil {
			return fmt.Errorf("failed to get storage handshake token: %w", err)
		}

		s.log.WithField("token", token).Debug("Storage handshake token already exists")
	} else {
		// Save the token to the store
		if err := s.store.SaveStorageHandshakeToken(ctx, s.Config.Name, token); err != nil {
			return fmt.Errorf("failed to save storage handshake token: %w", err)
		}

		// Sleep for a bit to give the store time to update
		time.Sleep(500 * time.Millisecond)
	}

	// Perform the handshake with the indexer
	rsp, err := s.indexer.GetStorageHandshakeToken(ctx, &pIndexer.GetStorageHandshakeTokenRequest{
		Node:  s.Config.Name,
		Token: token,
	})
	if err != nil {
		return fmt.Errorf("failed to get storage handshake token from indexer: %w", err)
	}

	if rsp.Token != token {
		return fmt.Errorf("storage handshake token mismatch: %s (ours) != %s (theirs)", token, rsp.Token)
	}

	s.log.Info("Storage handshake complete 🤝 - we are connected to the same storage backend as the indexer")

	return nil
}

func (s *agent) ServeMetrics(ctx context.Context) error {
	go func() {
		sm := http.NewServeMux()
		sm.Handle("/metrics", promhttp.Handler())

		server := &http.Server{
			Addr:              s.Config.MetricsAddr,
			ReadHeaderTimeout: 15 * time.Second,
			Handler:           sm,
		}

		s.log.Infof("Serving metrics at %s", s.Config.MetricsAddr)

		if err := server.ListenAndServe(); err != nil {
			s.log.Fatal(err)
		}
	}()

	return nil
}

func (s *agent) ServePProf(ctx context.Context) error {
	pprofServer := &http.Server{
		Addr:              *s.Config.PProfAddr,
		ReadHeaderTimeout: 120 * time.Second,
	}

	go func() {
		s.log.Infof("Serving pprof at %s", *s.Config.PProfAddr)

		if err := pprofServer.ListenAndServe(); err != nil {
			s.log.Fatal(err)
		}
	}()

	return nil
}
