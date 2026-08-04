package promotion

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Pairing across skipped slots: the pre-state is the post-state of the
// block's PARENT, resolved by parent_root - never slot-1.
func TestPairingAcrossSkippedSlots(t *testing.T) {
	e := newEnv(t, nil)
	gvr := fillRoot(0x11)

	// Epoch 5: parent at 160, child at 163; slots 161-162 skipped.
	raws := e.seedChain(gvr, 160, 163)
	e.seedHeadAnchor(224) // epoch 7 -> target epoch 5

	e.promoter.tick(t.Context())

	manifests := e.findManifests()
	require.Len(t, manifests, 1)

	m := e.readManifest(manifests[0])
	assert.Equal(t, uint64(163), m.Block.Slot)
	assert.Contains(t, m.Promotion.Triggers, TriggerGap)
	assert.True(t, m.Promotion.Decoded)
	assert.Equal(t, "altair", m.Fork.Name)
	assert.Equal(t, BranchCanonical, m.Promotion.Branch)
	assert.Equal(t, testNetwork+"-11111111", m.Network.Name)
	assert.Equal(t, testNetwork, m.Network.ConfigName)
	assert.Equal(t, normHex(gvr.String()), m.Network.GenesisValidatorsRoot)

	// Paired with the parent's post-state at slot 160, not slot 162.
	require.NotNil(t, m.Prestate)
	assert.Equal(t, uint64(160), m.Prestate.Slot)
	assert.True(t, m.Promotion.PairVerified)
	assert.Equal(t, testNode, m.Prestate.SourceNode)

	// expected_state_root is the PARENT block's own state_root - not the
	// candidate's, which is the child's POST-state root.
	assert.Equal(t, normHex(stateRootFor(160).String()), m.Prestate.ExpectedStateRoot)
	assert.NotEqual(t, m.Block.StateRoot, m.Prestate.ExpectedStateRoot)

	// Digests are computed over the decompressed bytes, and corpus objects
	// are byte-identical to what the node served, stored uncompressed.
	stateRaw := testState(gvr, 160)
	assert.Equal(t, sha256Hex(stateRaw), m.Prestate.SHA256)
	assert.Equal(t, len(stateRaw), m.Prestate.Size)
	assert.Equal(t, stateRaw, e.readCorpusObject(statePath(m.Prestate.SHA256)))

	blockPath := strings.TrimSuffix(manifests[0], captureManifestName) + captureBlockName
	assert.Equal(t, raws[163], e.readCorpusObject(blockPath))
	assert.Equal(t, sha256Hex(raws[163]), m.Block.SHA256)
}

// A reorg promotes BOTH branches; capture ids do not collide; the branch
// referenced by a later block is canonical.
func TestReorgPromotesBothBranches(t *testing.T) {
	e := newEnv(t, nil)
	gvr := fillRoot(0x11)
	now := time.Now()

	e.seedChain(gvr, 160, 161)

	// Two branches at 162, both children of 161.
	rootB, rootC := slotRoot(0xe1, 162), slotRoot(0xe2, 162)
	e.seedBlock(testNode, 162, 5, rootB, testBlock(t, 162, blockRootFor(161), slotRoot(0xb1, 162), nil), now.Add(time.Second))
	e.seedBlock("cl-2-test", 162, 5, rootC, testBlock(t, 162, blockRootFor(161), slotRoot(0xb2, 162), nil), now)

	// A child at 163 extends rootB: deterministic canonicality evidence.
	e.seedBlock(testNode, 163, 5, blockRootFor(163), testBlock(t, 163, rootB, stateRootFor(163), nil), now)

	e.seedHeadAnchor(224)
	e.promoter.tick(t.Context())

	manifests := e.findManifests()
	require.Len(t, manifests, 2)

	branches := map[string]string{}

	for _, path := range manifests {
		m := e.readManifest(path)
		assert.Equal(t, uint64(162), m.Block.Slot)
		assert.Contains(t, m.Promotion.Triggers, TriggerReorg)

		// Both branches pair against the SAME parent state at 161.
		require.NotNil(t, m.Prestate)
		assert.Equal(t, uint64(161), m.Prestate.Slot)
		assert.Equal(t, sha256Hex(testState(gvr, 161)), m.Prestate.SHA256)

		branches[m.Block.Root] = m.Promotion.Branch
	}

	assert.Equal(t, BranchCanonical, branches[normHex(rootB.String())])
	assert.Equal(t, BranchOrphaned, branches[normHex(rootC.String())])

	// The shared pre-state is deduplicated: exactly one object under states.
	states := 0

	for path := range e.corpusFiles() {
		if strings.HasPrefix(path, corpusStatePrefix) {
			states++
		}
	}

	assert.Equal(t, 1, states)
}

// Re-processing the retained window after a restart produces zero new objects
// and rewrites nothing.
func TestIdempotency(t *testing.T) {
	e := newEnv(t, nil)
	gvr := fillRoot(0x11)

	e.seedChain(gvr, 160, 163, 166)
	e.seedHeadAnchor(224)

	e.promoter.tick(t.Context())

	before := e.corpusFiles()
	require.NotEmpty(t, before)

	// Restart: no persistent cursor, full rescan of the retained window.
	e.restart()
	e.promoter.tick(t.Context())

	assert.Equal(t, before, e.corpusFiles())
}

// The rate cap skips (never queues, never deletes); reorg promotions bypass
// it; a rescan does not burn budget on already-promoted captures.
func TestRateCap(t *testing.T) {
	e := newEnv(t, func(c *Config) { c.RateCapPerHour = 1 })
	gvr := fillRoot(0x11)
	now := time.Now()

	// Two gap-triggered candidates (163, 166) but budget for one.
	e.seedChain(gvr, 160, 163, 166)

	// A reorg at 167 with a child at 168 extending branch A.
	rootA, rootB := slotRoot(0xe1, 167), slotRoot(0xe2, 167)
	e.seedBlock(testNode, 167, 5, rootA, testBlock(t, 167, blockRootFor(166), slotRoot(0xb1, 167), nil), now.Add(time.Second))
	e.seedBlock("cl-2-test", 167, 5, rootB, testBlock(t, 167, blockRootFor(166), slotRoot(0xb2, 167), nil), now)
	e.seedBlock(testNode, 168, 5, blockRootFor(168), testBlock(t, 168, rootA, stateRootFor(168), nil), now)

	e.seedHeadAnchor(224)
	e.promoter.tick(t.Context())

	slots := map[uint64]int{}
	for _, path := range e.findManifests() {
		slots[e.readManifest(path).Block.Slot]++
	}

	// 163 consumed the budget, 166 was skipped, both reorg branches bypassed.
	assert.Equal(t, map[uint64]int{163: 1, 167: 2}, slots)

	// The rate window is in-memory by design, so a restart grants fresh
	// budget. On rescan, 163 and both 167 branches are detected as already
	// promoted WITHOUT consuming it - which is exactly why the previously
	// capped 166 now gets through.
	e.restart()
	e.promoter.tick(t.Context())

	slots = map[uint64]int{}
	for _, path := range e.findManifests() {
		slots[e.readManifest(path).Block.Slot]++
	}

	assert.Equal(t, map[uint64]int{163: 1, 166: 1, 167: 2}, slots)
}

// After a restart-rescan, cap budget is NOT spent on captures that already
// exist in the corpus: a new trigger arriving right after a rescan is still
// admitted.
func TestRescanDoesNotBurnRateBudget(t *testing.T) {
	e := newEnv(t, func(c *Config) { c.RateCapPerHour = 1 })
	gvr := fillRoot(0x11)

	e.seedChain(gvr, 160, 163)
	e.seedHeadAnchor(224)

	e.promoter.tick(t.Context())
	require.Len(t, e.findManifests(), 1)

	// Restart, rescan (finds 163 already promoted), then a fresh trigger in
	// epoch 6 becomes visible.
	e.restart()
	e.promoter.tick(t.Context())

	e.seedChain(gvr, 192, 195) // epoch 6, gap at 195
	e.seedHeadAnchor(256)      // epoch 8 -> target 6
	e.promoter.tick(t.Context())

	slots := map[uint64]bool{}
	for _, path := range e.findManifests() {
		slots[e.readManifest(path).Block.Slot] = true
	}

	assert.True(t, slots[195], "fresh trigger after rescan must still have budget")
}

// An undecodable block (fork newer than the pinned library) is itself a
// trigger and is promoted with decoded=false, pair_verified=false.
func TestUndecodableForkPromoted(t *testing.T) {
	e := newEnv(t, nil)
	gvr := fillRoot(0x11)
	now := time.Now()

	e.seedChain(gvr, 162)

	raw := undecodableBlock(163, blockRootFor(162), slotRoot(0xcc, 163))
	e.seedBlock(testNode, 163, 5, slotRoot(0xdd, 163), raw, now)

	e.seedHeadAnchor(224)
	e.promoter.tick(t.Context())

	manifests := e.findManifests()
	require.Len(t, manifests, 1)

	m := e.readManifest(manifests[0])
	assert.Equal(t, []string{TriggerUndecodableFork}, m.Promotion.Triggers)
	assert.False(t, m.Promotion.Decoded)
	assert.False(t, m.Promotion.PairVerified)
	assert.Equal(t, "unknown", m.Fork.Name)
	assert.Contains(t, manifests[0], "/unknown/")

	// Pairing still resolved via byte-peeks; the claim is present but
	// unproven, and the block bytes are stored byte-identically.
	require.NotNil(t, m.Prestate)
	assert.Equal(t, uint64(162), m.Prestate.Slot)
	assert.Equal(t, normHex(stateRootFor(162).String()), m.Prestate.ExpectedStateRoot)

	blockPath := strings.TrimSuffix(manifests[0], captureManifestName) + captureBlockName
	assert.Equal(t, raw, e.readCorpusObject(blockPath))
}

// A missing parent state degrades to an unpaired block promotion - an
// unpaired real block still beats nothing.
func TestMissingParentStatePromotesUnpaired(t *testing.T) {
	e := newEnv(t, nil)
	gvr := fillRoot(0x11)
	now := time.Now()

	// Parent block at 160 but NO state at 160 (agent hole).
	e.seedBlock(testNode, 160, 5, blockRootFor(160), testBlock(t, 160, blockRootFor(159), stateRootFor(160), nil), now)
	e.seedBlock(testNode, 163, 5, blockRootFor(163), testBlock(t, 163, blockRootFor(160), stateRootFor(163), nil), now)

	// A state elsewhere lets the network GVR resolve.
	e.seedState(testNode, 163, stateRootFor(163), testState(gvr, 163), now)

	e.seedHeadAnchor(224)
	e.promoter.tick(t.Context())

	manifests := e.findManifests()
	require.Len(t, manifests, 1)

	m := e.readManifest(manifests[0])
	assert.Equal(t, uint64(163), m.Block.Slot)
	assert.Nil(t, m.Prestate)
	assert.False(t, m.Promotion.PairVerified)
	assert.Equal(t, testNetwork+"-11111111", m.Network.Name)

	for path := range e.corpusFiles() {
		assert.False(t, strings.HasPrefix(path, corpusStatePrefix), "no state objects expected, found %s", path)
	}
}

// A state whose GVR contradicts the network is never promoted under it:
// label integrity beats promote-freely.
func TestGVRMismatchStateNotPromoted(t *testing.T) {
	e := newEnv(t, nil)
	goodGvr, badGvr := fillRoot(0x11), fillRoot(0x22)
	now := time.Now()

	// The parent row exists but its object is gone, so the pairing claim
	// cannot be branch-verified and falls back to slot-level matching.
	e.insertBlockRow(testNode, 160, 5, blockRootFor(160).String(), "missing/location", now)
	e.seedBlock(testNode, 163, 5, blockRootFor(163), testBlock(t, 163, blockRootFor(160), stateRootFor(163), nil), now)

	// The state at the parent slot belongs to another chain (recreated
	// devnet with the same name); a newer state carries the real GVR.
	e.seedState(testNode, 160, stateRootFor(160), testState(badGvr, 160), now.Add(-time.Hour))
	e.seedState(testNode, 163, stateRootFor(163), testState(goodGvr, 163), now)

	e.seedHeadAnchor(224)
	e.promoter.tick(t.Context())

	manifests := e.findManifests()
	require.Len(t, manifests, 1)

	m := e.readManifest(manifests[0])
	assert.Equal(t, uint64(163), m.Block.Slot)
	assert.Nil(t, m.Prestate, "mismatched state must not be paired")
	assert.Equal(t, testNetwork+"-11111111", m.Network.Name)
	assert.Equal(t, normHex(goodGvr.String()), m.Network.GenesisValidatorsRoot)

	badSha := sha256Hex(testState(badGvr, 160))
	_, exists := e.corpusFiles()[statePath(badSha)]
	assert.False(t, exists, "mismatched state must not reach the corpus")
}

// Baseline fires on epoch%N == 0 for the first qualifying slot only, so
// re-processing selects the same slots.
func TestBaselineTrigger(t *testing.T) {
	e := newEnv(t, nil)
	gvr := fillRoot(0x11)

	e.seedChain(gvr, 256, 257) // epoch 8, 8%8 == 0
	e.seedHeadAnchor(320)      // epoch 10 -> target 8

	e.promoter.tick(t.Context())

	manifests := e.findManifests()
	require.Len(t, manifests, 1)

	m := e.readManifest(manifests[0])
	assert.Equal(t, uint64(256), m.Block.Slot)
	assert.Equal(t, []string{TriggerBaseline}, m.Promotion.Triggers)
}

func TestSlashingTrigger(t *testing.T) {
	e := newEnv(t, nil)
	gvr := fillRoot(0x11)
	now := time.Now()

	e.seedChain(gvr, 160)
	e.seedBlock(testNode, 161, 5, blockRootFor(161), testBlock(t, 161, blockRootFor(160), stateRootFor(161), withAttesterSlashing), now)
	e.seedHeadAnchor(224)

	e.promoter.tick(t.Context())

	manifests := e.findManifests()
	require.Len(t, manifests, 1)

	m := e.readManifest(manifests[0])
	assert.Equal(t, uint64(161), m.Block.Slot)
	assert.Equal(t, []string{TriggerSlashing}, m.Promotion.Triggers)
	assert.True(t, m.Promotion.PairVerified)
}

func TestLowParticipationTrigger(t *testing.T) {
	e := newEnv(t, nil)
	gvr := fillRoot(0x11)
	now := time.Now()

	e.seedChain(gvr, 160)
	e.seedBlock(testNode, 161, 5, blockRootFor(161), testBlock(t, 161, blockRootFor(160), stateRootFor(161), withSparseSyncBits), now)
	e.seedHeadAnchor(224)

	e.promoter.tick(t.Context())

	manifests := e.findManifests()
	require.Len(t, manifests, 1)

	m := e.readManifest(manifests[0])
	assert.Equal(t, uint64(161), m.Block.Slot)
	assert.Equal(t, []string{TriggerLowParticipation}, m.Promotion.Triggers)
}

// Disabled triggers do not fire.
func TestTriggerToggle(t *testing.T) {
	disabled := false
	e := newEnv(t, func(c *Config) { c.Triggers.Gap = &disabled })
	gvr := fillRoot(0x11)

	e.seedChain(gvr, 160, 163)
	e.seedHeadAnchor(224)

	e.promoter.tick(t.Context())

	assert.Empty(t, e.findManifests())
}

func TestCaptureID(t *testing.T) {
	id := captureID(384, "0xdeadbeefcafebabe0000000000000000000000000000000000000000000000ff")
	assert.Equal(t, "000000384-deadbeefcafe", id)

	// Lexical order equals slot order.
	assert.Less(t, captureID(99, "0xaa"), captureID(100, "0xaa"))
}

func TestNetworkID(t *testing.T) {
	assert.Equal(t, fmt.Sprintf("%s-2093236b", testNetwork), networkID(testNetwork, "0x2093236b11223344"))
}
