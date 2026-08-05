package promotion

import (
	"context"
	"fmt"
	"os"
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

// Rare-tier triggers (slashing etc.) draw from their own budget: a
// participation/gap flood that exhausts the common cap never crowds out the
// rarest captures.
func TestRareTriggerBeatsCommonFlood(t *testing.T) {
	e := newEnv(t, func(c *Config) { c.RateCapPerHour = 1 })
	gvr := fillRoot(0x11)
	now := time.Now()

	// 163 (gap) consumes the entire common budget; 168 (gap) is capped.
	e.seedChain(gvr, 160, 163, 164, 165, 168)

	// A slashing at 166 arrives mid-flood.
	e.seedBlock(testNode, 166, 5, slotRoot(0xe1, 166), testBlock(t, 166, blockRootFor(165), slotRoot(0xb1, 166), withAttesterSlashing), now)
	e.seedState(testNode, 166, slotRoot(0xb1, 166), testState(gvr, 166), now)

	e.seedHeadAnchor(224)
	e.promoter.tick(t.Context())

	slots := map[uint64][]string{}

	for _, path := range e.findManifests() {
		m := e.readManifest(path)
		slots[m.Block.Slot] = m.Promotion.Triggers
	}

	assert.Contains(t, slots, uint64(163), "common budget goes to the first gap")
	assert.NotContains(t, slots, uint64(168), "second gap must be rate capped")
	require.Contains(t, slots, uint64(166), "slashing must not compete with the flood")
	assert.Equal(t, []string{TriggerSlashing}, slots[166])
}

// The rare tier has its own hard ceiling: a mass-slashing incident cannot
// flood the corpus either.
func TestRareCapCeiling(t *testing.T) {
	e := newEnv(t, func(c *Config) { c.RareCapPerHour = 1 })
	gvr := fillRoot(0x11)
	now := time.Now()

	e.seedChain(gvr, 160)

	for _, slot := range []uint64{161, 162} {
		e.seedBlock(testNode, slot, 5, slotRoot(0xe1, slot), testBlock(t, slot, blockRootFor(slot-1), slotRoot(0xb1, slot), withAttesterSlashing), now)
		e.seedState(testNode, slot, slotRoot(0xb1, slot), testState(gvr, slot), now)
	}

	e.seedHeadAnchor(224)
	e.promoter.tick(t.Context())

	manifests := e.findManifests()
	require.Len(t, manifests, 1)
	assert.Equal(t, uint64(161), e.readManifest(manifests[0]).Block.Slot)
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

// A failed corpus write consumes no rate budget and holds the epoch back for
// retry: after the store recovers, the capture is promoted with the budget
// intact.
func TestFailedWriteConsumesNoBudgetAndRetries(t *testing.T) {
	e := newEnv(t, func(c *Config) { c.RateCapPerHour = 1 })
	gvr := fillRoot(0x11)

	e.seedChain(gvr, 160, 163)
	e.seedHeadAnchor(224)

	// Corpus outage: every write fails.
	requireWriteProtectable(t)
	require.NoError(t, os.Chmod(e.corpusDir, 0o555))

	e.promoter.tick(t.Context())
	assert.Empty(t, e.findManifests())

	// Recovery: the held-back epoch is retried and the (unconsumed) budget
	// admits the capture.
	require.NoError(t, os.Chmod(e.corpusDir, 0o755))

	e.promoter.tick(t.Context())
	require.Len(t, e.findManifests(), 1)
}

// A persistently failing epoch is retried boundedly, then abandoned loudly -
// one poison candidate must not block every later epoch until the reaper
// eats them.
func TestFailingEpochIsAbandonedAfterBoundedRetries(t *testing.T) {
	e := newEnv(t, nil)
	gvr := fillRoot(0x11)

	e.seedChain(gvr, 160, 163)
	e.seedHeadAnchor(224)

	requireWriteProtectable(t)
	require.NoError(t, os.Chmod(e.corpusDir, 0o555))

	for range maxEpochAttempts {
		e.promoter.tick(t.Context())
	}

	// The epoch was abandoned and progress advanced past it.
	assert.Equal(t, uint64(5), e.promoter.network(testNetwork).lastProcessedEpoch)

	// Even after recovery the abandoned capture is not retried (until a
	// restart rescan); nothing was promoted.
	require.NoError(t, os.Chmod(e.corpusDir, 0o755))

	e.promoter.tick(t.Context())
	assert.Empty(t, e.findManifests())
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

// A devnet recreated under the same name restarts at slot 0. The cursor only
// ever moves forward, so without noticing the lost height the service would
// sit above every real epoch and silently promote nothing until restart.
func TestNetworkResetOnHeightRegression(t *testing.T) {
	e := newEnv(t, nil)

	// Chain A, epoch 5.
	e.seedChain(fillRoot(0x11), 160, 163)
	e.seedHeadAnchor(224)

	e.promoter.tick(t.Context())
	require.Len(t, e.findManifests(), 1)
	require.Equal(t, uint64(5), e.promoter.network(testNetwork).lastProcessedEpoch)

	// The devnet is torn down and recreated under the same name: slots
	// restart at 0 and the reaper clears chain A.
	e.purge()

	gvrB := fillRoot(0x22)
	e.seedChain(gvrB, 0, 1, 32, 35)
	e.seedHeadAnchor(96) // epoch 3 -> target 1

	e.promoter.tick(t.Context())

	state := e.promoter.network(testNetwork)
	assert.Equal(t, uint64(1), state.lastProcessedEpoch, "cursor must follow the chain down")
	assert.Equal(t, normHex(gvrB.String()), state.gvr, "identity must follow the new chain")

	// Chain B's captures land, under chain B's name.
	networks := map[string]bool{}

	for _, path := range e.findManifests() {
		networks[e.readManifest(path).Network.Name] = true
	}

	assert.True(t, networks[testNetwork+"-22222222"], "chain B must be promoted, got %v", networks)
}

// When the identity changes while the buffer may still hold both chains, the
// backlog is skipped rather than relabelled: block-only promotions take their
// network id from the cache, so rescanning it would file the old chain's
// blocks under the new chain's name.
func TestIdentityChangeSkipsAmbiguousBacklog(t *testing.T) {
	e := newEnv(t, nil)

	e.seedChain(fillRoot(0x11), 160, 163)
	e.seedHeadAnchor(224)

	e.promoter.tick(t.Context())
	require.Len(t, e.findManifests(), 1)

	before := e.corpusFiles()

	// A newer state from a different chain appears under the same name,
	// while the old chain still holds the highest slots.
	gvrB := fillRoot(0x22)
	e.seedState(testNode, 300, slotRoot(0xb2, 300), testState(gvrB, 300), time.Now().Add(time.Hour))

	// Epoch 6 has a gap-triggered, block-only candidate: exactly the shape
	// that takes its network id from the cache.
	e.seedBlock(testNode, 192, 6, blockRootFor(192), testBlock(t, 192, blockRootFor(163), stateRootFor(192), nil), time.Now())
	e.seedHeadAnchor(256) // epoch 8 -> target 6

	e.expireIdentity()
	e.promoter.tick(t.Context())

	state := e.promoter.network(testNetwork)
	assert.Equal(t, normHex(gvrB.String()), state.gvr)
	assert.Equal(t, uint64(6), state.lastProcessedEpoch, "the ambiguous backlog is skipped, not rescanned")
	assert.Equal(t, before, e.corpusFiles(), "nothing from the old chain may be relabelled")
}

// One mis-indexed row must not drag the cursor into epochs that will never
// exist: the epoch column is agent-computed, so it is cross-checked against
// the head slot before it becomes the target.
func TestImplausibleHeadEpochIsClamped(t *testing.T) {
	e := newEnv(t, nil)

	e.seedChain(fillRoot(0x11), 160, 163)
	e.seedHeadAnchor(224)

	// A row claiming an absurd epoch for a modest slot.
	e.insertBlockRow(testNode, 225, 1_000_000, blockRootFor(225).String(), "missing/location", time.Now())

	e.promoter.tick(t.Context())

	// Bounded by the head slot, not by the bogus epoch.
	assert.LessOrEqual(t, e.promoter.network(testNetwork).lastProcessedEpoch, uint64(8))
	assert.Len(t, e.findManifests(), 1)
}

// A cold start over a long window drains across ticks instead of running one
// unbounded tick.
func TestEpochBacklogIsBoundedPerTick(t *testing.T) {
	e := newEnv(t, nil)
	gvr := fillRoot(0x11)

	e.seedChain(gvr, 0)
	e.seedHeadAnchor(uint64(maxEpochsPerTick+4) * 32)

	e.promoter.tick(t.Context())
	assert.Equal(t, uint64(maxEpochsPerTick-1), e.promoter.network(testNetwork).lastProcessedEpoch)

	e.promoter.tick(t.Context())
	assert.Equal(t, uint64(maxEpochsPerTick+2), e.promoter.network(testNetwork).lastProcessedEpoch)
}

// The baseline slot is chosen from the index, not from whichever blocks
// happened to load, so an unfetchable block cannot slide the baseline onto a
// neighbouring slot and promote a second baseline for the same epoch.
func TestBaselineDoesNotSlideOnUnloadableBlock(t *testing.T) {
	e := newEnv(t, nil)
	gvr := fillRoot(0x11)

	// Epoch 8 (8%8 == 0). The lowest indexed slot has no object behind it.
	e.insertBlockRow(testNode, 256, 8, blockRootFor(256).String(), "missing/location", time.Now())
	e.seedChain(gvr, 257, 258)
	e.seedHeadAnchor(320)

	e.promoter.tick(t.Context())

	// No baseline at all rather than a baseline at the wrong slot.
	for _, path := range e.findManifests() {
		assert.NotContains(t, e.readManifest(path).Promotion.Triggers, TriggerBaseline)
	}
}

// Reorg promotions have their own budget rather than no budget: per-slot
// self-limiting is not a bound in aggregate.
func TestReorgCapCeiling(t *testing.T) {
	e := newEnv(t, func(c *Config) { c.ReorgCapPerHour = 2 })
	gvr := fillRoot(0x11)
	now := time.Now()

	e.seedChain(gvr, 160)

	// Two reorged slots, four branches, budget for two.
	for _, slot := range []uint64{161, 162} {
		for _, kind := range []byte{0xe1, 0xe2} {
			e.seedBlock(testNode, slot, 5, slotRoot(kind, slot),
				testBlock(t, slot, blockRootFor(160), slotRoot(kind, slot), nil), now)
		}
	}

	e.seedHeadAnchor(224)
	e.promoter.tick(t.Context())

	assert.Len(t, e.findManifests(), 2)
}

// Stop must return whether or not the run loop was ever started: it also runs
// when Start failed on an unreachable corpus store, or when an earlier
// service failed first, and the context it is handed is the process context,
// which shutdown does not cancel. Waiting on something only a running loop
// can satisfy would hang the process past its own signal handler.
func TestStopWithoutStartDoesNotHang(t *testing.T) {
	e := newEnv(t, nil)

	done := make(chan error, 1)

	go func() { done <- e.promoter.Stop(context.Background()) }()

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("Stop hung with no run loop to wait for")
	}

	// Stopping twice must not panic on a re-closed channel.
	require.NoError(t, e.promoter.Stop(context.Background()))
}

// Stop waits for an in-flight tick, then returns.
func TestStopWaitsForRunningLoop(t *testing.T) {
	e := newEnv(t, nil)

	e.seedChain(fillRoot(0x11), 160, 163)
	e.seedHeadAnchor(224)

	require.NoError(t, e.promoter.Start(t.Context(), nil))

	done := make(chan error, 1)

	go func() { done <- e.promoter.Stop(context.Background()) }()

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(stopTimeout + 5*time.Second):
		t.Fatal("Stop outlived its own timeout")
	}

	// The first tick ran to completion before Stop returned.
	assert.Len(t, e.findManifests(), 1)
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

// The network name arrives from an agent and ends up in an object key; the
// filesystem store resolves "../" the way filesystems do.
func TestNetworkIDSanitisesTraversal(t *testing.T) {
	assert.Equal(t, ".._.._etc-2093236b", networkID("../../etc", "0x2093236b11223344"))
	assert.Equal(t, "unknown-2093236b", networkID("..", "0x2093236b11223344"))
	assert.Equal(t, "unknown-2093236b", networkID("", "0x2093236b11223344"))

	path := capturePath(networkID("../../etc", "0x2093236b"), "electra", "000000001-aa", captureManifestName)
	assert.NotContains(t, path, "../")
	assert.True(t, strings.HasPrefix(path, corpusCapturePrefix+"/"))
}
