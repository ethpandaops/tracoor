package agent

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	//nolint:gosec // only exposed if pprofAddr config is set
	_ "net/http/pprof"

	eth2v1 "github.com/ethpandaops/go-eth2-client/api/v1"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/tracoor/pkg/agent/ethereum"
	"github.com/ethpandaops/tracoor/pkg/agent/ethereum/execution"
	"github.com/ethpandaops/tracoor/pkg/agent/indexer"
	"github.com/ethpandaops/tracoor/pkg/compression"
	"github.com/ethpandaops/tracoor/pkg/networks"
	"github.com/ethpandaops/tracoor/pkg/observability"
	"github.com/ethpandaops/tracoor/pkg/proto/tracoor"
	pIndexer "github.com/ethpandaops/tracoor/pkg/proto/tracoor/indexer"
	"github.com/ethpandaops/tracoor/pkg/store"
	"github.com/go-co-op/gocron"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// indexerClient is the part of the indexer the agent actually calls. It is
// named here rather than in the client package so the fetch paths can be
// exercised without a server on the other end.
type indexerClient interface {
	CreateBeaconState(ctx context.Context, req *pIndexer.CreateBeaconStateRequest) (*pIndexer.CreateBeaconStateResponse, error)
	ListBeaconState(ctx context.Context, req *pIndexer.ListBeaconStateRequest) (*pIndexer.ListBeaconStateResponse, error)
	CreateBeaconBlock(ctx context.Context, req *pIndexer.CreateBeaconBlockRequest) (*pIndexer.CreateBeaconBlockResponse, error)
	ListBeaconBlock(ctx context.Context, req *pIndexer.ListBeaconBlockRequest) (*pIndexer.ListBeaconBlockResponse, error)
	CreateExecutionPayloadEnvelope(ctx context.Context, req *pIndexer.CreateExecutionPayloadEnvelopeRequest) (*pIndexer.CreateExecutionPayloadEnvelopeResponse, error)
	ListExecutionPayloadEnvelope(ctx context.Context, req *pIndexer.ListExecutionPayloadEnvelopeRequest) (*pIndexer.ListExecutionPayloadEnvelopeResponse, error)
	CreateBeaconBadBlock(ctx context.Context, req *pIndexer.CreateBeaconBadBlockRequest) (*pIndexer.CreateBeaconBadBlockResponse, error)
	ListBeaconBadBlock(ctx context.Context, req *pIndexer.ListBeaconBadBlockRequest) (*pIndexer.ListBeaconBadBlockResponse, error)
	CreateBeaconBadBlob(ctx context.Context, req *pIndexer.CreateBeaconBadBlobRequest) (*pIndexer.CreateBeaconBadBlobResponse, error)
	ListBeaconBadBlob(ctx context.Context, req *pIndexer.ListBeaconBadBlobRequest) (*pIndexer.ListBeaconBadBlobResponse, error)
	CreateExecutionBlockTrace(ctx context.Context, req *pIndexer.CreateExecutionBlockTraceRequest) (*pIndexer.CreateExecutionBlockTraceResponse, error)
	ListExecutionBlockTrace(ctx context.Context, req *pIndexer.ListExecutionBlockTraceRequest) (*pIndexer.ListExecutionBlockTraceResponse, error)
	CreateExecutionBadBlock(ctx context.Context, req *pIndexer.CreateExecutionBadBlockRequest) (*pIndexer.CreateExecutionBadBlockResponse, error)
	ListExecutionBadBlock(ctx context.Context, req *pIndexer.ListExecutionBadBlockRequest) (*pIndexer.ListExecutionBadBlockResponse, error)
	GetStorageHandshakeToken(ctx context.Context, req *pIndexer.GetStorageHandshakeTokenRequest) (*pIndexer.GetStorageHandshakeTokenResponse, error)
	GetBlob(ctx context.Context, req *pIndexer.GetBlobRequest) (*pIndexer.GetBlobResponse, error)
	CreateBlob(ctx context.Context, req *pIndexer.CreateBlobRequest) (*pIndexer.CreateBlobResponse, error)
	CreatePayloadDivergence(ctx context.Context, req *pIndexer.CreatePayloadDivergenceRequest) (*pIndexer.CreatePayloadDivergenceResponse, error)
}

var _ indexerClient = (*indexer.Client)(nil)

type agent struct {
	Config *Config

	node *ethereum.Node

	log logrus.FieldLogger

	metrics *Metrics

	scheduler *gocron.Scheduler

	indexer indexerClient

	store store.Store

	beaconStateQueue              chan *BeaconStateRequest
	beaconBlockQueue              chan *BeaconBlockRequest
	executionPayloadEnvelopeQueue chan *ExecutionPayloadEnvelopeRequest
	beaconBadBlockQueue           chan *BeaconBadBlockRequest
	beaconBadBlobQueue            chan *BeaconBadBlobRequest
	executionBlockTraceQueue      chan *ExecutionBlockTraceRequest
	executionBadBlockQueue        chan *ExecutionBadBlockRequest

	compressor *compression.Compressor

	breaker *circuitBreaker

	// pending collapses duplicate work: a reorg re-derives the same slots for
	// every artifact kind, and the queues block on send.
	pending *pendingItems

	// workers tracks every background loop the agent owns so a shutdown can
	// wait for them instead of abandoning them mid-fetch.
	workers workerGroup
}

const (
	namespace = "tracoor_agent"

	logKeyPurpose = "purpose"
	logKeySlot    = "slot"
	labelAgent    = "agent"
	labelQueue    = "queue"
	labelReason   = "reason"
)

func New(ctx context.Context, log logrus.FieldLogger, config *Config) (*agent, error) {
	if config == nil {
		return nil, errors.New("config is required")
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	node := ethereum.NewNode(ctx, log, &config.Ethereum, config.Name, config.Ethereum.SyncToleranceSlots)

	indexerClient, err := indexer.NewClient(config.Indexer, log)
	if err != nil {
		return nil, err
	}

	st, err := store.NewStore(namespace, log, config.Store.Type, config.Store.Config, store.DefaultOptions())
	if err != nil {
		return nil, err
	}

	return &agent{
		Config:                        config,
		node:                          node,
		log:                           log,
		metrics:                       GetMetricsInstance(namespace),
		scheduler:                     gocron.NewScheduler(time.Local),
		indexer:                       indexerClient,
		store:                         st,
		beaconStateQueue:              make(chan *BeaconStateRequest, 1000),
		beaconBlockQueue:              make(chan *BeaconBlockRequest, 1000),
		executionPayloadEnvelopeQueue: make(chan *ExecutionPayloadEnvelopeRequest, 1000),
		beaconBadBlockQueue:           make(chan *BeaconBadBlockRequest, 1000),
		beaconBadBlobQueue:            make(chan *BeaconBadBlobRequest, 1000),
		executionBlockTraceQueue:      make(chan *ExecutionBlockTraceRequest, 1000),
		executionBadBlockQueue:        make(chan *ExecutionBadBlockRequest, 1000),
		compressor:                    compression.NewCompressor(),
		breaker:                       newCircuitBreaker(),
		pending:                       newPendingItems(),
	}, nil
}

func (s *agent) Start(ctx context.Context) error {
	// Everything this agent owns hangs off a context of its own so that a node
	// that becomes unusable can stop this agent's workers without disturbing
	// whatever else shares the process.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	if s.Config.MetricsAddr != "" {
		observability.StartMetricsServer(ctx, s.Config.MetricsAddr)
	}

	if s.Config.PProfAddr != nil {
		if err := s.ServePProf(ctx); err != nil {
			return err
		}
	}

	enabledFeatures := s.Config.Ethereum.Features.EnabledFlags()

	s.log.
		WithField("version", tracoor.Full()).
		WithField("enabled_features", strings.Join(enabledFeatures, ", ")).
		Info("Starting tracoor in agent mode")

	s.node.OnReady(ctx, func(ctx context.Context) error {
		s.log.Info("Ethereum node is ready, setting up beacon and execution events")

		s.workers.Go(func() { s.processExecutionBlockTraceQueue(ctx) })
		s.workers.Go(func() { s.processExecutionBadBlockQueue(ctx) })

		s.node.Beacon().Node().OnBlock(ctx, func(ctx context.Context, event *eth2v1.BlockEvent) error {
			if !s.Config.Ethereum.Features.GetFetchExecutionBlockTrace() {
				return nil
			}

			logCtx := s.log.WithFields(logrus.Fields{
				"event_slot":  event.Slot,
				"event_root":  fmt.Sprintf("%#x", event.Block),
				logKeyPurpose: "execution_block_trace",
			})

			// Check if the block is too old to bother fetching.
			ignore, err := s.node.ShouldIgnoreEventFromSlot(event.Slot)
			if err != nil {
				logCtx.WithError(err).Error("Failed to check if the block is too old to fetch")

				return err
			}

			if ignore {
				logCtx.Debug("Skipping queueing execution block trace as the block is too old to fetch")

				return nil
			}

			s.enqueueExecutionBlockTrace(ctx, fmt.Sprintf("%#x", event.Block))

			return nil
		})

		return nil
	})

	s.node.Beacon().OnReady(ctx, func(ctx context.Context) error {
		s.log.Info("Beacon node is ready, setting up events that only depend on the beacon node")

		// Everything this agent records is addressed by network, so without one
		// there is nothing worth starting. This agent stops; the process, and
		// every other agent in it, is none the wiser.
		if s.node.Beacon().Metadata().Network.Name == networks.NetworkNameUnknown {
			return errors.New("unable to determine ethereum network, provide an override network name via ethereum.overrideNetworkName")
		}

		s.workers.Go(func() { s.processBeaconStateQueue(ctx) })
		s.workers.Go(func() { s.processBeaconBlockQueue(ctx) })
		s.workers.Go(func() { s.processExecutionPayloadEnvelopeQueue(ctx) })
		s.workers.Go(func() { s.processBeaconBadBlockQueue(ctx) })
		s.workers.Go(func() { s.processBeaconBadBlobQueue(ctx) })

		s.node.Beacon().Node().OnBlock(ctx, func(ctx context.Context, event *eth2v1.BlockEvent) error {
			logCtx := s.log.WithFields(logrus.Fields{
				"event_topic": "block",
				logKeySlot:    event.Slot,
				"root":        fmt.Sprintf("%#x", event.Block),
				logKeyPurpose: "beacon_state_and_block",
			})

			if ignore, err := s.node.ShouldIgnoreEventFromSlot(event.Slot); err != nil {
				logCtx.WithError(err).Error("Failed to check if the block is too old to fetch")

				return err
			} else if ignore {
				logCtx.Debug("Skipping queueing beacon state and block as the block is too old to fetch")

				return nil
			}

			time.Sleep(2000 * time.Millisecond)

			s.enqueueBeaconState(ctx, event.Slot)
			s.enqueueBeaconBlock(ctx, event.Slot)
			s.enqueueExecutionPayloadEnvelope(ctx, event.Slot)

			return nil
		})

		s.node.Beacon().Node().OnChainReOrg(ctx, func(ctx context.Context, chainReorg *eth2v1.ChainReorgEvent) error {
			logCtx := s.log.WithFields(
				logrus.Fields{
					"event_old_head_block": rootAsString(chainReorg.OldHeadBlock),
					"event_new_head_block": rootAsString(chainReorg.NewHeadBlock),
					"event_depth":          chainReorg.Depth,
					logKeyPurpose:          "chain_reorg",
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
				s.enqueueExecutionPayloadEnvelope(ctx, slot)
			}

			// Go back and fetch all the new execution block traces
			for slot := chainReorg.Slot; slot < phase0.Slot(headSlot.Number()); slot++ {
				if !s.Config.Ethereum.Features.GetFetchExecutionBlockTrace() {
					continue
				}

				logrus.
					WithField("target_slot", slot).
					Info("Queueing up a fresh execution block trace index after a beacon chain reorg")

				s.enqueueExecutionBlockTrace(ctx, fmt.Sprintf("%d", slot))
			}

			return nil
		})

		return nil
	})

	if s.Config.Ethereum.OverrideNetworkName != "" {
		s.log.WithField("network", s.Config.Ethereum.OverrideNetworkName).Info("Overriding network name")
	}

	if s.Config.Ethereum.Features.GetFetchBeaconBadBlock() && s.Config.Ethereum.Beacon.InvalidGossipVerifiedBlocksPath != nil {
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

	if s.Config.Ethereum.Features.GetFetchBeaconBadBlob() && s.Config.Ethereum.Beacon.InvalidGossipVerifiedBlobsPath != nil {
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

	var runErr error

	select {
	case <-ctx.Done():
		s.log.Info("Shutting down tracoor agent")
	case err := <-s.node.Failed():
		// The node this agent exists to watch is unusable. Only this agent
		// stops: whatever else shares the process still has healthy nodes of
		// its own to serve.
		s.log.WithError(err).Error("Ethereum node is unusable, stopping this agent")

		runErr = err

		// The workers unwind on cancellation, which the failure did not do for
		// us the way a shutdown would have.
		cancel()
	}

	s.shutdown(ctx)

	return runErr
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

		// pprof is a debugging convenience. Losing it costs a diagnostic, so it
		// is not worth a single artifact, let alone the process.
		if err := pprofServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.log.WithError(err).Error("Failed to serve pprof")
		}
	}()

	go func() {
		<-ctx.Done()

		// The shutdown is triggered by ctx being cancelled, so it cannot itself
		// run under ctx.
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), pprofShutdownTimeout)
		defer cancel()

		if err := pprofServer.Shutdown(shutdownCtx); err != nil {
			s.log.WithError(err).Debug("Failed to stop the pprof server")
		}
	}()

	return nil
}

const executionBlockResolveTimeout = 30 * time.Second

// resolveExecutionBlock resolves the execution block a beacon block commits to.
// Pre-gloas blocks embed the execution payload, so the hash and number are read
// straight off the block. Gloas (ePBS) blocks only commit to the payload's
// block hash via the bid, so the number comes from the execution node, which
// only knows it once the builder has revealed the payload. This runs on the
// queue worker rather than the beacon event callback: a node that is slow to
// reveal must not stall event processing for every other artifact.
func (s *agent) resolveExecutionBlock(ctx context.Context, blockID string) (hash string, number uint64, err error) {
	ctx, cancel := context.WithTimeout(ctx, executionBlockResolveTimeout)
	defer cancel()

	block, err := s.node.Beacon().GetVersionImmuneBlock(ctx, blockID)
	if err != nil {
		return "", 0, fmt.Errorf("failed to fetch beacon block: %w", err)
	}

	if block == nil {
		return "", 0, errors.New("beacon node returned a nil block")
	}

	payload := block.Data.Message.Body.ExecutionPayload
	if payload.BlockHash != "" {
		blockNumber, perr := strconv.ParseUint(payload.BlockNumber, 10, 64)
		if perr != nil {
			return "", 0, fmt.Errorf("failed to parse execution block number: %w", perr)
		}

		return payload.BlockHash, blockNumber, nil
	}

	bidBlockHash := block.Data.Message.Body.SignedExecutionPayloadBid.Message.BlockHash
	if bidBlockHash == "" {
		return "", 0, errors.New("beacon block contains no execution payload or execution payload bid")
	}

	blockNumber, err := s.node.Execution().GetBlockNumberByHash(ctx, bidBlockHash)
	if err != nil {
		if errors.Is(err, execution.ErrBlockNotFound) {
			return "", 0, fmt.Errorf("%w: execution block %s has not been revealed", errItemNotAvailable, bidBlockHash)
		}

		return "", 0, fmt.Errorf("failed to resolve execution block number for bid block hash %s: %w", bidBlockHash, err)
	}

	return bidBlockHash, blockNumber, nil
}

// executionBlockTraceStale reports whether the execution node has moved so far
// past this block that it can no longer re-execute it.
func (s *agent) executionBlockTraceStale(ctx context.Context, blockNumber uint64) bool {
	ctx, cancel := context.WithTimeout(ctx, executionBlockResolveTimeout)
	defer cancel()

	head, err := s.node.Execution().BlockNumber(ctx)
	if err != nil || head <= blockNumber {
		return false
	}

	return head-blockNumber > s.Config.Ethereum.GetExecutionBlockTraceAgeThresholdBlocks()
}

func rootAsString(r phase0.Root) string {
	return fmt.Sprintf("%#x", r)
}
