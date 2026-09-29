package indexer

import (
	"bytes"
	"context"
	"testing"

	pindexer "github.com/ethpandaops/tracoor/pkg/proto/tracoor/indexer"
	"github.com/ethpandaops/tracoor/pkg/store"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// Values shared by the indexer tests.
const (
	testDataLocation = "data.json"
	testNetwork      = "mainnet"
)

// The create path now demands that the payload a row points at is real: either a ready blob
// recorded at the row's location, or an object the store can see. These helpers give a request
// whichever of the two its kind needs, so tests can keep saying "create this".

func seedBlob(ctx context.Context, index *Indexer, kind, network, dedupKey, contentHash, location string) error {
	_, err := index.CreateBlob(ctx, &pindexer.CreateBlobRequest{
		Kind:        wrapperspb.String(kind),
		Network:     wrapperspb.String(network),
		DedupKey:    wrapperspb.String(dedupKey),
		ContentHash: wrapperspb.String(contentHash),
		Location:    wrapperspb.String(location),
	})

	return err
}

func seedObject(ctx context.Context, index *Indexer, location string) error {
	_, err := index.Store().SaveBeaconBadBlock(ctx, &store.SaveParams{
		Data:     bytes.NewReader([]byte("payload")),
		Location: location,
	})

	return err
}

func createBeaconState(ctx context.Context, index *Indexer, req *pindexer.CreateBeaconStateRequest) (*pindexer.CreateBeaconStateResponse, error) {
	if err := seedBlob(ctx, index, "beacon_state", req.GetNetwork().GetValue(), req.GetDedupKey().GetValue(),
		req.GetContentHash().GetValue(), req.GetLocation().GetValue()); err != nil {
		return nil, err
	}

	return index.CreateBeaconState(ctx, req)
}

func createBeaconBlock(ctx context.Context, index *Indexer, req *pindexer.CreateBeaconBlockRequest) (*pindexer.CreateBeaconBlockResponse, error) {
	if err := seedBlob(ctx, index, "beacon_block", req.GetNetwork().GetValue(), req.GetDedupKey().GetValue(),
		req.GetContentHash().GetValue(), req.GetLocation().GetValue()); err != nil {
		return nil, err
	}

	return index.CreateBeaconBlock(ctx, req)
}

func createExecutionPayloadEnvelope(ctx context.Context, index *Indexer, req *pindexer.CreateExecutionPayloadEnvelopeRequest) (*pindexer.CreateExecutionPayloadEnvelopeResponse, error) {
	if err := seedBlob(ctx, index, "execution_payload_envelope", req.GetNetwork().GetValue(), req.GetDedupKey().GetValue(),
		req.GetContentHash().GetValue(), req.GetLocation().GetValue()); err != nil {
		return nil, err
	}

	return index.CreateExecutionPayloadEnvelope(ctx, req)
}

func createExecutionBlockTrace(ctx context.Context, index *Indexer, req *pindexer.CreateExecutionBlockTraceRequest) (*pindexer.CreateExecutionBlockTraceResponse, error) {
	if err := seedBlob(ctx, index, "execution_block_trace", req.GetNetwork().GetValue(), req.GetDedupKey().GetValue(),
		req.GetContentHash().GetValue(), req.GetLocation().GetValue()); err != nil {
		return nil, err
	}

	return index.CreateExecutionBlockTrace(ctx, req)
}

func createBeaconBadBlock(ctx context.Context, index *Indexer, req *pindexer.CreateBeaconBadBlockRequest) (*pindexer.CreateBeaconBadBlockResponse, error) {
	if err := seedObject(ctx, index, req.GetLocation().GetValue()); err != nil {
		return nil, err
	}

	return index.CreateBeaconBadBlock(ctx, req)
}

func createExecutionBadBlock(ctx context.Context, index *Indexer, req *pindexer.CreateExecutionBadBlockRequest) (*pindexer.CreateExecutionBadBlockResponse, error) {
	if err := seedObject(ctx, index, req.GetLocation().GetValue()); err != nil {
		return nil, err
	}

	return index.CreateExecutionBadBlock(ctx, req)
}

func specForKind(t *testing.T, index *Indexer, kind string) purgeSpec {
	t.Helper()

	for _, spec := range index.purgeSpecs() {
		if spec.kind == kind {
			return spec
		}
	}

	t.Fatalf("no purge spec for kind %s", kind)

	return purgeSpec{}
}
