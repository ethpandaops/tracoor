package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ethpandaops/tracoor/pkg/agent/ethereum/execution"
	"github.com/ethpandaops/tracoor/pkg/compression"
	"github.com/ethpandaops/tracoor/pkg/proto/tracoor/indexer"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func (s *agent) fetchAndIndexExecutionBlockTrace(ctx context.Context, blockNumber uint64, blockHash string) error {
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
	} else if rsp != nil && len(rsp.ExecutionBlockTraces) > 0 {
		s.log.WithField("block_hash", blockHash).WithField("block_number", blockNumber).Debug("Execution block trace already indexed")

		return nil
	}

	// Fetch the execution block trace from the execution node.
	data, err := s.node.Execution().GetRawDebugBlockTrace(ctx, blockHash, s.node.Execution().Metadata().Client(ctx))
	if err != nil {
		return err
	}

	now := time.Now()

	compressedData, err := s.compressor.Compress(data, compression.Gzip)
	if err != nil {
		return errors.Wrapf(err, "failed to compress execution block trace")
	}

	location := CreateExecutionBlockTraceFileName(
		s.Config.Name,
		string(s.node.Beacon().Metadata().Network.Name),
		blockNumber,
		blockHash,
	)

	location = fmt.Sprintf("%s.json", location)

	location = compression.AddExtension(location, compression.Gzip)

	// Upload the execution block trace to the store.
	location, err = s.store.SaveExecutionBlockTrace(ctx, &compressedData, location)
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

	s.metrics.IncrementItemExported(ExecutionBlockTraceQueue, s.Config.Name)

	s.log.
		WithField("id", rrsp.GetId().GetValue()).
		WithField("location", location).
		Debug("Execution block trace indexed")

	return nil
}

func (s *agent) fetchAndIndexExecutionBadBlocks(ctx context.Context) error {
	// Fetch the bad blocks from the execution node.
	blocks, err := s.node.Execution().GetBadBlocks(ctx)
	if err != nil {
		return err
	}

	for _, block := range *blocks {
		b := block

		if err := s.indexExecutionBadBlock(ctx, &b); err != nil {
			s.log.WithError(err).Error("Failed to index execution bad block")
		}
	}

	return nil
}

func (s *agent) indexExecutionBadBlock(ctx context.Context, block *execution.BadBlock) error {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	// Check if we've already indexed this execution bad blocks.
	// The bad blocks RPC returns the most recent bad blocks so theres a high likelihood we've already indexed them.
	rsp, err := s.indexer.ListExecutionBadBlock(ctx, &indexer.ListExecutionBadBlockRequest{
		Node:      s.Config.Name,
		BlockHash: block.Hash,
	})
	if err != nil {
		s.log.
			WithField("block_hash", block.Hash).
			WithError(err).
			Warn("Failed to check if execution bad block is already indexed. Since these blocks are heavy we will NOT attempt to fetch and index anyway")

		return fmt.Errorf("failed to check if execution bad block is already indexed: %w", err)
	}

	if rsp != nil && len(rsp.ExecutionBadBlocks) > 0 {
		s.log.
			WithField("block_hash", block.Hash).
			Debug("Execution bad block already indexed")

		return nil
	}

	// Convert it to a byte array.
	rawBlockData, err := json.Marshal(block)
	if err != nil {
		s.log.WithError(err).Error("Failed to marshal execution bad block to JSON")

		return err
	}

	// Compress it
	compressedBlockData, err := s.compressor.Compress(&rawBlockData, compression.Gzip)
	if err != nil {
		s.log.WithError(err).Error("Failed to compress execution bad block")

		return err
	}

	location := CreateExecutionBadBlockFileName(
		s.Config.Name,
		string(s.node.Beacon().Metadata().Network.Name),
		block.Hash,
	)

	location = fmt.Sprintf("%s.json", location)

	location = compression.AddExtension(location, compression.Gzip)

	// Upload the execution block trace to the store.
	location, err = s.store.SaveExecutionBadBlock(ctx, &compressedBlockData, location)
	if err != nil {
		return errors.Wrap(err, "failed to save execution bad block to store")
	}

	req := &indexer.CreateExecutionBadBlockRequest{
		Node:                    wrapperspb.String(s.Config.Name),
		BlockHash:               wrapperspb.String(block.Hash),
		FetchedAt:               timestamppb.New(time.Now()),
		Location:                wrapperspb.String(location),
		Network:                 wrapperspb.String(string(s.node.Beacon().Metadata().Network.Name)),
		ExecutionImplementation: wrapperspb.String(s.node.Execution().Metadata().Client(ctx)),
		NodeVersion:             wrapperspb.String(s.node.Execution().Metadata().ClientVersion()),
	}

	// Attempt to parse the block number from the json of the block.
	// If the block is so bad that it doesn't even have a block number, we'll just go without.
	header, err := block.ParseBlockHeader()
	if err != nil {
		s.log.WithError(err).Error("Failed to parse block data from bad block")
	} else if header != nil {
		if header.Number != nil {
			req.BlockNumber = wrapperspb.Int64(header.Number.Int64())
		}

		if header.Extra != nil {
			sanitizedExtra := strings.ToValidUTF8(string(header.Extra), "")

			req.BlockExtraData = wrapperspb.String(sanitizedExtra)
		}
	}

	// Index the execution block trace.
	rrsp, err := s.indexer.CreateExecutionBadBlock(ctx, req)
	if err != nil {
		return errors.Wrapf(err, "failed to index execution bad block: %v", block.Hash)
	}

	s.metrics.IncrementItemExported(ExecutionBadBlockQueue, s.Config.Name)

	s.log.
		WithField("id", rrsp.GetId().GetValue()).
		WithField("location", location).
		Debug("Execution bad block indexed")

	return nil
}
