package promotion

import (
	"testing"

	"github.com/OffchainLabs/go-bitfield"
	"github.com/ethpandaops/go-eth2-client/spec"
	"github.com/ethpandaops/go-eth2-client/spec/electra"
	"github.com/ethpandaops/go-eth2-client/spec/gloas"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// decodeGloas builds a gloas block through the real decode path, so the
// triggers are evaluated against exactly what the promoter would hold.
func decodeGloas(t *testing.T, mutate func(body *gloas.BeaconBlockBody)) *spec.VersionedSignedBeaconBlock {
	t.Helper()

	raw := testGloasBlock(t, 100, fillRoot(0x0a), fillRoot(0x0b), mutate)

	decoded, err := decodeBlock(raw)
	require.NoError(t, err)
	require.Equal(t, spec.DataVersionGloas, decoded.Version)

	return decoded
}

func TestDecodedTriggersGloasSlashing(t *testing.T) {
	p := &Promoter{config: testConfig(t, t.TempDir())}

	block := decodeGloas(t, func(body *gloas.BeaconBlockBody) {
		body.ProposerSlashings = []*phase0.ProposerSlashing{{
			SignedHeader1: &phase0.SignedBeaconBlockHeader{Message: &phase0.BeaconBlockHeader{}},
			SignedHeader2: &phase0.SignedBeaconBlockHeader{Message: &phase0.BeaconBlockHeader{}},
		}}
	})

	assert.Contains(t, p.decodedTriggers(block), TriggerSlashing)
}

// Gloas moves execution requests out of the block body and into the parent
// payload's, so the versioned accessor declines them; the trigger has to read
// ParentExecutionRequests or it goes permanently silent after the fork.
func TestDecodedTriggersGloasExecutionRequests(t *testing.T) {
	p := &Promoter{config: testConfig(t, t.TempDir())}

	none := decodeGloas(t, nil)
	assert.NotContains(t, p.decodedTriggers(none), TriggerExecutionRequest)

	withDeposit := decodeGloas(t, func(body *gloas.BeaconBlockBody) {
		body.ParentExecutionRequests.Deposits = []*electra.DepositRequest{{}}
	})
	assert.Contains(t, p.decodedTriggers(withDeposit), TriggerExecutionRequest)

	// EIP-8282 builder requests are gloas-only and count the same.
	withBuilderExit := decodeGloas(t, func(body *gloas.BeaconBlockBody) {
		body.ParentExecutionRequests.BuilderExits = []*gloas.BuilderExitRequest{{}}
	})
	assert.Contains(t, p.decodedTriggers(withBuilderExit), TriggerExecutionRequest)
}

func TestDecodedTriggersGloasLowParticipation(t *testing.T) {
	p := &Promoter{config: testConfig(t, t.TempDir())}

	block := decodeGloas(t, func(body *gloas.BeaconBlockBody) {
		bits := bitfield.NewBitvector512()
		for i := range uint64(100) {
			bits.SetBitAt(i, true)
		}

		body.SyncAggregate.SyncCommitteeBits = bits
	})

	assert.Contains(t, p.decodedTriggers(block), TriggerLowParticipation)
}
