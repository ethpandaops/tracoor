package agent

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	eth2api "github.com/ethpandaops/go-eth2-client/api"
	eth2v1 "github.com/ethpandaops/go-eth2-client/api/v1"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/tracoor/pkg/agent/ethereum/beacon"
	"github.com/ethpandaops/tracoor/pkg/compression"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// emptySlotChain has blocks at slots 10, 11 and 13; slot 12 was missed.
var emptySlotChain = map[phase0.Slot]phase0.Root{
	10: {0x10},
	11: {0x11},
	13: {0x13},
}

// lodestarHeaders answers a slot the way Lodestar v1.50 does: headers/{slot} is 404 for an empty slot.
type lodestarHeaders struct{}

func (lodestarHeaders) FetchBeaconBlockHeader(_ context.Context, opts *eth2api.BeaconBlockHeaderOpts) (*eth2v1.BeaconBlockHeader, error) {
	var slot phase0.Slot
	if _, err := fmt.Sscanf(opts.Block, "%d", &slot); err != nil {
		return nil, err
	}

	root, ok := emptySlotChain[slot]
	if !ok {
		return nil, &eth2api.Error{Method: http.MethodGet, Endpoint: "/eth/v1/beacon/headers/" + opts.Block, StatusCode: http.StatusNotFound}
	}

	return headerAt(slot, root), nil
}

// latestAtOrBefore answers a slot with the latest block at or before it, as Lodestar's blocks/{slot}/root does.
type latestAtOrBefore struct{}

func (latestAtOrBefore) FetchBeaconBlockHeader(_ context.Context, opts *eth2api.BeaconBlockHeaderOpts) (*eth2v1.BeaconBlockHeader, error) {
	var slot phase0.Slot
	if _, err := fmt.Sscanf(opts.Block, "%d", &slot); err != nil {
		return nil, err
	}

	for at := slot; ; at-- {
		if root, ok := emptySlotChain[at]; ok {
			return headerAt(at, root), nil
		}
	}
}

type failingHeaders struct{ err error }

func (f failingHeaders) FetchBeaconBlockHeader(context.Context, *eth2api.BeaconBlockHeaderOpts) (*eth2v1.BeaconBlockHeader, error) {
	return nil, f.err
}

func headerAt(slot phase0.Slot, root phase0.Root) *eth2v1.BeaconBlockHeader {
	return &eth2v1.BeaconBlockHeader{
		Root:      root,
		Canonical: true,
		Header:    &phase0.SignedBeaconBlockHeader{Message: &phase0.BeaconBlockHeader{Slot: slot}},
	}
}

func TestBlockRootAtSlotTreatsTheEarlierBlockLodestarReturnsAsAnEmptySlot(t *testing.T) {
	for name, node := range map[string]blockHeaderSource{
		"not found":           lodestarHeaders{},
		"latest at or before": latestAtOrBefore{},
	} {
		t.Run(name, func(t *testing.T) {
			root, err := blockRootAtSlot(t.Context(), node, 12)
			require.ErrorIs(t, err, errSlotEmpty)
			require.Empty(t, root, "the block at slot 11 must never be recorded under slot 12")

			root, err = blockRootAtSlot(t.Context(), node, 13)
			require.NoError(t, err)
			require.Equal(t, phase0.Root{0x13}.String(), root)

			root, err = blockRootAtSlot(t.Context(), node, 11)
			require.NoError(t, err)
			require.Equal(t, phase0.Root{0x11}.String(), root)
		})
	}
}

func TestBlockRootAtSlotPassesOtherFailuresThrough(t *testing.T) {
	_, err := blockRootAtSlot(t.Context(), failingHeaders{err: &eth2api.Error{StatusCode: http.StatusInternalServerError}}, 12)
	require.Error(t, err)
	require.NotErrorIs(t, err, errSlotEmpty, "a failing node is not an empty slot")
}

func TestCheckBlockAtSlotRefusesABlockFromAnotherSlot(t *testing.T) {
	block := &beacon.VersionImmuneBlock{}
	block.Data.Message.Slot = "11"

	require.ErrorIs(t, checkBlockAtSlot("12", block), errSlotEmpty)
	require.NoError(t, checkBlockAtSlot("11", block))
	require.NoError(t, checkBlockAtSlot(phase0.Root{0x11}.String(), block), "a block fetched by root pins itself")
	require.NoError(t, checkBlockAtSlot("head", block))
}

// sszWithSlot is a payload whose slot field at offset reads slot.
func sszWithSlot(offset int, slot uint64, size int) []byte {
	data := payload(size)
	binary.LittleEndian.PutUint64(data[offset:], slot)

	return data
}

func serveSSZ(t *testing.T, data []byte, contentType string) string {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))

		_, _ = w.Write(data)
	}))
	t.Cleanup(server.Close)

	return server.URL
}

func TestGuardSlotRefusesTheParentStateServedForAnEmptySlot(t *testing.T) {
	fsStore, base := newTestFSStore(t)

	rsp := openTestResponse(t, serveSSZ(t, sszWithSlot(beaconStateSlotOffset, 11, 64*1024), "application/octet-stream"))

	_, _, err := streamSource(t.Context(), compression.NewCompressor(), guardSlot(sourceFromResponse(rsp), rsp.ContentType, beaconStateSlotOffset, 12), fsStore.SaveBeaconState, testPayloadLocation)
	require.ErrorIs(t, err, errSlotMismatch)
	require.NotErrorIs(t, err, errIncompleteTransfer, "the payload arrived whole, it just belongs to another slot")

	_, statErr := os.Stat(filepath.Join(base, filepath.FromSlash(testPayloadLocation)))
	require.True(t, os.IsNotExist(statErr), "nothing may be stored for a refused payload")
}

func TestGuardSlotPassesAPayloadForItsOwnSlot(t *testing.T) {
	for name, tc := range map[string]struct {
		offset int
		size   int
	}{
		"state": {offset: beaconStateSlotOffset, size: 64 * 1024},
		"block": {offset: signedBlockSlotOffset, size: 4 * 1024},
		// The slot field is split across reads when the first read is shorter than it.
		"block one byte at a time": {offset: signedBlockSlotOffset, size: signedBlockSlotOffset + 8},
	} {
		t.Run(name, func(t *testing.T) {
			fsStore, _ := newTestFSStore(t)
			data := sszWithSlot(tc.offset, 12, tc.size)

			src := guardSlot(sourceFromBytes(data), "application/octet-stream", tc.offset, 12)
			if tc.size == signedBlockSlotOffset+8 {
				src.body = &oneByteReader{ReadCloser: src.body}
			}

			_, result, err := streamSource(t.Context(), compression.NewCompressor(), src, fsStore.SaveBeaconBlock, testPayloadLocation)
			require.NoError(t, err)
			require.Equal(t, int64(len(data)), result.RawSize)
		})
	}
}

func TestGuardSlotRefusesAPayloadTooShortToHoldItsSlot(t *testing.T) {
	fsStore, _ := newTestFSStore(t)

	src := guardSlot(sourceFromBytes(payload(beaconStateSlotOffset+4)), "application/octet-stream", beaconStateSlotOffset, 12)

	_, _, err := streamSource(t.Context(), compression.NewCompressor(), src, fsStore.SaveBeaconState, testPayloadLocation)
	require.ErrorIs(t, err, errSlotMismatch)
}

func TestGuardSlotLeavesAResponseThatIsNotSSZAlone(t *testing.T) {
	_, guarded := guardSlot(sourceFromBytes([]byte(`{"data":{}}`)), "application/json", beaconStateSlotOffset, 12).body.(*slotGuard)
	require.False(t, guarded)
}

func TestRunQueueItemDropsAnEmptySlotWithoutBlamingTheNode(t *testing.T) {
	for _, tc := range []struct {
		err    error
		reason string
	}{
		{err: fmt.Errorf("%w: no block at slot 12", errSlotEmpty), reason: dropReasonEmptySlot},
		{err: fmt.Errorf("failed to stream payload: %w", errSlotMismatch), reason: dropReasonSlotMismatch},
	} {
		t.Run(tc.reason, func(t *testing.T) {
			s := newTestAgent("empty-slot-" + uuid.NewString()[:8])

			for range breakerFailureThreshold + 1 {
				calls := 0

				retryIn := s.runQueueItem(t.Context(), BeaconBlockQueue, s.log, retryPolicy{}, func(context.Context) error {
					calls++

					return tc.err
				})

				require.Zero(t, retryIn)
				require.Equal(t, 1, calls, "the answer for an empty slot does not change on a second read")
			}

			require.True(t, s.breaker.Allow(BeaconBlockQueue), "an empty slot says nothing about the node")
			require.Zero(t, unsupportedCount(s, BeaconBlockQueue))
			require.Equal(t, float64(breakerFailureThreshold+1), droppedCount(s, BeaconBlockQueue, tc.reason))
		})
	}
}

func TestRunQueueItemDoesNotWaitForAnEmptySlotToBeRevealed(t *testing.T) {
	s := newTestAgent("empty-slot-envelope-" + uuid.NewString()[:8])

	retryIn := s.runQueueItem(t.Context(), ExecutionPayloadEnvelopeQueue, s.log, retryPolicy{revealBy: time.Now().Add(time.Hour)}, func(context.Context) error {
		return fmt.Errorf("%w: no block at slot 12", errSlotEmpty)
	})

	require.Zero(t, retryIn, "an empty slot has no payload to wait for")
	require.Equal(t, float64(1), droppedCount(s, ExecutionPayloadEnvelopeQueue, dropReasonEmptySlot))
	require.Zero(t, retriedCount(s, ExecutionPayloadEnvelopeQueue, dropReasonNotAvailable))
}

type oneByteReader struct {
	io.ReadCloser
}

func (r *oneByteReader) Read(p []byte) (int, error) {
	if len(p) > 1 {
		p = p[:1]
	}

	return r.ReadCloser.Read(p)
}
