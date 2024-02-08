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

func (n *agent) enqueueBeaconState(ctx context.Context, slot phase0.Slot) {
	n.beaconStateQueue <- &BeaconStateRequest{
		Slot: slot,
	}
}

func (n *agent) enqueueExecutionBlockTrace(ctx context.Context, blockHash string, blockNumber uint64) {
	n.executionBlockTraceQueue <- &ExecutionBlockTraceRequest{
		BlockNumber: blockNumber,
		BlockHash:   blockHash,
	}
}

func (n *agent) enqueueBadBlock(ctx context.Context) {
	n.executionBadBlockQueue <- &BadBlockRequest{}
}

func (s *agent) processBeaconStateQueue(ctx context.Context) {
	for stateRequest := range s.beaconStateQueue {
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

		s.metrics.SetQueueSize(BeaconStateQueue, len(s.beaconStateQueue))
	}
}

func (s *agent) processExecutionBlockTraceQueue(ctx context.Context) {
	for traceRequest := range s.executionBlockTraceQueue {
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

		s.metrics.SetQueueSize(ExecutionBlockTraceQueue, len(s.executionBlockTraceQueue))
	}
}

func (s *agent) processBadBlockQueue(ctx context.Context) {
	for range s.executionBadBlockQueue {
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

		s.metrics.SetQueueSize(ExecutionBadBlockQueue, len(s.executionBadBlockQueue))
	}
}
