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
	"github.com/ethpandaops/tracoor/pkg/agent/ethereum"
	"github.com/ethpandaops/tracoor/pkg/agent/indexer"
	"github.com/ethpandaops/tracoor/pkg/networks"
	"github.com/ethpandaops/tracoor/pkg/observability"
	"github.com/ethpandaops/tracoor/pkg/proto/tracoor"
	pIndexer "github.com/ethpandaops/tracoor/pkg/proto/tracoor/indexer"
	"github.com/ethpandaops/tracoor/pkg/store"
	"github.com/go-co-op/gocron"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type agent struct {
	Config *Config

	node *ethereum.Node

	log logrus.FieldLogger

	metrics *Metrics

	scheduler *gocron.Scheduler

	indexer *indexer.Client

	store store.Store

	beaconStateQueue         chan *BeaconStateRequest
	beaconBlockQueue         chan *BeaconBlockRequest
	beaconBadBlockQueue      chan *BeaconBadBlockRequest
	beaconBadBlobQueue       chan *BeaconBadBlobRequest
	executionBlockTraceQueue chan *ExecutionBlockTraceRequest
	executionBadBlockQueue   chan *ExecutionBadBlockRequest
}

const namespace = "tracoor_agent"

func New(ctx context.Context, log logrus.FieldLogger, config *Config) (*agent, error) {
	if config == nil {
		return nil, errors.New("config is required")
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	node := ethereum.NewNode(ctx, log, &config.Ethereum, config.Name)

	indexerClient, err := indexer.NewClient(config.Indexer, log)
	if err != nil {
		return nil, err
	}

	st, err := store.NewStore(namespace, log, config.Store.Type, config.Store.Config, store.DefaultOptions())
	if err != nil {
		return nil, err
	}

	return &agent{
		Config:                   config,
		node:                     node,
		log:                      log,
		metrics:                  GetMetricsInstance(namespace),
		scheduler:                gocron.NewScheduler(time.Local),
		indexer:                  indexerClient,
		store:                    st,
		beaconStateQueue:         make(chan *BeaconStateRequest, 1000),
		beaconBlockQueue:         make(chan *BeaconBlockRequest, 1000),
		beaconBadBlockQueue:      make(chan *BeaconBadBlockRequest, 1000),
		beaconBadBlobQueue:       make(chan *BeaconBadBlobRequest, 1000),
		executionBlockTraceQueue: make(chan *ExecutionBlockTraceRequest, 1000),
		executionBadBlockQueue:   make(chan *ExecutionBadBlockRequest, 1000),
	}, nil
}

func (s *agent) Start(ctx context.Context) error {
	if s.Config.MetricsAddr != "" {
		observability.StartMetricsServer(ctx, s.Config.MetricsAddr)
	}

	if s.Config.PProfAddr != nil {
		if err := s.ServePProf(ctx); err != nil {
			return err
		}
	}

	s.log.
		WithField("version", tracoor.Full()).
		Info("Starting tracoor in agent mode")

	s.node.OnReady(ctx, func(ctx context.Context) error {
		s.log.Info("Ethereum node is ready, setting up beacon and execution events")

		go s.processExecutionBlockTraceQueue(ctx)
		go s.processExecutionBadBlockQueue(ctx)

		s.node.Beacon().Node().OnBlock(ctx, func(ctx context.Context, event *eth2v1.BlockEvent) error {
			logCtx := logrus.WithFields(logrus.Fields{
				"event_slot": event.Slot,
				"event_root": fmt.Sprintf("%#x", event.Block),
				"purpose":    "execution_block_trace",
			})

			if !s.node.Execution().Metadata().IsSynced() {
				logCtx.Debug("Skipping queueing execution block trace as the execution node is not yet synced")

				return nil
			}

			// Fetch the beacon block from the beacon node.
			block, err := s.node.Beacon().Node().FetchBlock(ctx, fmt.Sprintf("%#x", event.Block))
			if err != nil {
				logCtx.WithError(err).Error("Failed to fetch beacon block")

				return err
			}

			if block == nil {
				logCtx.Error("Failed to fetch beacon block - the beacon node returned a nil block.")

				return errors.New("failed to fetch beacon block")
			}

			// Rip out the execution block number from the block
			executionBlockNumber, err := block.ExecutionBlockNumber()
			if err != nil {
				logCtx.WithError(err).Error("Failed to get execution block number from beacon block")

				return err
			}

			executionBlockHash, err := block.ExecutionBlockHash()
			if err != nil {
				logCtx.WithError(err).Error("Failed to get execution block hash from beacon block")

				return err
			}

			s.enqueueExecutionBlockTrace(ctx, executionBlockHash.String(), executionBlockNumber)

			return nil
		})

		return nil
	})

	s.node.Beacon().OnReady(ctx, func(ctx context.Context) error {
		s.log.Info("Beacon node is ready, setting up events that only depend on the beacon node")

		go s.processBeaconStateQueue(ctx)
		go s.processBeaconBlockQueue(ctx)
		go s.processBeaconBadBlockQueue(ctx)
		go s.processBeaconBadBlobQueue(ctx)

		if s.node.Beacon().Metadata().Network.Name == networks.NetworkNameUnknown {
			s.log.Fatal("Unable to determine Ethereum network. Provide an override network name via ethereum.overrideNetworkName")
		}

		s.node.Beacon().Node().OnBlock(ctx, func(ctx context.Context, event *eth2v1.BlockEvent) error {
			if !s.node.Beacon().Metadata().Synced() {
				s.log.Debug("Skipping queueing beacon state from block event as the beacon node is not yet synced")

				return nil
			}

			time.Sleep(2000 * time.Millisecond)

			s.enqueueBeaconState(ctx, event.Slot)
			s.enqueueBeaconBlock(ctx, event.Slot)

			return nil
		})

		s.node.Beacon().Node().OnChainReOrg(ctx, func(ctx context.Context, chainReorg *eth2v1.ChainReorgEvent) error {
			logCtx := logrus.WithFields(
				logrus.Fields{
					"event_old_head_block": rootAsString(chainReorg.OldHeadBlock),
					"event_new_head_block": rootAsString(chainReorg.NewHeadBlock),
					"event_depth":          chainReorg.Depth,
					"purpose":              "chain_reorg",
					"event_slot":           chainReorg.Slot,
				},
			)

			logCtx.WithFields(
				logrus.Fields{
					"event_old_head_state": rootAsString(chainReorg.OldHeadState),
					"event_new_head_state": rootAsString(chainReorg.NewHeadState),
				},
			).Info("Chain reorg detected")

			// Go back and fetch all the new beacon states
			headSlot, _, err := s.node.Beacon().Metadata().Wallclock().Now()
			if err != nil {
				return err
			}

			for slot := chainReorg.Slot; slot < phase0.Slot(headSlot.Number()); slot++ {
				logCtx.WithField("target_slot", slot).Info("Queueing up a fresh beacon state index from reorg event")

				s.enqueueBeaconState(ctx, slot)
				s.enqueueBeaconBlock(ctx, slot)
			}

			// Go back and fetch all the new execution block traces
			for slot := chainReorg.Slot; slot < phase0.Slot(headSlot.Number()); slot++ {
				logCtx := logrus.WithField("target_slot", slot)

				block, err := s.node.Beacon().Node().FetchBlock(ctx, fmt.Sprintf("%d", slot))
				if err != nil {
					logCtx.
						WithError(err).
						Error("Failed to fetch beacon block when processing chain reorg")

					continue
				}

				if block == nil {
					logCtx.Error("Failed to fetch beacon block - the beacon node returned a nil block.")

					continue
				}

				// Rip out the execution block number from the block
				executionBlockNumber, err := block.ExecutionBlockNumber()
				if err != nil {
					logCtx.WithError(err).Error("Failed to get execution block number from beacon block")

					continue
				}

				executionBlockHash, err := block.ExecutionBlockHash()
				if err != nil {
					logCtx.WithError(err).Error("Failed to get execution block hash from beacon block")

					continue
				}

				logCtx.WithFields(logrus.Fields{
					"execution_block_number": executionBlockNumber,
					"execution_block_hash":   executionBlockHash.String(),
				}).Info("Queueing up a fresh execution block trace index after a beacon chain reorg")

				s.enqueueExecutionBlockTrace(ctx, executionBlockHash.String(), executionBlockNumber)
			}

			return nil
		})

		return nil
	})

	if s.Config.Ethereum.OverrideNetworkName != "" {
		s.log.WithField("network", s.Config.Ethereum.OverrideNetworkName).Info("Overriding network name")
	}

	if s.Config.Ethereum.Beacon.InvalidGossipVerifiedBlocksPath != nil {
		_, err := s.scheduler.Every(30).Seconds().Do(func() {
			path := s.Config.Ethereum.Beacon.InvalidGossipVerifiedBlocksPath
			if path != nil {
				s.enqueueBeaconBadBlock(ctx, *path)
			}
		})
		if err != nil {
			return err
		}
	}

	if s.Config.Ethereum.Beacon.InvalidGossipVerifiedBlobsPath != nil {
		_, err := s.scheduler.Every(30).Seconds().Do(func() {
			path := s.Config.Ethereum.Beacon.InvalidGossipVerifiedBlobsPath
			if path != nil {
				s.enqueueBeaconBadBlob(ctx, *path)
			}
		})
		if err != nil {
			return err
		}
	}

	_, err := s.scheduler.Every(90).Seconds().Do(func() {
		s.enqueueExecutionBadBlock(ctx)
	})
	if err != nil {
		return err
	}

	s.scheduler.StartAsync()

	if err := s.performTokenHandshake(ctx); err != nil {
		return err
	}

	if err := s.node.Start(ctx); err != nil {
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
		if err = s.store.SaveStorageHandshakeToken(ctx, s.Config.Name, token); err != nil {
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

func rootAsString(r phase0.Root) string {
	return fmt.Sprintf("%#x", r)
}
