package agent

import (
	"context"
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
)

type ExecutionBlockTraceRequest struct {
	BlockNumber uint64
	BlockHash   string
}

type BeaconStateRequest struct {
	Slot phase0.Slot
}

type BadBlockRequest struct {
}

func (s *agent) enqueueBeaconState(ctx context.Context, slot phase0.Slot) {
	s.beaconStateQueue <- &BeaconStateRequest{
		Slot: slot,
	}
}

func (s *agent) enqueueExecutionBlockTrace(ctx context.Context, blockHash string, blockNumber uint64) {
	s.executionBlockTraceQueue <- &ExecutionBlockTraceRequest{
		BlockNumber: blockNumber,
		BlockHash:   blockHash,
	}
}

func (s *agent) enqueueBadBlock(ctx context.Context) {
	s.executionBadBlockQueue <- &BadBlockRequest{}
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
		}

		s.metrics.ObserveQueueItemProcessingTime(
			BeaconStateQueue,
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
		}

		s.metrics.ObserveQueueItemProcessingTime(
			ExecutionBlockTraceQueue,
			time.Since(start),
		)
	}
}

func (s *agent) processBadBlockQueue(ctx context.Context) {
	for range s.executionBadBlockQueue {
		s.metrics.SetQueueSize(ExecutionBadBlockQueue, len(s.executionBadBlockQueue))

		start := time.Now()

		if err := s.fetchAndIndexBadBlocks(ctx); err != nil {
			s.log.
				WithError(err).
				Error("Failed to fetch and index execution bad blocks")
		}

		s.metrics.ObserveQueueItemProcessingTime(
			ExecutionBlockTraceQueue,
			time.Since(start),
		)
	}
}
