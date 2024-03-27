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

type ExecutionBlockTraceRequest struct {
	BlockNumber uint64
	BlockHash   string
}

type ExecutionBadBlockRequest struct {
}

func (s *agent) enqueueBeaconState(ctx context.Context, slot phase0.Slot) {
	s.beaconStateQueue <- &BeaconStateRequest{
		Slot: slot,
	}
}

func (s *agent) enqueueBeaconBlock(ctx context.Context, slot phase0.Slot) {
	s.beaconBlockQueue <- &BeaconBlockRequest{
		Slot: slot,
	}
}

func (s *agent) enqueueBeaconBadBlock(ctx context.Context, path string) {
	s.beaconBadBlockQueue <- &BeaconBadBlockRequest{
		Path: path,
	}
}

func (s *agent) enqueueExecutionBlockTrace(ctx context.Context, blockHash string, blockNumber uint64) {
	s.executionBlockTraceQueue <- &ExecutionBlockTraceRequest{
		BlockNumber: blockNumber,
		BlockHash:   blockHash,
	}
}

func (s *agent) enqueueExecutionBadBlock(ctx context.Context) {
	s.executionBadBlockQueue <- &ExecutionBadBlockRequest{}
}

func (s *agent) processBeaconStateQueue(ctx context.Context) {
	for stateRequest := range s.beaconStateQueue {
		s.metrics.SetQueueSize(BeaconStateQueue, len(s.beaconStateQueue))

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
		)
	}
}

func (s *agent) processBeaconBlockQueue(ctx context.Context) {
	for blockRequest := range s.beaconBlockQueue {
		s.metrics.SetQueueSize(BeaconBlockQueue, len(s.beaconBlockQueue))

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
		)
	}
}

func (s *agent) processBeaconBadBlockQueue(ctx context.Context) {
	for badBlockRequest := range s.beaconBadBlockQueue {
		s.metrics.SetQueueSize(BeaconBadBlockQueue, len(s.beaconBadBlockQueue))

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
		)
	}
}

func (s *agent) processExecutionBlockTraceQueue(ctx context.Context) {
	for traceRequest := range s.executionBlockTraceQueue {
		s.metrics.SetQueueSize(ExecutionBlockTraceQueue, len(s.executionBlockTraceQueue))

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
		)
	}
}

func (s *agent) processExecutionBadBlockQueue(ctx context.Context) {
	for range s.executionBadBlockQueue {
		s.metrics.SetQueueSize(ExecutionBadBlockQueue, len(s.executionBadBlockQueue))

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
		)
	}
}
