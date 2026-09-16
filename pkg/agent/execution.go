package agent

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ethpandaops/tracoor/pkg/agent/ethereum/execution"
	"github.com/ethpandaops/tracoor/pkg/compression"
	"github.com/ethpandaops/tracoor/pkg/proto/tracoor/indexer"
	"github.com/ethpandaops/tracoor/pkg/store"
	"github.com/pkg/errors"
	"golang.org/x/sync/semaphore"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func (s *agent) fetchAndIndexExecutionBlockTrace(ctx context.Context, blockNumber uint64, blockHash string) error {
	ctx, cancel := context.WithTimeout(ctx, s.Config.FetchTimeouts.ExecutionBlockTrace())
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
	dedupKey := executionBlockTraceDedupKey(
		blockNumber,
		blockHash,
		implementation,
		nodeVersion,
		traceConfig.GetTraceDisableMemory(),
		traceConfig.GetTraceDisableStack(),
		traceConfig.GetTraceDisableStorage(),
	)

	target := &dedupTarget{
		kind:      store.BlockTraceDataType,
		queue:     ExecutionBlockTraceQueue,
		network:   network,
		dedupKey:  dedupKey,
		directory: ExecutionBlockTraceDirectory(network, blockNumber),
		// The identity carries the key as well as the block, so two clients
		// that produced byte-identical traces still own separate objects.
		identity:   ExecutionBlockTraceIdentity(blockHash, dedupKey),
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

			// A node that answers with no result, or with a JSON null, has no
			// trace for this block. Hashing and indexing that answer would
			// record an artifact nobody can use and diverge it against every
			// peer that returned a real trace.
			if data == nil || isEmptyJSONResult(*data) {
				return nil, fmt.Errorf("%w: execution node returned no trace for block %s", errItemNotAvailable, blockHash)
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

// isEmptyJSONResult reports whether a JSON-RPC result carries nothing. An
// absent result and an explicit null both hash and compress perfectly well,
// which is exactly why they have to be refused rather than archived.
func isEmptyJSONResult(data []byte) bool {
	trimmed := bytes.TrimSpace(data)

	return len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null"))
}

// badBlockFetchSlots bounds how many debug_getBadBlocks responses are being
// streamed at once across every agent in the process. Sized once, from the
// first agent to ask; agents share a process precisely so their config agrees.
var (
	badBlockFetchSlots     *semaphore.Weighted
	badBlockFetchSlotsOnce sync.Once
)

func acquireBadBlockFetchSlot(ctx context.Context, limit int64) (func(), error) {
	badBlockFetchSlotsOnce.Do(func() {
		badBlockFetchSlots = semaphore.NewWeighted(limit)
	})

	if err := badBlockFetchSlots.Acquire(ctx, 1); err != nil {
		return nil, err
	}

	return func() { badBlockFetchSlots.Release(1) }, nil
}

// hashSet is a concurrency-safe set of hashes whose zero value is ready to use.
type hashSet struct {
	mu    sync.Mutex
	items map[string]struct{}
}

func (h *hashSet) has(hash string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	_, ok := h.items[hash]

	return ok
}

func (h *hashSet) add(hash string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.items == nil {
		h.items = make(map[string]struct{})
	}

	h.items[hash] = struct{}{}
}

// fetchAndIndexExecutionBadBlocks streams the node's bad block list and indexes
// whatever in it is new. The node reports its entire history on every poll,
// so almost every block in the stream is skipped on its hash alone, before
// its body is ever decoded.
func (s *agent) fetchAndIndexExecutionBadBlocks(ctx context.Context) error {
	release, err := acquireBadBlockFetchSlot(ctx, s.Config.Ethereum.GetMaxConcurrentExecutionBadBlockFetches())
	if err != nil {
		return fmt.Errorf("failed to acquire execution bad block fetch slot: %w", err)
	}

	defer release()

	network := string(s.node.Beacon().Metadata().Network.Name)

	var reported, indexed int

	err = s.node.Execution().ForEachBadBlock(ctx, s.Config.FetchTimeouts.ExecutionBadBlock(),
		func(ctx context.Context, hash string) bool {
			reported++

			return s.executionBadBlockWanted(ctx, network, hash)
		},
		func(ctx context.Context, block *execution.BadBlock) error {
			if ierr := s.indexExecutionBadBlock(ctx, network, block); ierr != nil {
				s.log.
					WithField("block_hash", block.Hash).
					WithError(ierr).
					Error("Failed to index execution bad block")

				// One block failing is no reason to abandon the rest of the
				// stream, unless the failure is the context going away.
				return ctx.Err()
			}

			indexed++

			return nil
		},
	)
	if err != nil {
		return err
	}

	s.log.
		WithField("reported", reported).
		WithField("indexed", indexed).
		Debug("Execution bad blocks scanned")

	return nil
}

// executionBadBlockWanted reports whether a bad block the node just listed
// still needs indexing. Once the indexer has confirmed a block the answer is
// remembered, so the indexer is only asked about hashes it has not confirmed
// before rather than about the node's whole history every poll.
func (s *agent) executionBadBlockWanted(ctx context.Context, network, hash string) bool {
	if s.indexedExecutionBadBlocks.has(hash) {
		return false
	}

	ctx, cancel := context.WithTimeout(ctx, s.Config.FetchTimeouts.ExecutionBadBlock())
	defer cancel()

	rsp, err := s.indexer.ListExecutionBadBlock(ctx, &indexer.ListExecutionBadBlockRequest{
		Node:      s.Config.Name,
		BlockHash: hash,
		Network:   network,
	})
	if err != nil {
		// These blocks are heavy, so an unanswered question means "not now"
		// rather than "fetch it anyway"; the next poll asks again.
		s.log.
			WithField("block_hash", hash).
			WithError(err).
			Warn("Failed to check if execution bad block is already indexed; leaving it for the next poll")

		return false
	}

	if rsp != nil && len(rsp.ExecutionBadBlocks) > 0 {
		s.indexedExecutionBadBlocks.add(hash)

		return false
	}

	return true
}

func (s *agent) indexExecutionBadBlock(ctx context.Context, network string, block *execution.BadBlock) error {
	ctx, cancel := context.WithTimeout(ctx, s.Config.FetchTimeouts.ExecutionBadBlock())
	defer cancel()

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

	location := CreateExecutionBadBlockFileName(s.Config.Name, network, block.Hash)

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
		Network:                 wrapperspb.String(network),
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

	s.indexedExecutionBadBlocks.add(block.Hash)

	s.metrics.IncrementItemExported(ExecutionBadBlockQueue, s.Config.Name)

	s.log.
		WithField("id", rrsp.GetId().GetValue()).
		WithField("location", location).
		Debug("Execution bad block indexed")

	return nil
}
