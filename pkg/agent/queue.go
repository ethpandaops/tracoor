package agent

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/ethpandaops/go-eth2-client/spec/phase0"
)

// Each request carries the pending-set key its producer claimed, so the worker
// that finishes it releases exactly that claim rather than deriving the key a
// second time.

type BeaconStateRequest struct {
	Slot phase0.Slot

	key string
}

type BeaconBlockRequest struct {
	Slot phase0.Slot

	key string
}

type ExecutionPayloadEnvelopeRequest struct {
	Slot phase0.Slot

	key string
}

type BeaconBadBlockRequest struct {
	Path string

	key string
}

type BeaconBadBlobRequest struct {
	Path string

	key string
}

// ExecutionBlockTraceRequest identifies the beacon block whose execution
// payload should be traced. It carries no execution block hash or number:
// resolving those can block on a builder revealing a payload, so it happens on
// the queue worker rather than in the beacon event callback that queues it.
type ExecutionBlockTraceRequest struct {
	BlockID string

	key string
}

type ExecutionBadBlockRequest struct {
	key string
}

// enqueue hands an item to a queue, giving up if the agent is shutting down.
// The queues block when full, so without the cancellation case a shutdown would
// be held open by whichever producer happened to be mid-send. It reports
// whether the item was accepted.
func enqueue[T any](ctx context.Context, queue chan<- T, item T) bool {
	select {
	case queue <- item:
		return true
	case <-ctx.Done():
		return false
	}
}

// drainQueue runs handler over a queue until it closes or the agent is shutting
// down. Cancellation is checked ahead of the queue so a backlog cannot hold a
// shutdown open: anything left behind is re-derived from the chain next time
// there is a reason to.
func drainQueue[T any](ctx context.Context, queue <-chan T, handler func(item T)) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		select {
		case <-ctx.Done():
			return
		case item, ok := <-queue:
			if !ok {
				return
			}

			handler(item)
		}
	}
}

func (s *agent) enqueueBeaconState(ctx context.Context, slot phase0.Slot) {
	if !s.Config.Ethereum.Features.GetFetchBeaconState() || !s.allowArtifact(BeaconStateQueue) {
		return
	}

	key := pendingKey(BeaconStateQueue, slotIdentifier(slot))
	if !s.claimQueueItem(BeaconStateQueue, key) {
		return
	}

	if !enqueue(ctx, s.beaconStateQueue, &BeaconStateRequest{Slot: slot, key: key}) {
		s.releaseQueueItem(key)
	}
}

func (s *agent) enqueueBeaconBlock(ctx context.Context, slot phase0.Slot) {
	if !s.Config.Ethereum.Features.GetFetchBeaconBlock() || !s.allowArtifact(BeaconBlockQueue) {
		return
	}

	key := pendingKey(BeaconBlockQueue, slotIdentifier(slot))
	if !s.claimQueueItem(BeaconBlockQueue, key) {
		return
	}

	if !enqueue(ctx, s.beaconBlockQueue, &BeaconBlockRequest{Slot: slot, key: key}) {
		s.releaseQueueItem(key)
	}
}

func (s *agent) enqueueExecutionPayloadEnvelope(ctx context.Context, slot phase0.Slot) {
	if !s.Config.Ethereum.Features.GetFetchExecutionPayloadEnvelope() || !s.allowArtifact(ExecutionPayloadEnvelopeQueue) {
		return
	}

	key := pendingKey(ExecutionPayloadEnvelopeQueue, slotIdentifier(slot))
	if !s.claimQueueItem(ExecutionPayloadEnvelopeQueue, key) {
		return
	}

	if !enqueue(ctx, s.executionPayloadEnvelopeQueue, &ExecutionPayloadEnvelopeRequest{Slot: slot, key: key}) {
		s.releaseQueueItem(key)
	}
}

func (s *agent) enqueueBeaconBadBlock(ctx context.Context, path string) {
	if !s.Config.Ethereum.Features.GetFetchBeaconBadBlock() {
		return
	}

	key := pendingKey(BeaconBadBlockQueue, path)
	if !s.claimQueueItem(BeaconBadBlockQueue, key) {
		return
	}

	if !enqueue(ctx, s.beaconBadBlockQueue, &BeaconBadBlockRequest{Path: path, key: key}) {
		s.releaseQueueItem(key)
	}
}

func (s *agent) enqueueBeaconBadBlob(ctx context.Context, path string) {
	if !s.Config.Ethereum.Features.GetFetchBeaconBadBlob() {
		return
	}

	key := pendingKey(BeaconBadBlobQueue, path)
	if !s.claimQueueItem(BeaconBadBlobQueue, key) {
		return
	}

	if !enqueue(ctx, s.beaconBadBlobQueue, &BeaconBadBlobRequest{Path: path, key: key}) {
		s.releaseQueueItem(key)
	}
}

func (s *agent) enqueueExecutionBlockTrace(ctx context.Context, blockID string) {
	if !s.Config.Ethereum.Features.GetFetchExecutionBlockTrace() || !s.allowArtifact(ExecutionBlockTraceQueue) {
		return
	}

	key := pendingKey(ExecutionBlockTraceQueue, blockID)
	if !s.claimQueueItem(ExecutionBlockTraceQueue, key) {
		return
	}

	if !enqueue(ctx, s.executionBlockTraceQueue, &ExecutionBlockTraceRequest{BlockID: blockID, key: key}) {
		s.releaseQueueItem(key)
	}
}

func (s *agent) enqueueExecutionBadBlock(ctx context.Context) {
	if !s.Config.Ethereum.Features.GetFetchExecutionBadBlock() {
		return
	}

	// The execution node only ever reports its current bad blocks, so there is
	// one item to have outstanding, not one per identifier.
	key := pendingKey(ExecutionBadBlockQueue, "")
	if !s.claimQueueItem(ExecutionBadBlockQueue, key) {
		return
	}

	if !enqueue(ctx, s.executionBadBlockQueue, &ExecutionBadBlockRequest{key: key}) {
		s.releaseQueueItem(key)
	}
}

func slotIdentifier(slot phase0.Slot) string {
	return strconv.FormatUint(uint64(slot), 10)
}

func (s *agent) processBeaconStateQueue(ctx context.Context) {
	if !s.Config.Ethereum.Features.GetFetchBeaconState() {
		return
	}

	drainQueue(ctx, s.beaconStateQueue, func(stateRequest *BeaconStateRequest) {
		defer s.releaseQueueItem(stateRequest.key)

		s.metrics.SetQueueSize(BeaconStateQueue, len(s.beaconStateQueue), s.Config.Name)

		start := time.Now()

		_, nowEpoch, err := s.node.Beacon().Metadata().Wallclock().Now()
		if err != nil {
			s.log.WithError(err).Error("Failed to get current time")

			return
		}

		targetEpoch := s.node.Beacon().Metadata().Wallclock().Epochs().FromSlot(uint64(stateRequest.Slot))
		targetEpochNumber := targetEpoch.Number()

		logCtx := s.log.WithField("slot", stateRequest.Slot)

		// If the slot is older than the allowed number of epochs we'll skip it.
		if nowEpoch.Number()-targetEpochNumber > s.Config.Ethereum.BeaconStateAgeThresholdEpochs {
			s.metrics.IncrementItemSkipped(BeaconStateQueue, s.Config.Name)
			s.metrics.IncrementItemDropped(BeaconStateQueue, s.Config.Name, dropReasonStale)
		} else {
			s.runQueueItem(ctx, BeaconStateQueue, logCtx, func(ctx context.Context) error {
				return s.fetchAndIndexBeaconState(ctx, stateRequest.Slot)
			})
		}

		s.metrics.ObserveQueueItemProcessingTime(
			BeaconStateQueue,
			time.Since(start),
			s.Config.Name,
		)
	})
}

func (s *agent) processBeaconBlockQueue(ctx context.Context) {
	if !s.Config.Ethereum.Features.GetFetchBeaconBlock() {
		return
	}

	drainQueue(ctx, s.beaconBlockQueue, func(blockRequest *BeaconBlockRequest) {
		defer s.releaseQueueItem(blockRequest.key)

		s.metrics.SetQueueSize(BeaconBlockQueue, len(s.beaconBlockQueue), s.Config.Name)

		start := time.Now()

		logCtx := s.log.WithField("slot", blockRequest.Slot)

		s.runQueueItem(ctx, BeaconBlockQueue, logCtx, func(ctx context.Context) error {
			return s.fetchAndIndexBeaconBlock(ctx, blockRequest.Slot)
		})

		s.metrics.ObserveQueueItemProcessingTime(
			BeaconBlockQueue,
			time.Since(start),
			s.Config.Name,
		)
	})
}

func (s *agent) processExecutionPayloadEnvelopeQueue(ctx context.Context) {
	if !s.Config.Ethereum.Features.GetFetchExecutionPayloadEnvelope() {
		return
	}

	drainQueue(ctx, s.executionPayloadEnvelopeQueue, func(envelopeRequest *ExecutionPayloadEnvelopeRequest) {
		defer s.releaseQueueItem(envelopeRequest.key)

		s.metrics.SetQueueSize(ExecutionPayloadEnvelopeQueue, len(s.executionPayloadEnvelopeQueue), s.Config.Name)

		start := time.Now()

		logCtx := s.log.WithField("slot", envelopeRequest.Slot)

		s.runQueueItem(ctx, ExecutionPayloadEnvelopeQueue, logCtx, func(ctx context.Context) error {
			return s.fetchAndIndexExecutionPayloadEnvelope(ctx, envelopeRequest.Slot)
		})

		s.metrics.ObserveQueueItemProcessingTime(
			ExecutionPayloadEnvelopeQueue,
			time.Since(start),
			s.Config.Name,
		)
	})
}

func (s *agent) processBeaconBadBlockQueue(ctx context.Context) {
	if !s.Config.Ethereum.Features.GetFetchBeaconBadBlock() {
		return
	}

	drainQueue(ctx, s.beaconBadBlockQueue, func(badBlockRequest *BeaconBadBlockRequest) {
		defer s.releaseQueueItem(badBlockRequest.key)

		s.metrics.SetQueueSize(BeaconBadBlockQueue, len(s.beaconBadBlockQueue), s.Config.Name)

		start := time.Now()

		if err := s.fetchAndIndexBeaconBadBlocks(ctx, badBlockRequest.Path); err != nil {
			s.log.
				WithError(err).
				Error("Failed to fetch and index beacon bad blocks")

			return
		}

		s.metrics.ObserveQueueItemProcessingTime(
			BeaconBadBlockQueue,
			time.Since(start),
			s.Config.Name,
		)
	})
}

func (s *agent) processBeaconBadBlobQueue(ctx context.Context) {
	if !s.Config.Ethereum.Features.GetFetchBeaconBadBlob() {
		return
	}

	drainQueue(ctx, s.beaconBadBlobQueue, func(badBlobRequest *BeaconBadBlobRequest) {
		defer s.releaseQueueItem(badBlobRequest.key)

		s.metrics.SetQueueSize(BeaconBadBlobQueue, len(s.beaconBadBlobQueue), s.Config.Name)

		start := time.Now()

		if err := s.fetchAndIndexBeaconBadBlobs(ctx, badBlobRequest.Path); err != nil {
			s.log.
				WithError(err).
				Error("Failed to fetch and index beacon bad blocks")

			return
		}

		s.metrics.ObserveQueueItemProcessingTime(
			BeaconBadBlobQueue,
			time.Since(start),
			s.Config.Name,
		)
	})
}

func (s *agent) processExecutionBlockTraceQueue(ctx context.Context) {
	if !s.Config.Ethereum.Features.GetFetchExecutionBlockTrace() {
		return
	}

	drainQueue(ctx, s.executionBlockTraceQueue, func(traceRequest *ExecutionBlockTraceRequest) {
		defer s.releaseQueueItem(traceRequest.key)

		s.metrics.SetQueueSize(ExecutionBlockTraceQueue, len(s.executionBlockTraceQueue), s.Config.Name)

		start := time.Now()

		logCtx := s.log.WithField("block_id", traceRequest.BlockID)

		s.runQueueItem(ctx, ExecutionBlockTraceQueue, logCtx, func(ctx context.Context) error {
			blockHash, blockNumber, err := s.resolveExecutionBlock(ctx, traceRequest.BlockID)
			if err != nil {
				return err
			}

			if s.executionBlockTraceStale(ctx, blockNumber) {
				return fmt.Errorf("%w: execution block %d", errItemStale, blockNumber)
			}

			return s.fetchAndIndexExecutionBlockTrace(ctx, blockNumber, blockHash)
		})

		s.metrics.ObserveQueueItemProcessingTime(
			ExecutionBlockTraceQueue,
			time.Since(start),
			s.Config.Name,
		)
	})
}

func (s *agent) processExecutionBadBlockQueue(ctx context.Context) {
	if !s.Config.Ethereum.Features.GetFetchExecutionBadBlock() {
		return
	}

	drainQueue(ctx, s.executionBadBlockQueue, func(badBlockRequest *ExecutionBadBlockRequest) {
		defer s.releaseQueueItem(badBlockRequest.key)

		s.metrics.SetQueueSize(ExecutionBadBlockQueue, len(s.executionBadBlockQueue), s.Config.Name)

		start := time.Now()

		if err := s.fetchAndIndexExecutionBadBlocks(ctx); err != nil {
			s.log.
				WithError(err).
				Error("Failed to fetch and index execution bad blocks")

			return
		}

		s.metrics.ObserveQueueItemProcessingTime(
			ExecutionBadBlockQueue,
			time.Since(start),
			s.Config.Name,
		)
	})
}
