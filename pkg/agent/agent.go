package agent

import (
	"context"
	"errors"
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
	"github.com/ethpandaops/tracoor/pkg/store"
	"github.com/go-co-op/gocron"
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

	if err := s.beacon.Start(ctx); err != nil {
		return err
	}

	cancel := make(chan os.Signal, 1)
	signal.Notify(cancel, syscall.SIGTERM, syscall.SIGINT)

	sig := <-cancel
	s.log.Printf("Caught signal: %v", sig)

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
