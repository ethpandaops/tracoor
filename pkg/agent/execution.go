package agent

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ethpandaops/tracoor/pkg/agent/ethereum/execution"
	"github.com/ethpandaops/tracoor/pkg/compression"
	"github.com/ethpandaops/tracoor/pkg/proto/tracoor/indexer"
	"github.com/ethpandaops/tracoor/pkg/store"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func (s *agent) fetchAndIndexExecutionBlockTrace(ctx context.Context, blockNumber uint64, blockHash string) error {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	network := string(s.node.Beacon().Metadata().Network.Name)

	// Check if we've somehow already indexed this execution block trace.
	rsp, err := s.indexer.ListExecutionBlockTrace(ctx, &indexer.ListExecutionBlockTraceRequest{
		Node:      s.Config.Name,
		BlockHash: blockHash,
		Network:   network,
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

	now := time.Now()

	var (
		implementation = s.node.Execution().Metadata().Client(ctx)
		nodeVersion    = s.node.Execution().Metadata().ClientVersion()
		traceConfig    = s.Config.Ethereum.Execution
	)

	// The trace parameters belong in the key: they are per-agent configuration,
	// and two agents that disagree about them get different bytes back for the
	// same block from the same client.
	target := &dedupTarget{
		kind:    store.BlockTraceDataType,
		queue:   ExecutionBlockTraceQueue,
		network: network,
		dedupKey: executionBlockTraceDedupKey(
			blockNumber,
			blockHash,
			implementation,
			nodeVersion,
			traceConfig.GetTraceDisableMemory(),
			traceConfig.GetTraceDisableStack(),
			traceConfig.GetTraceDisableStorage(),
		),
		directory:  ExecutionBlockTraceDirectory(network, blockNumber),
		identity:   blockHash,
		extension:  ".json",
		identifier: blockHash,
		// A trace key is a guess rather than a promise, and JSON that differs
		// only in ordering is nobody's bug, so a mismatch here is not an alarm.
		severity: divergenceSeverityNotice,
		save:     s.store.SaveExecutionBlockTrace,
		remove:   s.store.DeleteExecutionBlockTrace,
	}

	// Traces arrive as a JSON-RPC envelope the result has to be pulled out of,
	// so unlike the consensus artifacts they are still buffered whole.
	fetcher := &bufferedFetcher{
		fetch: func(ctx context.Context) ([]byte, error) {
			data, ferr := s.node.Execution().GetRawDebugBlockTrace(ctx, blockHash, implementation)
			if ferr != nil {
				return nil, ferr
			}

			if data == nil {
				return nil, errors.New("execution node returned no trace")
			}

			return *data, nil
		},
		compressor: s.compressor,
		save:       s.store.SaveExecutionBlockTrace,
	}

	return s.indexDeduplicated(ctx, target, fetcher, func(ctx context.Context, outcome *dedupOutcome) error {
		_, err := s.indexer.CreateExecutionBlockTrace(ctx, &indexer.CreateExecutionBlockTraceRequest{
			Node:                    wrapperspb.String(s.Config.Name),
			BlockNumber:             wrapperspb.Int64(int64(blockNumber)), //nolint:gosec // safe.
			BlockHash:               wrapperspb.String(blockHash),
			FetchedAt:               timestamppb.New(now),
			ContentEncoding:         wrapperspb.String(compression.Default.ContentEncoding),
			Location:                wrapperspb.String(outcome.Location),
			Network:                 wrapperspb.String(network),
			ExecutionImplementation: wrapperspb.String(implementation),
			NodeVersion:             wrapperspb.String(nodeVersion),
			ContentHash:             wrapperspb.String(outcome.ContentHash),
			VerifiedAt:              timestamppb.New(outcome.VerifiedAt),
			ContentMatchedAt:        contentMatchedAt(outcome),
			DedupKey:                wrapperspb.String(target.dedupKey),
		})

		return err
	})
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
		Network:   string(s.node.Beacon().Metadata().Network.Name),
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

	// Bad blocks are node-local: another node may never have seen this block at
	// all. They are hashed anyway, so identical bytes on two nodes can be
	// noticed rather than assumed.
	contentHash := sha256.Sum256(rawBlockData)

	s.metrics.IncrementPayloadVerified(ExecutionBadBlockQueue, s.Config.Name)
	s.metrics.AddFetchedBytes(ExecutionBadBlockQueue, s.Config.Name, int64(len(rawBlockData)))

	// Compress it
	compressedBlockData, err := s.compressor.Compress(&rawBlockData, compression.Default)
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

	// Upload the execution block trace to the store.
	location, err = s.store.SaveExecutionBadBlock(ctx, &store.SaveParams{
		Data:            bytes.NewReader(compressedBlockData),
		Location:        location,
		ContentEncoding: compression.Default.ContentEncoding,
	})
	if err != nil {
		return errors.Wrap(err, "failed to save execution bad block to store")
	}

	s.metrics.AddStoredBytes(ExecutionBadBlockQueue, s.Config.Name, int64(len(compressedBlockData)))

	req := &indexer.CreateExecutionBadBlockRequest{
		Node:                    wrapperspb.String(s.Config.Name),
		BlockHash:               wrapperspb.String(block.Hash),
		FetchedAt:               timestamppb.New(time.Now()),
		Location:                wrapperspb.String(location),
		ContentEncoding:         wrapperspb.String(compression.Default.ContentEncoding),
		Network:                 wrapperspb.String(string(s.node.Beacon().Metadata().Network.Name)),
		ExecutionImplementation: wrapperspb.String(s.node.Execution().Metadata().Client(ctx)),
		NodeVersion:             wrapperspb.String(s.node.Execution().Metadata().ClientVersion()),
		ContentHash:             wrapperspb.String(hex.EncodeToString(contentHash[:])),
		VerifiedAt:              timestamppb.New(time.Now()),
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
