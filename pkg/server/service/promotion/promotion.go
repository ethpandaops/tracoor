// Package promotion implements the tracoor promotion service: it watches the
// short-retention capture buffer with hindsight and copies interesting
// (pre-state, block) pairs - plus a manifest - into a separate, global,
// append-only corpus store that outlives every devnet.
//
// The service never deletes anything, anywhere: copy-only toward the corpus,
// read-only toward the buffer. It keeps no persistent state; promotion is
// idempotent by construction (content-addressed states, (slot, block_root)
// derived capture ids), so a restart simply rescans the still-retained
// window.
package promotion

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/attestantio/go-eth2-client/spec"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/ethpandaops/tracoor/pkg/compression"
	"github.com/ethpandaops/tracoor/pkg/proto/tracoor"
	"github.com/ethpandaops/tracoor/pkg/server/persistence"
	"github.com/ethpandaops/tracoor/pkg/store"
)

const (
	// ServiceType uniquely identifies the promotion service.
	ServiceType = "tracoor.promotion"

	namespace = "tracoor_server_promotion"

	orderSlotAsc  = "slot ASC"
	orderSlotDesc = "slot DESC"

	// epochRowLimit bounds a single epoch listing; 32-ish slots times a
	// handful of nodes sits far below it. Hitting it is reported, never
	// silent: a truncated listing is indistinguishable from a short epoch.
	epochRowLimit = 10000

	// maxEpochAttempts bounds how often a failing epoch is retried before
	// the service abandons it and moves on.
	maxEpochAttempts = 10

	// maxEpochsPerTick bounds the work of a single tick. A cold start on a
	// long retention window, or a mis-indexed row claiming an absurd epoch,
	// must not turn one tick into an unbounded run of queries; the backlog
	// simply drains over the following ticks.
	maxEpochsPerTick = 64

	// stopTimeout bounds how long Stop waits for an in-flight tick. The
	// context the server hands to Stop is the process context, which is not
	// cancelled during shutdown, so the bound has to come from here: a tick
	// blocked on a slow corpus store must not hold the process open.
	stopTimeout = 30 * time.Second

	// networkIdentityTTL is how long a resolved genesis validators root is
	// trusted before it is re-resolved. Devnets are torn down and recreated
	// under the same config name constantly, so the GVR is refreshed rather
	// than pinned for the process lifetime - at the cost of one state fetch
	// per network per TTL.
	networkIdentityTTL = 15 * time.Minute

	// Shared log-field and metric-label keys.
	labelNetwork   = "network"
	labelSlot      = "slot"
	labelBlockRoot = "block_root"
	labelEpoch     = "epoch"
)

// Promoter is the promotion service. All mutable state is owned by the single
// run goroutine; no locking is required beyond the stop guard.
type Promoter struct {
	log     logrus.FieldLogger
	config  *Config
	db      *persistence.Indexer
	buffer  store.Store
	corpus  store.Store
	metrics *Metrics

	// networks holds per-network progress for this process lifetime only -
	// deliberately not persisted. On restart the whole retained window is
	// rescanned; idempotency makes that free.
	networks map[string]*networkState

	done chan struct{}

	// running counts the run loop, so Stop waits for a tick that is
	// actually in flight and returns immediately for a loop that was never
	// started.
	running  sync.WaitGroup
	stopOnce sync.Once
}

// networkState is everything the service remembers about one network name
// between ticks. Kurtosis devnets are recreated under the same name
// constantly, so this state is explicitly resettable: see reset.
type networkState struct {
	// lastProcessedEpoch is the cursor; seen distinguishes "epoch 0 done"
	// from "nothing done yet".
	lastProcessedEpoch uint64
	seen               bool

	// gvr is the genesis validators root the network is currently filed
	// under, and when it was resolved. Pairings verified by root carry
	// their own GVR; this only anchors fallback pairings and block-only
	// promotions.
	gvr         string
	gvrResolved time.Time

	rate rateWindow

	// retry bounds how often a failing epoch is retried before the service
	// advances past it: only the first failing epoch can block, so a single
	// entry per network suffices.
	retry *epochRetry
}

// reset drops everything derived from a chain that no longer exists, so the
// next pass rescans the retained window from its oldest epoch. The rate
// window deliberately survives: a reset must not hand out a fresh hourly
// budget, or repeated resets would become a way around the flood guard.
func (s *networkState) reset() {
	s.lastProcessedEpoch = 0
	s.seen = false
	s.gvr = ""
	s.gvrResolved = time.Time{}
	s.retry = nil
}

type rateWindow struct {
	start       time.Time
	commonCount uint64
	rareCount   uint64
	reorgCount  uint64
}

type epochRetry struct {
	epoch    uint64
	attempts int
}

// NewPromoter creates the promotion service. The corpus store is built from
// the service's own store config and must not be the buffer store.
func NewPromoter(ctx context.Context, log logrus.FieldLogger, conf *Config, db *persistence.Indexer, buffer store.Store) (*Promoter, error) {
	if err := conf.Validate(); err != nil {
		return nil, err
	}

	corpus, err := store.NewStore(namespace, log, conf.Store.Type, conf.Store.Config, store.DefaultOptions())
	if err != nil {
		return nil, errors.Wrap(err, "failed to create corpus store")
	}

	return &Promoter{
		log:      log.WithField("server/module", ServiceType),
		config:   conf,
		db:       db,
		buffer:   buffer,
		corpus:   corpus,
		metrics:  GetMetricsInstance(namespace),
		networks: make(map[string]*networkState),
		done:     make(chan struct{}),
	}, nil
}

// network returns the mutable state of a network, creating it on first sight.
func (p *Promoter) network(name string) *networkState {
	state, ok := p.networks[name]
	if !ok {
		state = &networkState{}
		p.networks[name] = state
	}

	return state
}

// Start implements service.GRPCService. The promotion service exposes no RPCs
// of its own; it only registers its background loop.
func (p *Promoter) Start(ctx context.Context, _ *grpc.Server) error {
	p.log.WithFields(logrus.Fields{
		"lag_epochs":         p.config.LagEpochs,
		"check_interval":     p.config.CheckInterval.Duration.String(),
		"rate_cap_per_hour":  p.config.RateCapPerHour,
		"rare_cap_per_hour":  p.config.RareCapPerHour,
		"reorg_cap_per_hour": p.config.ReorgCapPerHour,
	}).Info("Starting promotion service")

	if err := p.corpus.Healthy(ctx); err != nil {
		return errors.Wrap(err, "failed to connect to corpus store")
	}

	p.running.Add(1)

	go p.run(ctx)

	return nil
}

// Stop implements service.GRPCService. It waits for an in-flight tick to
// finish: the service only ever adds objects, but a half-written capture is
// worth avoiding when the shutdown is orderly.
//
// The wait must be able to give up. Stop also runs when Start never launched
// the loop - the corpus store was unreachable, or an earlier service failed
// first - and it is called with the process context, which shutdown does not
// cancel. A wait that only a running loop can satisfy would hang the process
// with the signal handler already spent, leaving SIGKILL as the only way out.
func (p *Promoter) Stop(ctx context.Context) error {
	p.log.Info("Stopping promotion service")

	p.stopOnce.Do(func() { close(p.done) })

	// Zero when the loop never started, so this returns immediately.
	finished := make(chan struct{})

	go func() {
		p.running.Wait()
		close(finished)
	}()

	timeout := time.NewTimer(stopTimeout)
	defer timeout.Stop()

	select {
	case <-finished:
	case <-ctx.Done():
		p.log.Warn("Context cancelled while waiting for the promotion loop to finish")
	case <-timeout.C:
		p.log.Warn("Timed out waiting for the promotion loop to finish")
	}

	return nil
}

func (p *Promoter) run(ctx context.Context) {
	defer p.running.Done()

	// Process immediately: everything still inside the retention window is
	// fair game, oldest first, and re-processing is idempotent.
	p.tick(ctx)

	ticker := time.NewTicker(p.config.CheckInterval.Duration)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.done:
			return
		case <-ticker.C:
			p.tick(ctx)
		}
	}
}

func (p *Promoter) tick(ctx context.Context) {
	values, err := p.db.DistinctBeaconBlockValues(ctx, []string{labelNetwork})
	if err != nil {
		p.log.WithError(err).Error("Failed to list networks")

		return
	}

	for _, network := range values.Network {
		if network == "" {
			continue
		}

		if err := p.processNetwork(ctx, network); err != nil {
			p.metrics.ObserveError(network)
			p.log.WithError(err).WithField(labelNetwork, network).Error("Failed to process network")
		}
	}
}

func (p *Promoter) processNetwork(ctx context.Context, network string) error {
	head, err := p.edgeBlock(ctx, network, orderSlotDesc)
	if err != nil || head == nil {
		return err
	}

	state := p.network(network)

	headSlot := uint64(head.Slot) //nolint:gosec // slots are non-negative
	headEpoch := p.headEpoch(network, head)

	if headEpoch < p.config.LagEpochs {
		return nil
	}

	target := headEpoch - p.config.LagEpochs

	// A cursor ahead of the target means the chain under this name lost
	// height: a devnet recreated under the same name (slots restart at 0),
	// or a mis-indexed row that has since been reaped. Without this the
	// cursor - which only ever moves forward - would sit above every real
	// epoch and the service would silently promote nothing until restart.
	if state.seen && target < state.lastProcessedEpoch {
		p.log.WithFields(logrus.Fields{
			labelNetwork:  network,
			"cursor":      state.lastProcessedEpoch,
			"head_epoch":  headEpoch,
			"head_slot":   headSlot,
			"target":      target,
			"reset_cause": "height_regression",
		}).Warn("Network lost height; resetting promotion cursor and network identity")

		state.reset()
		p.metrics.ObserveNetworkReset(network, ResetReasonHeightRegression)
	}

	// Resolve the network identity before promoting anything under it. A
	// changed GVR is the same event seen from the other side, and it can
	// arrive while the buffer still holds both chains.
	if p.refreshIdentity(ctx, network, state) && state.seen {
		// The backlog between the cursor and the target belongs to a chain
		// this name no longer denotes. Skip it rather than relabel its
		// captures under the new identity: block-only promotions take
		// their network id from the cache, so a rescan here would file the
		// old chain's blocks under the new chain's name. Once the reaper
		// finishes clearing the old chain the height regression above
		// fires and rescans what actually remains.
		p.log.WithFields(logrus.Fields{
			labelNetwork: network,
			"from":       state.lastProcessedEpoch + 1,
			"skipped_to": target,
		}).Warn("Skipping the epoch backlog spanning a network identity change")

		state.lastProcessedEpoch = target
	}

	from := state.lastProcessedEpoch + 1
	if !state.seen {
		oldest, oerr := p.edgeBlock(ctx, network, orderSlotAsc)
		if oerr != nil || oldest == nil {
			return oerr
		}

		from = uint64(oldest.Epoch) //nolint:gosec // epochs are non-negative
	}

	if from > target {
		return nil
	}

	last := target
	if last-from >= maxEpochsPerTick {
		last = from + maxEpochsPerTick - 1

		p.log.WithFields(logrus.Fields{
			labelNetwork: network,
			"from":       from,
			"until":      last,
			"target":     target,
		}).Info("Epoch backlog exceeds one tick's budget; draining over subsequent ticks")
	}

	for epoch := from; epoch <= last; epoch++ {
		if err := p.processEpoch(ctx, network, epoch, headSlot); err != nil {
			// Hold the epoch back and retry next tick - the existence
			// probe makes re-processing free - but only boundedly: a
			// deterministic poison candidate must not block every later
			// epoch until the reaper eats them.
			retry := state.retry
			if retry == nil || retry.epoch != epoch {
				retry = &epochRetry{epoch: epoch}
				state.retry = retry
			}

			retry.attempts++
			if retry.attempts < maxEpochAttempts {
				return errors.Wrapf(err, "failed to process epoch %d (attempt %d/%d)", epoch, retry.attempts, maxEpochAttempts)
			}

			p.log.WithError(err).WithFields(logrus.Fields{labelNetwork: network, labelEpoch: epoch}).Error("Abandoning epoch after repeated failures; its unpromoted captures are lost")
			p.metrics.ObserveError(network)
		}

		state.retry = nil
		state.lastProcessedEpoch = epoch
		state.seen = true

		p.metrics.ObserveLastProcessedEpoch(network, epoch)
	}

	return nil
}

// headEpoch reads the head row's epoch, cross-checked against its slot. The
// epoch column is agent-computed, so one mis-indexed row could otherwise drag
// the cursor into epochs that will never exist and stall the service until
// the row is reaped. The configured spec is the only cross-check available
// server-side; a persistent complaint here means slotsPerEpoch is wrong for
// this network, which would also make the retention check wrong.
func (p *Promoter) headEpoch(network string, head *persistence.BeaconBlock) uint64 {
	headEpoch, headSlot := uint64(head.Epoch), uint64(head.Slot) //nolint:gosec // slots/epochs are non-negative

	plausible := headSlot/p.config.SlotsPerEpoch + 1
	if headEpoch <= plausible {
		return headEpoch
	}

	p.log.WithFields(logrus.Fields{
		labelNetwork: network,
		"head_epoch": headEpoch,
		labelSlot:    headSlot,
		"clamped_to": plausible,
	}).Error("Indexed head epoch is implausible for its slot; clamping. Check that promotion slotsPerEpoch matches the network")
	p.metrics.ObserveSkip(network, SkipReasonImplausibleEpoch)

	return plausible
}

// refreshIdentity re-resolves the network's genesis validators root when the
// cached one has aged out, and reports whether it changed. A changed GVR
// means this name now denotes a different chain: pinning the first one for
// the process lifetime would file every later block-only capture under a
// network that no longer exists.
func (p *Promoter) refreshIdentity(ctx context.Context, network string, state *networkState) bool {
	if state.gvr != "" && time.Since(state.gvrResolved) < networkIdentityTTL {
		return false
	}

	gvr := p.resolveNetworkGVR(ctx, network)
	if gvr == "" {
		return false
	}

	previous := state.gvr
	changed := previous != "" && previous != gvr

	if changed {
		p.log.WithFields(logrus.Fields{
			labelNetwork:   network,
			"previous_gvr": previous,
			"current_gvr":  gvr,
		}).Warn("Network genesis validators root changed; the name now denotes a different chain")

		state.retry = nil

		p.metrics.ObserveNetworkReset(network, ResetReasonIdentityChange)
	}

	state.gvr = gvr
	state.gvrResolved = time.Now()

	return changed
}

func (p *Promoter) edgeBlock(ctx context.Context, network, order string) (*persistence.BeaconBlock, error) {
	rows, err := p.db.ListBeaconBlock(ctx, &persistence.BeaconBlockFilter{Network: &network}, &persistence.PaginationCursor{Limit: 1, OrderBy: order})
	if err != nil || len(rows) == 0 {
		return nil, err
	}

	return rows[0], nil
}

// candidate is one distinct block root at one slot, with the raw bytes any
// node served for it.
type candidate struct {
	slot    uint64
	root    string // normalised 0x hex, from the index
	node    string
	raw     []byte // decompressed SSZ, exactly as the node served it
	peek    *blockPeek
	decoded *spec.VersionedSignedBeaconBlock // nil when undecodable
	latest  time.Time                        // newest FetchedAt across rows
}

// epochPass is the working set for one (network, epoch).
type epochPass struct {
	network  string
	epoch    uint64
	headSlot uint64

	bySlot map[uint64][]*candidate
	byRoot map[string]*candidate
	slots  []uint64

	// firstIndexedSlot is the lowest slot the index holds for this epoch,
	// before any object load succeeded or failed. The baseline trigger
	// keys off it so that an unfetchable block cannot slide the baseline
	// onto a different slot between scans.
	firstIndexedSlot uint64

	// children holds every parent_root referenced by a candidate in this
	// pass: deterministic evidence for canonical-branch resolution.
	children map[string]bool

	// lookahead caches the parent roots referenced by the next epoch's
	// earliest blocks, resolved at most once per pass.
	lookahead       map[string]bool
	lookaheadLoaded bool
}

func (p *Promoter) processEpoch(ctx context.Context, network string, epoch, headSlot uint64) error {
	rows, err := p.db.ListBeaconBlock(ctx, &persistence.BeaconBlockFilter{Network: &network, Epoch: &epoch}, &persistence.PaginationCursor{Limit: epochRowLimit, OrderBy: orderSlotAsc})
	if err != nil {
		return err
	}

	if len(rows) == 0 {
		return nil
	}

	if len(rows) == epochRowLimit {
		// The listing was truncated, which is indistinguishable downstream
		// from an epoch that simply had fewer blocks. Say so.
		p.metrics.ObserveSkip(network, SkipReasonRowLimit)
		p.log.WithFields(logrus.Fields{
			labelNetwork: network,
			labelEpoch:   epoch,
			"limit":      epochRowLimit,
		}).Warn("Epoch listing hit the row limit; some captures in this epoch are not considered")
	}

	pass := &epochPass{
		network:          network,
		epoch:            epoch,
		headSlot:         headSlot,
		bySlot:           make(map[uint64][]*candidate),
		byRoot:           make(map[string]*candidate),
		children:         make(map[string]bool),
		firstIndexedSlot: firstIndexedSlot(rows),
	}

	p.buildCandidates(ctx, pass, rows)

	sort.Slice(pass.slots, func(i, j int) bool { return pass.slots[i] < pass.slots[j] })

	baselineSlot, haveBaseline := p.baselineSlot(pass)

	var failed error

	for _, slot := range pass.slots {
		cands := pass.bySlot[slot]
		isReorg := len(cands) > 1
		branches := p.resolveBranches(ctx, pass, cands)

		for _, cand := range cands {
			parent := p.resolveParent(ctx, pass, normHex(cand.peek.ParentRoot.String()))

			triggers := p.evaluateCandidate(cand, parent, isReorg, haveBaseline && slot == baselineSlot)
			if len(triggers) == 0 {
				continue
			}

			if err := p.promote(ctx, pass, cand, parent, triggers, branches[cand.root]); err != nil {
				p.metrics.ObserveError(network)
				p.log.WithError(err).WithFields(logrus.Fields{labelNetwork: network, labelSlot: slot, labelBlockRoot: cand.root}).Error("Failed to promote candidate")

				// Hold the epoch back so the capture is retried next tick;
				// the existence probe makes re-processing free, and the
				// retention window bounds how long a retry can matter.
				failed = err
			}
		}
	}

	return failed
}

// firstIndexedSlot returns the lowest non-negative slot among the rows.
func firstIndexedSlot(rows []*persistence.BeaconBlock) uint64 {
	first := uint64(0)
	found := false

	for _, row := range rows {
		if row.Slot < 0 {
			continue
		}

		slot := uint64(row.Slot)
		if !found || slot < first {
			first, found = slot, true
		}
	}

	return first
}

// buildCandidates fetches every distinct root's bytes once and byte-peeks the
// fork-stable fields. Rows whose object is gone or malformed degrade to
// skipped candidates with a metric, never to a failed epoch.
func (p *Promoter) buildCandidates(ctx context.Context, pass *epochPass, rows []*persistence.BeaconBlock) {
	byRoot := make(map[string][]*persistence.BeaconBlock)

	for _, row := range rows {
		if row.Slot < 0 || row.BlockRoot == "" {
			continue
		}

		byRoot[normHex(row.BlockRoot)] = append(byRoot[normHex(row.BlockRoot)], row)
	}

	for root, group := range byRoot {
		cand := &candidate{
			slot: uint64(group[0].Slot), //nolint:gosec // guarded non-negative above
			root: root,
		}

		for _, row := range group {
			if row.FetchedAt.After(cand.latest) {
				cand.latest = row.FetchedAt
			}
		}

		if !p.loadCandidate(ctx, pass.network, cand, group) {
			continue
		}

		pass.byRoot[root] = cand
		pass.children[normHex(cand.peek.ParentRoot.String())] = true

		if _, ok := pass.bySlot[cand.slot]; !ok {
			pass.slots = append(pass.slots, cand.slot)
		}

		pass.bySlot[cand.slot] = append(pass.bySlot[cand.slot], cand)
	}

	// Deterministic order for same-slot branches.
	for _, cands := range pass.bySlot {
		sort.Slice(cands, func(i, j int) bool { return cands[i].root < cands[j].root })
	}
}

// loadCandidate fetches, decompresses and peeks a block from any node's copy.
func (p *Promoter) loadCandidate(ctx context.Context, network string, cand *candidate, rows []*persistence.BeaconBlock) bool {
	malformed := false

	for _, row := range rows {
		data, err := p.buffer.GetBeaconBlock(ctx, row.Location)
		if err != nil || data == nil {
			continue
		}

		raw, err := decompress(*data, row.ContentEncoding)
		if err != nil {
			malformed = true

			p.log.WithError(err).WithField("location", row.Location).Warn("Failed to decompress block")

			continue
		}

		peek, err := peekBlock(raw)
		if err != nil || peek.Slot != cand.slot {
			// Truncated or misfiled: promote nothing under a label the
			// payload contradicts.
			malformed = true

			p.log.WithError(err).WithFields(logrus.Fields{"location": row.Location, labelSlot: cand.slot}).Warn("Block bytes fail sanity rules")

			continue
		}

		cand.raw = raw
		cand.peek = peek
		cand.node = row.Node
		cand.decoded, _ = decodeBlock(raw)

		return true
	}

	// One skip per candidate, attributed to what actually went wrong: bytes
	// we could not trust, or bytes we could not find.
	if malformed {
		p.metrics.ObserveSkip(network, SkipReasonMalformed)
	} else {
		p.metrics.ObserveSkip(network, SkipReasonMissingObject)
	}

	return false
}

// baselineSlot returns the baseline slot of a baseline epoch: the lowest slot
// the INDEX holds for it, not the lowest slot that happened to load. Which
// epochs fire is a pure function of the epoch number, and which slot fires is
// a pure function of the index, so a rescan selects the same capture. Keying
// off the loaded set instead would slide the baseline onto a neighbouring
// slot whenever an object became unfetchable, promoting a second baseline for
// an epoch that already had one.
func (p *Promoter) baselineSlot(pass *epochPass) (uint64, bool) {
	if !triggerEnabled(p.config.Triggers.Baseline) || p.config.BaselineEveryNEpochs == 0 {
		return 0, false
	}

	if pass.epoch%p.config.BaselineEveryNEpochs != 0 || len(pass.slots) == 0 {
		return 0, false
	}

	// The block at that slot may be unloadable, in which case this epoch
	// simply has no baseline - deterministically.
	return pass.firstIndexedSlot, true
}

// parentInfo is everything known about a candidate's parent, resolved by
// parent_root - never by slot arithmetic: skipped slots make them differ.
type parentInfo struct {
	known   bool
	slot    uint64
	peek    *blockPeek
	version spec.DataVersion
}

func (p *Promoter) resolveParent(ctx context.Context, pass *epochPass, parentRoot string) parentInfo {
	if cand, ok := pass.byRoot[parentRoot]; ok {
		info := parentInfo{known: true, slot: cand.slot, peek: cand.peek}
		if cand.decoded != nil {
			info.version = cand.decoded.Version
		}

		return info
	}

	rows, err := p.db.ListBeaconBlock(ctx, &persistence.BeaconBlockFilter{Network: &pass.network, BlockRoot: &parentRoot}, &persistence.PaginationCursor{Limit: 10, OrderBy: orderSlotAsc})
	if err != nil || len(rows) == 0 || rows[0].Slot < 0 {
		return parentInfo{}
	}

	info := parentInfo{known: true, slot: uint64(rows[0].Slot)} //nolint:gosec // guarded non-negative above

	cand := &candidate{slot: info.slot, root: parentRoot}
	if p.loadCandidate(ctx, pass.network, cand, rows) {
		info.peek = cand.peek
		if cand.decoded != nil {
			info.version = cand.decoded.Version
		}
	}

	return info
}

func (p *Promoter) evaluateCandidate(cand *candidate, parent parentInfo, isReorg, isBaseline bool) []string {
	var triggers []string

	if isReorg && triggerEnabled(p.config.Triggers.Reorg) {
		triggers = append(triggers, TriggerReorg)
	}

	if cand.decoded == nil {
		// A fork too new to decode is a fork whose states are all
		// interesting.
		if triggerEnabled(p.config.Triggers.UndecodableFork) {
			triggers = append(triggers, TriggerUndecodableFork)
		}
	} else {
		triggers = append(triggers, p.decodedTriggers(cand.decoded)...)
	}

	if triggerEnabled(p.config.Triggers.Gap) && parent.known && cand.slot > parent.slot {
		if cand.slot-parent.slot-1 >= p.config.GapTrigger {
			triggers = append(triggers, TriggerGap)
		}
	}

	if triggerEnabled(p.config.Triggers.ForkBoundary) && cand.decoded != nil &&
		parent.version != spec.DataVersionUnknown && parent.version != cand.decoded.Version {
		triggers = append(triggers, TriggerForkBoundary)
	}

	if isBaseline {
		triggers = append(triggers, TriggerBaseline)
	}

	return triggers
}

// resolveBranches labels each root at a slot canonical or orphaned. The
// preferred evidence is deterministic: the root referenced as parent_root by
// a later block won. Recency of capture is the fallback (tracoor re-captures
// the winning branch after a reorg event).
func (p *Promoter) resolveBranches(ctx context.Context, pass *epochPass, cands []*candidate) map[string]string {
	out := make(map[string]string, len(cands))
	if len(cands) == 1 {
		out[cands[0].root] = BranchCanonical

		return out
	}

	referenced := make([]*candidate, 0, len(cands))

	for _, cand := range cands {
		if pass.children[cand.root] {
			referenced = append(referenced, cand)
		}
	}

	if len(referenced) == 0 {
		// The reorged slot has no in-pass child (epoch tail); look ahead
		// one epoch for parent references before falling back to recency.
		lookahead := p.lookaheadChildren(ctx, pass)
		for _, cand := range cands {
			if lookahead[cand.root] {
				referenced = append(referenced, cand)
			}
		}
	}

	if len(referenced) == 0 {
		referenced = cands
	}

	canonical := referenced[0]
	for _, cand := range referenced[1:] {
		if cand.latest.After(canonical.latest) {
			canonical = cand
		}
	}

	for _, cand := range cands {
		if cand == canonical {
			out[cand.root] = BranchCanonical
		} else {
			out[cand.root] = BranchOrphaned
		}
	}

	return out
}

// lookaheadChildren byte-peeks the earliest blocks of the next epoch and
// returns the set of parent roots they reference. The result is cached on the
// pass: every reorged slot in an epoch asks the same question, and answering
// it costs a query plus up to 20 block fetches.
func (p *Promoter) lookaheadChildren(ctx context.Context, pass *epochPass) map[string]bool {
	if pass.lookaheadLoaded {
		return pass.lookahead
	}

	out := p.fetchLookaheadChildren(ctx, pass)

	pass.lookahead = out
	pass.lookaheadLoaded = true

	return out
}

func (p *Promoter) fetchLookaheadChildren(ctx context.Context, pass *epochPass) map[string]bool {
	out := make(map[string]bool)
	next := pass.epoch + 1

	rows, err := p.db.ListBeaconBlock(ctx, &persistence.BeaconBlockFilter{Network: &pass.network, Epoch: &next}, &persistence.PaginationCursor{Limit: 20, OrderBy: orderSlotAsc})
	if err != nil {
		return out
	}

	seen := make(map[string]bool)

	for _, row := range rows {
		root := normHex(row.BlockRoot)
		if seen[root] {
			continue
		}

		seen[root] = true

		data, err := p.buffer.GetBeaconBlock(ctx, row.Location)
		if err != nil || data == nil {
			continue
		}

		raw, err := decompress(*data, row.ContentEncoding)
		if err != nil {
			continue
		}

		if peek, perr := peekBlock(raw); perr == nil {
			out[normHex(peek.ParentRoot.String())] = true
		}
	}

	return out
}

// rateTier classifies a capture for admission. Reorgs get their own, very
// generous budget: they are self-limiting per slot and the orphaned branch is
// unobtainable anywhere else, but per-slot self-limiting is not a bound in
// aggregate. Rare-but-not-bounded classes (slashings can arrive by the
// hundreds in a mass-slashing incident) get a budget of their own so they
// never compete with a participation/gap flood, yet keep a hard ceiling.
// Everything repetitive or spammable draws from the common budget.
type rateTier int

const (
	tierReorg rateTier = iota
	tierRare
	tierCommon
)

func classifyTier(triggers []string) rateTier {
	if slices.Contains(triggers, TriggerReorg) {
		return tierReorg
	}

	for _, trigger := range triggers {
		switch trigger {
		case TriggerSlashing, TriggerVoluntaryExit, TriggerBLSToExecutionChange, TriggerForkBoundary:
			return tierRare
		}
	}

	return tierCommon
}

// skipReason names the metric label for a tier that ran out of budget.
func (t rateTier) skipReason() string {
	switch t {
	case tierReorg:
		return SkipReasonRateCappedReorg
	case tierRare:
		return SkipReasonRateCappedRare
	default:
		return SkipReasonRateCappedCommon
	}
}

// window returns the network's current hourly budget, rolling it over when
// the previous one has expired. The window is tumbling, not sliding.
func (p *Promoter) window(state *networkState) *rateWindow {
	now := time.Now()

	if state.rate.start.IsZero() || now.Sub(state.rate.start) >= time.Hour {
		state.rate = rateWindow{start: now}
	}

	return &state.rate
}

// headroom reports whether the tier's per-network hourly budget has room,
// without consuming any.
func (p *Promoter) headroom(state *networkState, tier rateTier) bool {
	w := p.window(state)

	switch tier {
	case tierReorg:
		return w.reorgCount < p.config.ReorgCapPerHour
	case tierRare:
		return w.rareCount < p.config.RareCapPerHour
	default:
		return w.commonCount < p.config.RateCapPerHour
	}
}

// consume takes one promotion from the tier's hourly budget. When a budget
// is exhausted the caller skips - it never queues and never deletes.
func (p *Promoter) consume(state *networkState, tier rateTier) {
	w := p.window(state)

	switch tier {
	case tierReorg:
		w.reorgCount++
	case tierRare:
		w.rareCount++
	default:
		w.commonCount++
	}
}

// promote copies the candidate block, its pre-state (the post-state of its
// PARENT, resolved by parent_root) and a manifest into the corpus.
//
// Ordering matters: already-promoted captures are detected before the rate
// cap is consulted, so a restart-rescan of the retained window is free and
// never burns the hourly budget on captures the corpus already has.
func (p *Promoter) promote(ctx context.Context, pass *epochPass, cand *candidate, parent parentInfo, triggers []string, branch string) error {
	forkName := unknownComponent
	if cand.decoded != nil {
		forkName = cand.decoded.Version.String()
	}

	id := captureID(cand.slot, cand.root)
	tier := classifyTier(triggers)
	state := p.network(pass.network)

	// Provisional dedupe with the cached network GVR: on a rescan this
	// skips the capture before any state-sized fetch happens.
	provisionalKey := ""

	if state.gvr != "" {
		provisionalKey = capturePath(networkID(pass.network, state.gvr), forkName, id, captureManifestName)
		if exists, err := p.corpus.Exists(ctx, provisionalKey); err == nil && exists {
			p.metrics.ObserveSkip(pass.network, SkipReasonAlreadyPromoted)

			return nil
		}
	}

	// Cheap headroom probe before the state fetch; the budget is only
	// consumed once the capture is known to be new.
	if !p.headroom(state, tier) {
		reason := tier.skipReason()

		p.metrics.ObserveSkip(pass.network, reason)
		p.log.WithFields(logrus.Fields{labelNetwork: pass.network, labelSlot: cand.slot, labelBlockRoot: cand.root, "tier": reason}).Debug("Rate cap exhausted, skipping promotion")

		return nil
	}

	// expected_state_root is the PARENT block's own state_root field - not
	// the candidate's, which is the child's POST-state root, off by exactly
	// one block. Omitted rather than substituted when the parent block is
	// missing: a missing claim is honest, a wrong one poisons.
	expectedStateRoot := ""
	if parent.peek != nil {
		expectedStateRoot = normHex(parent.peek.StateRoot.String())
	}

	stateRaw, prestate := p.resolvePrestate(ctx, pass.network, state, parent, expectedStateRoot)

	gvr := ""

	if stateRaw != nil {
		peek, err := peekState(stateRaw)
		if err != nil {
			return err
		}

		gvr = normHex(peek.GenesisValidatorsRoot.String())
	} else {
		// Block-only promotion: the network GVR must come from elsewhere
		// in the buffer. Without one the capture cannot be filed under an
		// unambiguous network id, and a wrong prefix mixes incompatible
		// chains - skip, visibly.
		gvr = state.gvr
		if gvr == "" {
			p.metrics.ObserveSkip(pass.network, SkipReasonGVRAmbiguous)
			p.log.WithFields(logrus.Fields{labelNetwork: pass.network, labelSlot: cand.slot}).Warn("No genesis validators root resolvable; refusing to label capture")

			return nil
		}
	}

	// A paired state is first-hand evidence of the network's identity, so
	// it seeds the cache when the periodic refresh has nothing yet.
	if state.gvr == "" {
		state.gvr = gvr
		state.gvrResolved = time.Now()
	}

	network := networkID(pass.network, gvr)

	// The pair's own GVR is authoritative for the capture key; re-check
	// existence when it differs from the provisional guess.
	manifestKey := capturePath(network, forkName, id, captureManifestName)
	if manifestKey != provisionalKey {
		if exists, err := p.corpus.Exists(ctx, manifestKey); err == nil && exists {
			p.metrics.ObserveSkip(pass.network, SkipReasonAlreadyPromoted)

			return nil
		}
	}

	if stateRaw != nil {
		sha := sha256Hex(stateRaw)
		if err := p.writeState(ctx, pass.network, stateRaw, sha); err != nil {
			return errors.Wrap(err, "failed to write pre-state")
		}

		prestate.SHA256 = sha
		prestate.Size = len(stateRaw)
		prestate.ExpectedStateRoot = expectedStateRoot
	}

	if _, err := p.corpus.SaveRaw(ctx, &store.SaveParams{Data: &cand.raw, Location: capturePath(network, forkName, id, captureBlockName)}); err != nil {
		return errors.Wrap(err, "failed to write block")
	}

	p.metrics.ObserveCorpusBytes(pass.network, "block", len(cand.raw))

	manifest := &Manifest{
		Schema:     manifestSchema,
		Tool:       tracoor.Full(),
		PromotedAt: time.Now().UTC(),
		Network: ManifestNetwork{
			Name:                  network,
			ConfigName:            pass.network,
			GenesisValidatorsRoot: gvr,
			SlotsPerEpoch:         p.config.SlotsPerEpoch,
			SecondsPerSlot:        uint64(p.config.SecondsPerSlot.Seconds()),
		},
		Fork: ManifestFork{Name: forkName},
		Block: ManifestBlock{
			Slot:       cand.slot,
			Root:       cand.root,
			ParentRoot: normHex(cand.peek.ParentRoot.String()),
			StateRoot:  normHex(cand.peek.StateRoot.String()),
			SHA256:     sha256Hex(cand.raw),
			Size:       len(cand.raw),
		},
		Prestate: prestate,
		Promotion: ManifestPromotion{
			Triggers: triggers,
			Decoded:  cand.decoded != nil,
			// An undecodable fork leaves the pairing claim unproven even
			// when the byte-peeked chain checks out.
			PairVerified: cand.decoded != nil && prestate != nil && prestate.ExpectedStateRoot != "",
			Branch:       branch,
			LagEpochs:    p.config.LagEpochs,
		},
		Context: ManifestContext{HeadSlotAtPromotion: pass.headSlot},
	}

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return errors.Wrap(err, "failed to marshal manifest")
	}

	if _, err := p.corpus.SaveRaw(ctx, &store.SaveParams{Data: &data, Location: manifestKey}); err != nil {
		return errors.Wrap(err, "failed to write manifest")
	}

	// The manifest is the commit point: budget is only consumed once the
	// capture actually landed, so a corpus-store outage cannot burn the
	// hourly window on failed writes.
	p.consume(state, tier)

	p.metrics.ObserveCorpusBytes(pass.network, "manifest", len(data))
	p.metrics.ObserveCapture(pass.network, branch)

	if prestate == nil {
		p.metrics.ObserveUnpaired(pass.network)
	}

	for _, trigger := range triggers {
		p.metrics.ObserveTrigger(pass.network, trigger)
	}

	p.log.WithFields(logrus.Fields{
		labelNetwork:    network,
		labelSlot:       cand.slot,
		labelBlockRoot:  cand.root,
		"triggers":      triggers,
		"branch":        branch,
		"pair_verified": manifest.Promotion.PairVerified,
	}).Info("Promoted capture to corpus")

	return nil
}

// resolvePrestate locates and verifies the parent's post-state in the buffer.
// When the parent block's own state_root is known, only the branch-exact row
// qualifies; without it, a slot-level fallback is accepted but the pairing
// stays unverified and must at least be GVR-consistent with the network.
func (p *Promoter) resolvePrestate(ctx context.Context, network string, state *networkState, parent parentInfo, expectedStateRoot string) ([]byte, *ManifestPrestate) {
	if !parent.known {
		return nil, nil
	}

	rows, err := p.db.ListBeaconState(ctx, &persistence.BeaconStateFilter{Network: &network, Slot: &parent.slot}, &persistence.PaginationCursor{Limit: 100, OrderBy: "fetched_at ASC"})
	if err != nil {
		return nil, nil
	}

	for _, row := range rows {
		if expectedStateRoot != "" && normHex(row.StateRoot) != expectedStateRoot {
			continue
		}

		data, gerr := p.buffer.GetBeaconState(ctx, row.Location)
		if gerr != nil || data == nil {
			continue
		}

		raw, derr := decompress(*data, row.ContentEncoding)
		if derr != nil {
			continue
		}

		peek, perr := peekState(raw)
		if perr != nil || peek.Slot != parent.slot {
			// Misfiled or truncated: never pair under a claim the bytes
			// contradict.
			p.metrics.ObserveSkip(network, SkipReasonMalformed)

			continue
		}

		if expectedStateRoot == "" {
			// Fallback pairing cannot be branch-verified; refuse states
			// whose GVR contradicts the network to avoid mixing recreated
			// chains under one prefix.
			gvr := normHex(peek.GenesisValidatorsRoot.String())
			if state.gvr != "" && state.gvr != gvr {
				p.metrics.ObserveSkip(network, SkipReasonGVRAmbiguous)

				continue
			}
		}

		return raw, &ManifestPrestate{
			Slot:       parent.slot,
			SourceNode: row.Node,
		}
	}

	return nil, nil
}

// writeState writes a content-addressed, uncompressed state object. The
// existence probe is an optimisation, not a correctness requirement: two
// racing deployments write identical bytes to identical keys.
func (p *Promoter) writeState(ctx context.Context, network string, raw []byte, sha string) error {
	location := statePath(sha)

	exists, err := p.corpus.Exists(ctx, location)
	if err == nil && exists {
		return nil
	}

	if _, err := p.corpus.SaveRaw(ctx, &store.SaveParams{Data: &raw, Location: location}); err != nil {
		return err
	}

	p.metrics.ObserveCorpusBytes(network, "state", len(raw))

	return nil
}

// resolveNetworkGVR reads a network's genesis validators root from its
// freshest retained state. Freshest, not any: after a devnet is recreated
// under the same name the buffer briefly holds both chains, and the newest
// state is the one that says which chain the name means now. Caching is the
// caller's business - see refreshIdentity.
func (p *Promoter) resolveNetworkGVR(ctx context.Context, network string) string {
	rows, err := p.db.ListBeaconState(ctx, &persistence.BeaconStateFilter{Network: &network}, &persistence.PaginationCursor{Limit: 5, OrderBy: "fetched_at DESC"})
	if err != nil {
		return ""
	}

	for _, row := range rows {
		data, gerr := p.buffer.GetBeaconState(ctx, row.Location)
		if gerr != nil || data == nil {
			continue
		}

		raw, derr := decompress(*data, row.ContentEncoding)
		if derr != nil {
			continue
		}

		if peek, perr := peekState(raw); perr == nil {
			return normHex(peek.GenesisValidatorsRoot.String())
		}
	}

	return ""
}

// decompress undoes the buffer's content encoding. Corpus objects are always
// written uncompressed: the sha256 of the raw SSZ is the identity.
func decompress(data []byte, contentEncoding string) ([]byte, error) {
	switch contentEncoding {
	case compression.Gzip.ContentEncoding:
		reader, err := gzip.NewReader(bytes.NewReader(data))
		if err != nil {
			return nil, err
		}
		defer reader.Close()

		return io.ReadAll(reader)
	case compression.None.ContentEncoding, "":
		return data, nil
	default:
		return nil, fmt.Errorf("unsupported content encoding: %q", contentEncoding)
	}
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)

	return hex.EncodeToString(sum[:])
}

// normHex lower-cases a hex string and guarantees a 0x prefix so roots from
// the index and roots from byte-peeks compare equal.
func normHex(s string) string {
	return "0x" + strings.ToLower(strings.TrimPrefix(s, "0x"))
}
