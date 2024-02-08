package agent

import (
	"context"

	"github.com/attestantio/go-eth2-client/spec/phase0"
)

type ExecutionBlockTraceRequest struct {
	BlockNumber uint64
	BlockHash   string
}

type BeaconStateRequest struct {
	Root phase0.Root
	Slot phase0.Slot
}

type BadBlockRequest struct {
}

func (n *agent) enqueueBeaconState(ctx context.Context, root phase0.Root, slot phase0.Slot) {
	n.beaconStateQueue <- &BeaconStateRequest{
		Root: root,
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

func (s *agent) processBeaconStateQueue(ctx context.Context) error {
	for stateRequest := range s.beaconStateQueue {
		if err := s.fetchAndIndexBeaconState(ctx, stateRequest.Root, stateRequest.Slot); err != nil {
			s.log.
				WithError(err).
				WithField("state_root", rootAsString(stateRequest.Root)).
				WithField("slot", stateRequest.Slot).
				Error("Failed to fetch and index beacon state")
		}

		s.metrics.SetQueueSize(BeaconStateQueue, len(s.beaconStateQueue))
	}

	return ctx.Err()
}

func (s *agent) processExecutionBlockTraceQueue(ctx context.Context) error {
	for traceRequest := range s.executionBlockTraceQueue {
		if err := s.fetchAndIndexExecutionBlockTrace(ctx, traceRequest.BlockNumber, traceRequest.BlockHash); err != nil {
			s.log.
				WithError(err).
				WithField("block_hash", traceRequest.BlockHash).
				WithField("block_number", traceRequest.BlockNumber).
				Error("Failed to fetch and index execution block trace")
		}

		s.metrics.SetQueueSize(ExecutionBlockTraceQueue, len(s.executionBlockTraceQueue))
	}

	return ctx.Err()
}

func (s *agent) processBadBlockQueue(ctx context.Context) error {
	for _ = range s.executionBadBlockQueue {
		if err := s.fetchAndIndexBadBlocks(ctx); err != nil {
			s.log.
				WithError(err).
				Error("Failed to fetch and index execution bad blocks")
		}

		s.metrics.SetQueueSize(ExecutionBadBlockQueue, len(s.executionBadBlockQueue))
	}

	return ctx.Err()
}
