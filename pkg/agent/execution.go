package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/ethpandaops/tracoor/pkg/proto/tracoor/indexer"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func (s *agent) fetchAndIndexExecutionBlockTrace(ctx context.Context, blockNumber uint64, blockHash string) error {
	start := time.Now()
	defer func() {
		s.metrics.ObserveQueueItemProcessingTime(
			ExecutionBlockTraceQueue,
			time.Now().Sub(start),
		)
	}()

	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	// Check if we've somehow already indexed this execution block trace.
	rsp, err := s.indexer.ListExecutionBlockTrace(ctx, &indexer.ListExecutionBlockTraceRequest{
		Node:      s.Config.Name,
		BlockHash: blockHash,
	})
	if err != nil {
		s.log.
			WithField("block_hash", blockHash).
			WithField("block_number", blockNumber).
			WithError(err).
			Warn("Failed to check if execution block trace is already indexed. Will attempt to fetch and index anyway")
	} else {
		if rsp != nil && len(rsp.ExecutionBlockTraces) > 0 {
			s.log.WithField("block_hash", blockHash).WithField("block_number", blockNumber).Debug("Execution block trace already indexed")

			return nil
		}
	}

	// Fetch the execution block trace from the execution node.
	data, err := s.node.Execution().GetRawDebugBlockTrace(ctx, blockHash)
	if err != nil {
		return err
	}

	now := time.Now()

	location := CreateExecutionBlockTraceFileName(
		s.Config.Name,
		string(s.node.Beacon().Metadata().Network.Name),
		blockNumber,
		blockHash,
	)

	location = fmt.Sprintf("%s.json", location)

	// Upload the execution block trace to the store.
	location, err = s.store.SaveExecutionBlockTrace(ctx, data, location)
	if err != nil {
		return errors.Wrap(err, "failed to save execution block trace to store")
	}

	// Index the execution block trace.
	rrsp, err := s.indexer.CreateExecutionBlockTrace(ctx, &indexer.CreateExecutionBlockTraceRequest{
		Node:                    wrapperspb.String(s.Config.Name),
		BlockNumber:             wrapperspb.Int64(int64(blockNumber)),
		BlockHash:               wrapperspb.String(blockHash),
		FetchedAt:               timestamppb.New(now),
		Location:                wrapperspb.String(location),
		Network:                 wrapperspb.String(string(s.node.Beacon().Metadata().Network.Name)),
		ExecutionImplementation: wrapperspb.String(s.node.Execution().Metadata().Client(ctx)),
		NodeVersion:             wrapperspb.String(s.node.Execution().Metadata().ClientVersion()),
	})
	if err != nil {
		return err
	}

	s.log.
		WithField("id", rrsp.GetId().GetValue()).
		WithField("location", location).
		Debug("Execution block trace indexed")

	return nil
}
