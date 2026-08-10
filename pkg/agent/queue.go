package agent

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/ethpandaops/go-eth2-client/spec/phase0"
)

// queueItem is what every queued request carries regardless of which artifact
// it is for: the pending-set key its producer claimed, so the worker that
// finishes it releases exactly that claim rather than deriving the key a second
// time, and when it was accepted, so the wait ahead of a worker can be told
// apart from the time the worker itself spends.
type queueItem struct {
	key        string
	enqueuedAt time.Time
}

func newQueueItem(key string) queueItem {
	return queueItem{key: key, enqueuedAt: time.Now()}
}

func (i queueItem) claimKey() string { return i.key }

func (i queueItem) acceptedAt() time.Time { return i.enqueuedAt }

// queued is the shape every queue worker relies on, satisfied by embedding
// queueItem.
type queued interface {
	claimKey() string
	acceptedAt() time.Time
}

type BeaconStateRequest struct {
	Slot phase0.Slot

	queueItem
}

type BeaconBlockRequest struct {
	Slot phase0.Slot

	queueItem
}

type ExecutionPayloadEnvelopeRequest struct {
	Slot phase0.Slot

	queueItem
}

type BeaconBadBlockRequest struct {
	Path string

	queueItem
}

type BeaconBadBlobRequest struct {
	Path string

	queueItem
}

// ExecutionBlockTraceRequest identifies the beacon block whose execution
// payload should be traced. It carries no execution block hash or number:
// resolving those can block on a builder revealing a payload, so it happens on
// the queue worker rather than in the beacon event callback that queues it.
type ExecutionBlockTraceRequest struct {
	BlockID string

	queueItem
}

type ExecutionBadBlockRequest struct {
	queueItem
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

	if !enqueue(ctx, s.beaconStateQueue, &BeaconStateRequest{Slot: slot, queueItem: newQueueItem(key)}) {
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

	if !enqueue(ctx, s.beaconBlockQueue, &BeaconBlockRequest{Slot: slot, queueItem: newQueueItem(key)}) {
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

	if !enqueue(ctx, s.executionPayloadEnvelopeQueue, &ExecutionPayloadEnvelopeRequest{Slot: slot, queueItem: newQueueItem(key)}) {
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

	if !enqueue(ctx, s.beaconBadBlockQueue, &BeaconBadBlockRequest{Path: path, queueItem: newQueueItem(key)}) {
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

	if !enqueue(ctx, s.beaconBadBlobQueue, &BeaconBadBlobRequest{Path: path, queueItem: newQueueItem(key)}) {
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

	if !enqueue(ctx, s.executionBlockTraceQueue, &ExecutionBlockTraceRequest{BlockID: blockID, queueItem: newQueueItem(key)}) {
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

	if !enqueue(ctx, s.executionBadBlockQueue, &ExecutionBadBlockRequest{queueItem: newQueueItem(key)}) {
		s.releaseQueueItem(key)
	}
}

func slotIdentifier(slot phase0.Slot) string {
	return strconv.FormatUint(uint64(slot), 10)
}

// runWorker drains a queue and accounts for every item it takes out of it: the
// claim released, the wait it served before a worker was free, and the time the
// worker then spent on it. The accounting lives here rather than in the handlers
// so an item that is dropped, skipped or abandoned half way is measured the same
// as one that succeeded — a handler that returns early is exactly the case the
// numbers need to include.
func runWorker[T queued](ctx context.Context, s *agent, kind Queue, queue <-chan T, handle func(item T)) {
	drainQueue(ctx, queue, func(item T) {
		defer s.releaseQueueItem(item.claimKey())

		s.metrics.SetQueueSize(kind, len(queue), s.Config.Name)
		s.metrics.ObserveQueueWaitTime(kind, time.Since(item.acceptedAt()), s.Config.Name)

		start := time.Now()

		defer func() {
			s.metrics.ObserveQueueItemProcessingTime(kind, time.Since(start), s.Config.Name)
		}()

		handle(item)
	})
}

func (s *agent) processBeaconStateQueue(ctx context.Context) {
	if !s.Config.Ethereum.Features.GetFetchBeaconState() {
		return
	}

	runWorker(ctx, s, BeaconStateQueue, s.beaconStateQueue, func(stateRequest *BeaconStateRequest) {
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

			return
		}

		s.runQueueItem(ctx, BeaconStateQueue, logCtx, func(ctx context.Context) error {
			return s.fetchAndIndexBeaconState(ctx, stateRequest.Slot)
		})
	})
}

func (s *agent) processBeaconBlockQueue(ctx context.Context) {
	if !s.Config.Ethereum.Features.GetFetchBeaconBlock() {
		return
	}

	runWorker(ctx, s, BeaconBlockQueue, s.beaconBlockQueue, func(blockRequest *BeaconBlockRequest) {
		logCtx := s.log.WithField("slot", blockRequest.Slot)

		s.runQueueItem(ctx, BeaconBlockQueue, logCtx, func(ctx context.Context) error {
			return s.fetchAndIndexBeaconBlock(ctx, blockRequest.Slot)
		})
	})
}

func (s *agent) processExecutionPayloadEnvelopeQueue(ctx context.Context) {
	if !s.Config.Ethereum.Features.GetFetchExecutionPayloadEnvelope() {
		return
	}

	runWorker(ctx, s, ExecutionPayloadEnvelopeQueue, s.executionPayloadEnvelopeQueue, func(envelopeRequest *ExecutionPayloadEnvelopeRequest) {
		logCtx := s.log.WithField("slot", envelopeRequest.Slot)

		s.runQueueItem(ctx, ExecutionPayloadEnvelopeQueue, logCtx, func(ctx context.Context) error {
			return s.fetchAndIndexExecutionPayloadEnvelope(ctx, envelopeRequest.Slot)
		})
	})
}

func (s *agent) processBeaconBadBlockQueue(ctx context.Context) {
	if !s.Config.Ethereum.Features.GetFetchBeaconBadBlock() {
		return
	}

	runWorker(ctx, s, BeaconBadBlockQueue, s.beaconBadBlockQueue, func(badBlockRequest *BeaconBadBlockRequest) {
		if err := s.fetchAndIndexBeaconBadBlocks(ctx, badBlockRequest.Path); err != nil {
			s.log.
				WithError(err).
				Error("Failed to fetch and index beacon bad blocks")
		}
	})
}

func (s *agent) processBeaconBadBlobQueue(ctx context.Context) {
	if !s.Config.Ethereum.Features.GetFetchBeaconBadBlob() {
		return
	}

	runWorker(ctx, s, BeaconBadBlobQueue, s.beaconBadBlobQueue, func(badBlobRequest *BeaconBadBlobRequest) {
		if err := s.fetchAndIndexBeaconBadBlobs(ctx, badBlobRequest.Path); err != nil {
			s.log.
				WithError(err).
				Error("Failed to fetch and index beacon bad blobs")
		}
	})
}

func (s *agent) processExecutionBlockTraceQueue(ctx context.Context) {
	if !s.Config.Ethereum.Features.GetFetchExecutionBlockTrace() {
		return
	}

	runWorker(ctx, s, ExecutionBlockTraceQueue, s.executionBlockTraceQueue, func(traceRequest *ExecutionBlockTraceRequest) {
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
	})
}

func (s *agent) processExecutionBadBlockQueue(ctx context.Context) {
	if !s.Config.Ethereum.Features.GetFetchExecutionBadBlock() {
		return
	}

	runWorker(ctx, s, ExecutionBadBlockQueue, s.executionBadBlockQueue, func(_ *ExecutionBadBlockRequest) {
		if err := s.fetchAndIndexExecutionBadBlocks(ctx); err != nil {
			s.log.
				WithError(err).
				Error("Failed to fetch and index execution bad blocks")
		}
	})
}
