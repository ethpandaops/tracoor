package promotion

import (
	"encoding/binary"
	"testing"

	"github.com/attestantio/go-eth2-client/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPeekBlock(t *testing.T) {
	parent := fillRoot(0x0a)
	state := fillRoot(0x0b)
	raw := testBlock(t, 12345, parent, state, nil)

	peek, err := peekBlock(raw)
	require.NoError(t, err)

	assert.Equal(t, uint64(12345), peek.Slot)
	assert.Equal(t, parent, peek.ParentRoot)
	assert.Equal(t, state, peek.StateRoot)
}

func TestPeekBlockRejectsTruncated(t *testing.T) {
	raw := testBlock(t, 1, fillRoot(0x0a), fillRoot(0x0b), nil)

	_, err := peekBlock(raw[:100])
	assert.ErrorIs(t, err, errBlockMalformed)
}

func TestPeekBlockRejectsBadOffset(t *testing.T) {
	raw := testBlock(t, 1, fillRoot(0x0a), fillRoot(0x0b), nil)
	corrupted := append([]byte{}, raw...)
	binary.LittleEndian.PutUint32(corrupted[0:4], 96)

	_, err := peekBlock(corrupted)
	assert.ErrorIs(t, err, errBlockMalformed)
}

func TestPeekState(t *testing.T) {
	gvr := fillRoot(0x1c)
	raw := testState(gvr, 777)

	peek, err := peekState(raw)
	require.NoError(t, err)

	assert.Equal(t, uint64(777), peek.Slot)
	assert.Equal(t, gvr, peek.GenesisValidatorsRoot)

	_, err = peekState(raw[:40])
	assert.ErrorIs(t, err, errStateMalformed)
}

func TestDecodeBlock(t *testing.T) {
	raw := testBlock(t, 42, fillRoot(0x0a), fillRoot(0x0b), nil)

	decoded, err := decodeBlock(raw)
	require.NoError(t, err)
	assert.Equal(t, spec.DataVersionAltair, decoded.Version)

	slot, err := decoded.Slot()
	require.NoError(t, err)
	assert.Equal(t, uint64(42), uint64(slot))
}

// undecodableBlock builds bytes that pass the fork-stable sanity rules but
// decode under no supported fork - the shape of a brand-new fork.
func undecodableBlock(slot uint64, parentRoot, stateRoot [32]byte) []byte {
	raw := make([]byte, 220)
	binary.LittleEndian.PutUint32(raw[0:4], blockMessageOffset)
	binary.LittleEndian.PutUint64(raw[blockSlotOffset:blockSlotOffset+8], slot)
	copy(raw[blockParentRootOffset:], parentRoot[:])
	copy(raw[blockStateRootOffset:], stateRoot[:])

	for i := blockMinLen; i < len(raw); i++ {
		raw[i] = 0xff
	}

	return raw
}

func TestDecodeBlockUndecodableFork(t *testing.T) {
	raw := undecodableBlock(99, fillRoot(0x0a), fillRoot(0x0b))

	peek, err := peekBlock(raw)
	require.NoError(t, err)
	assert.Equal(t, uint64(99), peek.Slot)

	_, err = decodeBlock(raw)
	assert.ErrorIs(t, err, errUndecodableFork)
}
