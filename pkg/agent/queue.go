package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/ethpandaops/go-eth2-client/spec/phase0"
)

type BeaconStateRequest struct {
	Slot phase0.Slot
}

type BeaconBlockRequest struct {
	Slot phase0.Slot
}

type ExecutionPayloadEnvelopeRequest struct {
	Slot phase0.Slot
}

type BeaconBadBlockRequest struct {
	Path string
}

type BeaconBadBlobRequest struct {
	Path string
}

// ExecutionBlockTraceRequest identifies the beacon block whose execution
// payload should be traced. It carries no execution block hash or number:
// resolving those can block on a builder revealing a payload, so it happens on
// the queue worker rather than in the beacon event callback that queues it.
type ExecutionBlockTraceRequest struct {
	BlockID string
}

type ExecutionBadBlockRequest struct {
}

func (s *agent) enqueueBeaconState(ctx context.Context, slot phase0.Slot) {
	if !s.Config.Ethereum.Features.GetFetchBeaconState() || !s.allowArtifact(BeaconStateQueue) {
		return
	}

	s.beaconStateQueue <- &BeaconStateRequest{
		Slot: slot,
	}
}

func (s *agent) enqueueBeaconBlock(ctx context.Context, slot phase0.Slot) {
	if !s.Config.Ethereum.Features.GetFetchBeaconBlock() || !s.allowArtifact(BeaconBlockQueue) {
		return
	}

	s.beaconBlockQueue <- &BeaconBlockRequest{
		Slot: slot,
	}
}

func (s *agent) enqueueExecutionPayloadEnvelope(ctx context.Context, slot phase0.Slot) {
	if !s.Config.Ethereum.Features.GetFetchExecutionPayloadEnvelope() || !s.allowArtifact(ExecutionPayloadEnvelopeQueue) {
		return
	}

	s.executionPayloadEnvelopeQueue <- &ExecutionPayloadEnvelopeRequest{
		Slot: slot,
	}
}

func (s *agent) enqueueBeaconBadBlock(ctx context.Context, path string) {
	if !s.Config.Ethereum.Features.GetFetchBeaconBadBlock() {
		return
	}

	s.beaconBadBlockQueue <- &BeaconBadBlockRequest{
		Path: path,
	}
}

func (s *agent) enqueueBeaconBadBlob(ctx context.Context, path string) {
	if !s.Config.Ethereum.Features.GetFetchBeaconBadBlob() {
		return
	}

	s.beaconBadBlobQueue <- &BeaconBadBlobRequest{
		Path: path,
	}
}

func (s *agent) enqueueExecutionBlockTrace(ctx context.Context, blockID string) {
	if !s.Config.Ethereum.Features.GetFetchExecutionBlockTrace() || !s.allowArtifact(ExecutionBlockTraceQueue) {
		return
	}

	s.executionBlockTraceQueue <- &ExecutionBlockTraceRequest{
		BlockID: blockID,
	}
}

func (s *agent) enqueueExecutionBadBlock(ctx context.Context) {
	if !s.Config.Ethereum.Features.GetFetchExecutionBadBlock() {
		return
	}

	s.executionBadBlockQueue <- &ExecutionBadBlockRequest{}
}

func (s *agent) processBeaconStateQueue(ctx context.Context) {
	if !s.Config.Ethereum.Features.GetFetchBeaconState() {
		return
	}

	for stateRequest := range s.beaconStateQueue {
		s.metrics.SetQueueSize(BeaconStateQueue, len(s.beaconStateQueue), s.Config.Name)

		start := time.Now()

		_, nowEpoch, err := s.node.Beacon().Metadata().Wallclock().Now()
		if err != nil {
			s.log.WithError(err).Error("Failed to get current time")

			continue
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
	}
}

func (s *agent) processBeaconBlockQueue(ctx context.Context) {
	if !s.Config.Ethereum.Features.GetFetchBeaconBlock() {
		return
	}

	for blockRequest := range s.beaconBlockQueue {
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
	}
}

func (s *agent) processExecutionPayloadEnvelopeQueue(ctx context.Context) {
	if !s.Config.Ethereum.Features.GetFetchExecutionPayloadEnvelope() {
		return
	}

	for envelopeRequest := range s.executionPayloadEnvelopeQueue {
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
	}
}

func (s *agent) processBeaconBadBlockQueue(ctx context.Context) {
	if !s.Config.Ethereum.Features.GetFetchBeaconBadBlock() {
		return
	}

	for badBlockRequest := range s.beaconBadBlockQueue {
		s.metrics.SetQueueSize(BeaconBadBlockQueue, len(s.beaconBadBlockQueue), s.Config.Name)

		start := time.Now()

		if err := s.fetchAndIndexBeaconBadBlocks(ctx, badBlockRequest.Path); err != nil {
			s.log.
				WithError(err).
				Error("Failed to fetch and index beacon bad blocks")

			continue
		}

		s.metrics.ObserveQueueItemProcessingTime(
			BeaconBadBlockQueue,
			time.Since(start),
			s.Config.Name,
		)
	}
}

func (s *agent) processBeaconBadBlobQueue(ctx context.Context) {
	if !s.Config.Ethereum.Features.GetFetchBeaconBadBlob() {
		return
	}

	for badBlobRequest := range s.beaconBadBlobQueue {
		s.metrics.SetQueueSize(BeaconBadBlobQueue, len(s.beaconBadBlobQueue), s.Config.Name)

		start := time.Now()

		if err := s.fetchAndIndexBeaconBadBlobs(ctx, badBlobRequest.Path); err != nil {
			s.log.
				WithError(err).
				Error("Failed to fetch and index beacon bad blocks")

			continue
		}

		s.metrics.ObserveQueueItemProcessingTime(
			BeaconBadBlobQueue,
			time.Since(start),
			s.Config.Name,
		)
	}
}

func (s *agent) processExecutionBlockTraceQueue(ctx context.Context) {
	if !s.Config.Ethereum.Features.GetFetchExecutionBlockTrace() {
		return
	}

	for traceRequest := range s.executionBlockTraceQueue {
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
	}
}

func (s *agent) processExecutionBadBlockQueue(ctx context.Context) {
	if !s.Config.Ethereum.Features.GetFetchExecutionBadBlock() {
		return
	}

	for range s.executionBadBlockQueue {
		s.metrics.SetQueueSize(ExecutionBadBlockQueue, len(s.executionBadBlockQueue), s.Config.Name)

		start := time.Now()

		if err := s.fetchAndIndexExecutionBadBlocks(ctx); err != nil {
			s.log.
				WithError(err).
				Error("Failed to fetch and index execution bad blocks")

			continue
		}

		s.metrics.ObserveQueueItemProcessingTime(
			ExecutionBadBlockQueue,
			time.Since(start),
			s.Config.Name,
		)
	}
}
