package indexer

import (
	"context"

	"github.com/ethpandaops/tracoor/pkg/proto/tracoor/indexer"
)

// The filter-value and count handlers below front their database reads with the read cache.
// The uncached implementations live alongside the rest of each artifact's handlers.

func (i *Indexer) CountBeaconState(ctx context.Context, req *indexer.CountBeaconStateRequest) (*indexer.CountBeaconStateResponse, error) {
	return cachedRead(ctx, i.reads, req, i.countBeaconState)
}

func (i *Indexer) ListUniqueBeaconStateValues(ctx context.Context, req *indexer.ListUniqueBeaconStateValuesRequest) (*indexer.ListUniqueBeaconStateValuesResponse, error) {
	return cachedRead(ctx, i.reads, req, i.listUniqueBeaconStateValues)
}

func (i *Indexer) CountBeaconBlock(ctx context.Context, req *indexer.CountBeaconBlockRequest) (*indexer.CountBeaconBlockResponse, error) {
	return cachedRead(ctx, i.reads, req, i.countBeaconBlock)
}

func (i *Indexer) ListUniqueBeaconBlockValues(ctx context.Context, req *indexer.ListUniqueBeaconBlockValuesRequest) (*indexer.ListUniqueBeaconBlockValuesResponse, error) {
	return cachedRead(ctx, i.reads, req, i.listUniqueBeaconBlockValues)
}

func (i *Indexer) CountExecutionPayloadEnvelope(ctx context.Context, req *indexer.CountExecutionPayloadEnvelopeRequest) (*indexer.CountExecutionPayloadEnvelopeResponse, error) {
	return cachedRead(ctx, i.reads, req, i.countExecutionPayloadEnvelope)
}

func (i *Indexer) ListUniqueExecutionPayloadEnvelopeValues(ctx context.Context, req *indexer.ListUniqueExecutionPayloadEnvelopeValuesRequest) (*indexer.ListUniqueExecutionPayloadEnvelopeValuesResponse, error) {
	return cachedRead(ctx, i.reads, req, i.listUniqueExecutionPayloadEnvelopeValues)
}

func (i *Indexer) CountBeaconBadBlock(ctx context.Context, req *indexer.CountBeaconBadBlockRequest) (*indexer.CountBeaconBadBlockResponse, error) {
	return cachedRead(ctx, i.reads, req, i.countBeaconBadBlock)
}

func (i *Indexer) ListUniqueBeaconBadBlockValues(ctx context.Context, req *indexer.ListUniqueBeaconBadBlockValuesRequest) (*indexer.ListUniqueBeaconBadBlockValuesResponse, error) {
	return cachedRead(ctx, i.reads, req, i.listUniqueBeaconBadBlockValues)
}

func (i *Indexer) CountBeaconBadBlob(ctx context.Context, req *indexer.CountBeaconBadBlobRequest) (*indexer.CountBeaconBadBlobResponse, error) {
	return cachedRead(ctx, i.reads, req, i.countBeaconBadBlob)
}

func (i *Indexer) ListUniqueBeaconBadBlobValues(ctx context.Context, req *indexer.ListUniqueBeaconBadBlobValuesRequest) (*indexer.ListUniqueBeaconBadBlobValuesResponse, error) {
	return cachedRead(ctx, i.reads, req, i.listUniqueBeaconBadBlobValues)
}

func (i *Indexer) CountExecutionBlockTrace(ctx context.Context, req *indexer.CountExecutionBlockTraceRequest) (*indexer.CountExecutionBlockTraceResponse, error) {
	return cachedRead(ctx, i.reads, req, i.countExecutionBlockTrace)
}

func (i *Indexer) ListUniqueExecutionBlockTraceValues(ctx context.Context, req *indexer.ListUniqueExecutionBlockTraceValuesRequest) (*indexer.ListUniqueExecutionBlockTraceValuesResponse, error) {
	return cachedRead(ctx, i.reads, req, i.listUniqueExecutionBlockTraceValues)
}

func (i *Indexer) CountExecutionBadBlock(ctx context.Context, req *indexer.CountExecutionBadBlockRequest) (*indexer.CountExecutionBadBlockResponse, error) {
	return cachedRead(ctx, i.reads, req, i.countExecutionBadBlock)
}

func (i *Indexer) ListUniqueExecutionBadBlockValues(ctx context.Context, req *indexer.ListUniqueExecutionBadBlockValuesRequest) (*indexer.ListUniqueExecutionBadBlockValuesResponse, error) {
	return cachedRead(ctx, i.reads, req, i.listUniqueExecutionBadBlockValues)
}
