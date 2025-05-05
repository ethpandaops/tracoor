package agent

import (
	"context"
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
)

type BeaconStateRequest struct {
	Slot phase0.Slot
}

type BeaconBlockRequest struct {
	Slot phase0.Slot
}

type BeaconBadBlockRequest struct {
	Path string
}

type BeaconBadBlobRequest struct {
	Path string
}

type ExecutionBlockTraceRequest struct {
	BlockNumber uint64
	BlockHash   string
}

type ExecutionBadBlockRequest struct {
}

func (s *agent) enqueueBeaconState(ctx context.Context, slot phase0.Slot) {
	if !s.Config.Ethereum.Features.GetFetchBeaconState() {
		return
	}

	_, nowEpoch, err := s.node.Beacon().Metadata().Wallclock().Now()
	if err != nil {
		s.log.WithError(err).Error("Failed to get current time")

		return
	}

	targetEpoch := s.node.Beacon().Metadata().Wallclock().Epochs().FromSlot(uint64(slot))
	targetEpochNumber := targetEpoch.Number()

	// If the slot is more than the allowed number of epochs old we'll skip it.
	if nowEpoch.Number()-targetEpochNumber > s.Config.Ethereum.FetchOldBeaconStates.Epochs {
		if s.Config.Ethereum.FetchOldBeaconStates.Enabled == nil ||
			!*s.Config.Ethereum.FetchOldBeaconStates.Enabled {
			s.metrics.IncrementItemSkipped(BeaconStateQueue, s.Config.Name)

			return
		}
	}

	s.beaconStateQueue <- &BeaconStateRequest{
		Slot: slot,
	}
}

func (s *agent) enqueueBeaconBlock(ctx context.Context, slot phase0.Slot) {
	if !s.Config.Ethereum.Features.GetFetchBeaconBlock() {
		return
	}

	s.beaconBlockQueue <- &BeaconBlockRequest{
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

func (s *agent) enqueueExecutionBlockTrace(ctx context.Context, blockHash string, blockNumber uint64) {
	if !s.Config.Ethereum.Features.GetFetchExecutionBlockTrace() {
		return
	}

	s.executionBlockTraceQueue <- &ExecutionBlockTraceRequest{
		BlockNumber: blockNumber,
		BlockHash:   blockHash,
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

		if err := s.fetchAndIndexBeaconState(ctx, stateRequest.Slot); err != nil {
			s.log.
				WithError(err).
				WithField("slot", stateRequest.Slot).
				Error("Failed to fetch and index beacon state")

			continue
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

		if err := s.fetchAndIndexBeaconBlock(ctx, blockRequest.Slot); err != nil {
			s.log.
				WithError(err).
				WithField("slot", blockRequest.Slot).
				Error("Failed to fetch and index beacon block")

			continue
		}

		s.metrics.ObserveQueueItemProcessingTime(
			BeaconBlockQueue,
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

		if err := s.fetchAndIndexExecutionBlockTrace(ctx, traceRequest.BlockNumber, traceRequest.BlockHash); err != nil {
			s.log.
				WithError(err).
				WithField("block_hash", traceRequest.BlockHash).
				WithField("block_number", traceRequest.BlockNumber).
				Error("Failed to fetch and index execution block trace")

			continue
		}

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
