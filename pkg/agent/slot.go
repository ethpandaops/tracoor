package agent

import (
	"context"
	"encoding/binary"
	goerrors "errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	eth2api "github.com/ethpandaops/go-eth2-client/api"
	eth2v1 "github.com/ethpandaops/go-eth2-client/api/v1"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/tracoor/pkg/agent/ethereum/beacon"
	"github.com/ethpandaops/tracoor/pkg/mime"
)

const (
	// signedBlockSlotOffset is where a SignedBeaconBlock's slot sits in its SSZ, in every fork.
	signedBlockSlotOffset = 100

	// beaconStateSlotOffset is where a BeaconState's slot sits in its SSZ, in every fork.
	beaconStateSlotOffset = 40
)

// blockHeaderSource is the part of a beacon node that resolves a slot to its block.
type blockHeaderSource interface {
	FetchBeaconBlockHeader(ctx context.Context, opts *eth2api.BeaconBlockHeaderOpts) (*eth2v1.BeaconBlockHeader, error)
}

// blockRootAtSlot is the root of the block proposed at slot, or errSlotEmpty when the node holds none there.
func blockRootAtSlot(ctx context.Context, node blockHeaderSource, slot phase0.Slot) (string, error) {
	header, err := node.FetchBeaconBlockHeader(ctx, &eth2api.BeaconBlockHeaderOpts{Block: strconv.FormatUint(uint64(slot), 10)})
	if err != nil {
		if isNotFound(err) {
			return "", fmt.Errorf("%w: no block at slot %d", errSlotEmpty, slot)
		}

		return "", fmt.Errorf("failed to fetch beacon block header: %w", err)
	}

	if header == nil || header.Header == nil || header.Header.Message == nil {
		return "", goerrors.New("beacon node returned an empty block header")
	}

	// Some clients answer a slot with the latest block at or before it, which belongs to an earlier slot.
	if got := header.Header.Message.Slot; got != slot {
		return "", fmt.Errorf("%w: asked for slot %d, node answered with the block from slot %d", errSlotEmpty, slot, got)
	}

	return header.Root.String(), nil
}

func isNotFound(err error) bool {
	var apiErr *eth2api.Error

	return goerrors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound
}

// slotID reports the slot a block ID names, if it names one rather than a root or a tag such as head.
func slotID(blockID string) (uint64, bool) {
	slot, err := strconv.ParseUint(blockID, 10, 64)

	return slot, err == nil
}

// checkBlockAtSlot refuses a block fetched by slot number that belongs to another slot; other block IDs pass.
func checkBlockAtSlot(blockID string, block *beacon.VersionImmuneBlock) error {
	want, ok := slotID(blockID)
	if !ok {
		return nil
	}

	got, err := strconv.ParseUint(block.Data.Message.Slot, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse beacon block slot %q: %w", block.Data.Message.Slot, err)
	}

	if got != want {
		return fmt.Errorf("%w: asked for slot %d, node answered with the block from slot %d", errSlotEmpty, want, got)
	}

	return nil
}

// guardSlot makes an SSZ payload's read fail with errSlotMismatch unless the slot at offset is want.
func guardSlot(src *payloadSource, contentType string, offset int, want phase0.Slot) *payloadSource {
	if !strings.HasPrefix(contentType, string(mime.ContentTypeOctet)) {
		return src
	}

	src.body = &slotGuard{ReadCloser: src.body, offset: offset, want: uint64(want)}

	return src
}

// slotGuard checks the slot field as the first bytes of a payload pass through it.
type slotGuard struct {
	io.ReadCloser

	offset  int
	want    uint64
	prefix  []byte
	checked bool
}

func (g *slotGuard) Read(p []byte) (int, error) {
	n, err := g.ReadCloser.Read(p)

	if !g.checked {
		need := g.offset + 8 - len(g.prefix)
		g.prefix = append(g.prefix, p[:min(n, need)]...)

		if len(g.prefix) == g.offset+8 {
			g.checked = true

			if got := binary.LittleEndian.Uint64(g.prefix[g.offset:]); got != g.want {
				return 0, fmt.Errorf("%w: asked for slot %d, node served slot %d", errSlotMismatch, g.want, got)
			}
		} else if goerrors.Is(err, io.EOF) {
			return 0, fmt.Errorf("%w: payload of %d bytes is too short to hold its slot", errSlotMismatch, len(g.prefix))
		}
	}

	return n, err
}
