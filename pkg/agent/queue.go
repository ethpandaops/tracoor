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

func (s *agent) processBeaconStateQueue(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case stateRequest := <-s.beaconStateQueue:
		if err := s.fetchAndIndexBeaconState(ctx, stateRequest.Root, stateRequest.Slot); err != nil {
			s.log.
				WithError(err).
				WithField("state_root", rootAsString(stateRequest.Root)).
				WithField("slot", stateRequest.Slot).
				Error("Failed to fetch and index beacon state")
		}
	}

	return nil
}

func (s *agent) processExecutionBlockTraceQueue(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case traceRequest := <-s.executionBlockTraceQueue:
		if err := s.fetchAndIndexExecutionBlockTrace(ctx, traceRequest.BlockNumber, traceRequest.BlockHash); err != nil {
			s.log.
				WithError(err).
				WithField("block_hash", traceRequest.BlockHash).
				WithField("block_number", traceRequest.BlockNumber).
				Error("Failed to fetch and index execution block trace")
		}
	}

	return nil
}
