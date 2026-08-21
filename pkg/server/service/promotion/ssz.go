package promotion

import (
	"bytes"
	"encoding/binary"
	"errors"

	"github.com/ethpandaops/go-eth2-client/spec"
	"github.com/ethpandaops/go-eth2-client/spec/altair"
	"github.com/ethpandaops/go-eth2-client/spec/bellatrix"
	"github.com/ethpandaops/go-eth2-client/spec/capella"
	"github.com/ethpandaops/go-eth2-client/spec/deneb"
	"github.com/ethpandaops/go-eth2-client/spec/electra"
	"github.com/ethpandaops/go-eth2-client/spec/gloas"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
)

// Fork-stable byte offsets into raw SSZ. They allow extraction without typed
// decoding, which keeps pairing working on forks the pinned go-eth2-client
// cannot decode yet. Stable phase0 through gloas; re-verify per new fork.
const (
	// SignedBeaconBlock: a 4-byte offset to the variable `message` followed
	// by the fixed 96-byte signature, so the leading offset must equal 100.
	blockMessageOffset    = 100
	blockSlotOffset       = 100
	blockParentRootOffset = 116
	blockStateRootOffset  = 148
	blockMinLen           = blockStateRootOffset + rootLen

	// BeaconState fixed prefix, identical in all forks.
	stateGVROffset  = 8
	stateSlotOffset = 40
	stateMinLen     = stateSlotOffset + 8

	rootLen = 32
)

var (
	errBlockMalformed  = errors.New("signed beacon block bytes are truncated or misfiled (leading offset != 100)")
	errStateMalformed  = errors.New("beacon state bytes are shorter than the fixed prefix")
	errUndecodableFork = errors.New("no supported fork decodes this block")
)

// blockPeek is the fork-stable view of a SignedBeaconBlock's raw SSZ.
type blockPeek struct {
	Slot       uint64
	ParentRoot phase0.Root
	StateRoot  phase0.Root
}

// peekBlock extracts the fork-stable fields of a SignedBeaconBlock. A payload
// failing the sanity rules is truncated or misfiled; promote nothing under a
// label it contradicts.
func peekBlock(raw []byte) (*blockPeek, error) {
	if len(raw) < blockMinLen {
		return nil, errBlockMalformed
	}

	if binary.LittleEndian.Uint32(raw[0:4]) != blockMessageOffset {
		return nil, errBlockMalformed
	}

	p := &blockPeek{
		Slot: binary.LittleEndian.Uint64(raw[blockSlotOffset : blockSlotOffset+8]),
	}

	copy(p.ParentRoot[:], raw[blockParentRootOffset:blockParentRootOffset+rootLen])
	copy(p.StateRoot[:], raw[blockStateRootOffset:blockStateRootOffset+rootLen])

	return p, nil
}

// statePeek is the fork-stable view of a BeaconState's raw SSZ prefix.
type statePeek struct {
	GenesisValidatorsRoot phase0.Root
	Slot                  uint64
}

func peekState(raw []byte) (*statePeek, error) {
	if len(raw) < stateMinLen {
		return nil, errStateMalformed
	}

	p := &statePeek{
		Slot: binary.LittleEndian.Uint64(raw[stateSlotOffset : stateSlotOffset+8]),
	}

	copy(p.GenesisValidatorsRoot[:], raw[stateGVROffset:stateGVROffset+rootLen])

	return p, nil
}

type sszBlock interface {
	UnmarshalSSZ([]byte) error
	MarshalSSZ() ([]byte, error)
}

// roundTrips reports whether raw decodes into b AND re-encodes to the exact
// same bytes. SSZ offsets make wrong-fork decodes overwhelmingly fail; the
// byte-identical round-trip closes the rest.
func roundTrips(b sszBlock, raw []byte) bool {
	if err := b.UnmarshalSSZ(raw); err != nil {
		return false
	}

	enc, err := b.MarshalSSZ()

	return err == nil && bytes.Equal(enc, raw)
}

// decodeBlock attempts a typed decode of raw SignedBeaconBlock SSZ, trying
// forks newest first. A fork too new for the pinned go-eth2-client returns
// errUndecodableFork - which is itself a trigger, never a hard failure.
// Structurally identical forks report the older name (heze blocks decode as
// gloas, fulu blocks as electra); triggers evaluate identically either way.
func decodeBlock(raw []byte) (*spec.VersionedSignedBeaconBlock, error) {
	if b := new(gloas.SignedBeaconBlock); roundTrips(b, raw) {
		return &spec.VersionedSignedBeaconBlock{Version: spec.DataVersionGloas, Gloas: b}, nil
	}

	if b := new(electra.SignedBeaconBlock); roundTrips(b, raw) {
		return &spec.VersionedSignedBeaconBlock{Version: spec.DataVersionElectra, Electra: b}, nil
	}

	if b := new(deneb.SignedBeaconBlock); roundTrips(b, raw) {
		return &spec.VersionedSignedBeaconBlock{Version: spec.DataVersionDeneb, Deneb: b}, nil
	}

	if b := new(capella.SignedBeaconBlock); roundTrips(b, raw) {
		return &spec.VersionedSignedBeaconBlock{Version: spec.DataVersionCapella, Capella: b}, nil
	}

	if b := new(bellatrix.SignedBeaconBlock); roundTrips(b, raw) {
		return &spec.VersionedSignedBeaconBlock{Version: spec.DataVersionBellatrix, Bellatrix: b}, nil
	}

	if b := new(altair.SignedBeaconBlock); roundTrips(b, raw) {
		return &spec.VersionedSignedBeaconBlock{Version: spec.DataVersionAltair, Altair: b}, nil
	}

	if b := new(phase0.SignedBeaconBlock); roundTrips(b, raw) {
		return &spec.VersionedSignedBeaconBlock{Version: spec.DataVersionPhase0, Phase0: b}, nil
	}

	return nil, errUndecodableFork
}
