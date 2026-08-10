package indexer

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"time"

	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/tracoor/pkg/server/persistence"
	"github.com/ethpandaops/tracoor/pkg/store"
	"github.com/sirupsen/logrus"
)

// permanentStoreWaitTimeout bounds how long a purge waits for the permanent store to finish
// with a block. It sits above the permanent store's own lock retry window so that a block
// contended by another replica is not abandoned before that window expires.
const permanentStoreWaitTimeout = 45 * time.Second

const (
	// retentionInterval is the pause between passes once there is nothing left to purge.
	retentionInterval = time.Minute
	// purgePageSize is both the read page and the delete chunk. It is small enough to stay
	// well inside SQLite's bind-variable limit and large enough that a busy network drains in
	// a handful of round trips.
	purgePageSize = 500
	// maxPagesPerPass stops a single kind from monopolising a cycle when its backlog is huge.
	maxPagesPerPass = 40
	// maxStoreDeleteAttempts is how often an object delete is retried before the location is
	// quarantined and reported. A location that fails this often is not going to succeed.
	maxStoreDeleteAttempts = 3
	// maxQuarantineEntries bounds the retry list so a store outage cannot grow it without end.
	maxQuarantineEntries = 10000
	// rootDisagreementSlots is how far back the root-disagreement detector looks, in slots.
	rootDisagreementSlots = 64
	// maxReportedDisagreements bounds the set of already-warned slots.
	maxReportedDisagreements = 10000
)

// Blob collection skip reasons.
const (
	skipReasonReferenced = "referenced"
	skipReasonRaced      = "raced"
	skipReasonStoreError = "store_error"
	skipReasonResurrect  = "resurrected"
	skipReasonError      = "error"
)

// objectReaper owns the locations whose object outlived its row. Rows are deleted before
// objects, so a failed object delete has nothing left to drive a retry from — this list is
// that driver. A location that has failed maxStoreDeleteAttempts times is dropped and reported
// rather than retried forever.
type objectReaper struct {
	mu       sync.Mutex
	failures map[string]int
}

func newObjectReaper() *objectReaper {
	return &objectReaper{failures: make(map[string]int)}
}

// pending returns the locations owed a retry.
func (r *objectReaper) pending() []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.failures) == 0 {
		return nil
	}

	locations := make([]string, 0, len(r.failures))
	for location := range r.failures {
		locations = append(locations, location)
	}

	return locations
}

// settle records the outcome of one round: everything attempted and not named in failed is
// forgotten, everything that failed has its attempt counted, and the locations that have now
// run out of attempts are returned for the caller to report. Attempt counts survive across
// rounds, which is the whole point of keeping them.
func (r *objectReaper) settle(attempted, failed []string) []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	stillFailing := make(map[string]struct{}, len(failed))
	for _, location := range failed {
		stillFailing[location] = struct{}{}
	}

	for _, location := range attempted {
		if _, ok := stillFailing[location]; !ok {
			delete(r.failures, location)
		}
	}

	quarantined := make([]string, 0)

	for _, location := range failed {
		attempts := r.failures[location] + 1

		if attempts >= maxStoreDeleteAttempts || len(r.failures) >= maxQuarantineEntries {
			delete(r.failures, location)

			quarantined = append(quarantined, location)

			continue
		}

		r.failures[location] = attempts
	}

	return quarantined
}

// disagreementReporter remembers which (kind, network, slot) has already been warned about, so
// a disagreement that persists for hours is logged once rather than every cycle.
type disagreementReporter struct {
	mu   sync.Mutex
	seen map[string]struct{}
}

func newDisagreementReporter() *disagreementReporter {
	return &disagreementReporter{seen: make(map[string]struct{})}
}

func (d *disagreementReporter) firstTime(kind, network string, slot int64) bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	key := kind + "/" + network + "/" + strconv.FormatInt(slot, 10)

	if _, ok := d.seen[key]; ok {
		return false
	}

	// The window slides, so old entries are dead weight. Dropping the whole set costs at most
	// one repeated warning per live disagreement.
	if len(d.seen) >= maxReportedDisagreements {
		d.seen = make(map[string]struct{})
	}

	d.seen[key] = struct{}{}

	return true
}

// purgeSpec is one kind's retention job.
type purgeSpec struct {
	kind      string
	retention time.Duration
	list      func(ctx context.Context, before time.Time, limit int) ([]*persistence.ExpiringArtifact, error)
	// prepare runs before a page is deleted, for the one kind that has somewhere else to be
	// first.
	prepare func(ctx context.Context, rows []*persistence.ExpiringArtifact) error
}

func (i *Indexer) purgeSpecs() []purgeSpec {
	return []purgeSpec{
		{
			kind:      persistence.KindBeaconState,
			retention: i.config.Retention.BeaconStates.Duration,
			list:      i.db.ListExpiringBeaconStates,
		},
		{
			kind:      persistence.KindBeaconBlock,
			retention: i.config.Retention.BeaconBlocks.Duration,
			list:      i.db.ListExpiringBeaconBlocks,
			prepare:   i.archiveBlocksBeforePurge,
		},
		{
			kind:      persistence.KindExecutionPayloadEnvelope,
			retention: i.config.Retention.ExecutionPayloadEnvelopes.Duration,
			list:      i.db.ListExpiringExecutionPayloadEnvelopes,
		},
		{
			kind:      persistence.KindBeaconBadBlock,
			retention: i.config.Retention.BeaconBadBlocks.Duration,
			list:      i.db.ListExpiringBeaconBadBlocks,
		},
		{
			kind:      persistence.KindBeaconBadBlob,
			retention: i.config.Retention.BeaconBadBlobs.Duration,
			list:      i.db.ListExpiringBeaconBadBlobs,
		},
		{
			kind:      persistence.KindExecutionBlockTrace,
			retention: i.config.Retention.ExecutionBlockTraces.Duration,
			list:      i.db.ListExpiringExecutionBlockTraces,
		},
		{
			kind:      persistence.KindExecutionBadBlock,
			retention: i.config.Retention.ExecutionBadBlocks.Duration,
			list:      i.db.ListExpiringExecutionBadBlocks,
		},
	}
}

func (i *Indexer) startRetentionWatchers(ctx context.Context) {
	i.log.WithFields(logrus.Fields{
		persistence.KindBeaconState:              i.config.Retention.BeaconStates.Duration,
		persistence.KindBeaconBlock:              i.config.Retention.BeaconBlocks.Duration,
		persistence.KindExecutionPayloadEnvelope: i.config.Retention.ExecutionPayloadEnvelopes.Duration,
		persistence.KindBeaconBadBlock:           i.config.Retention.BeaconBadBlocks.Duration,
		persistence.KindBeaconBadBlob:            i.config.Retention.BeaconBadBlobs.Duration,
		persistence.KindExecutionBlockTrace:      i.config.Retention.ExecutionBlockTraces.Duration,
		persistence.KindExecutionBadBlock:        i.config.Retention.ExecutionBadBlocks.Duration,
		"blob_gc_grace":                          i.config.BlobGCGracePeriod.Duration,
	}).Info("Starting retention watcher")

	for {
		i.runRetentionCycle(ctx)

		select {
		case <-time.After(retentionInterval):
		case <-ctx.Done():
			return
		}
	}
}

// runRetentionCycle is one full pass: every artifact kind, then the payloads nothing points at
// any more, then the housekeeping that rides the same cadence.
func (i *Indexer) runRetentionCycle(ctx context.Context) {
	for _, spec := range i.purgeSpecs() {
		if err := i.purgeArtifacts(ctx, spec); err != nil {
			i.log.WithError(err).WithField(KeyKind, spec.kind).Error("Failed to purge expired rows")
		}
	}

	if err := i.purgeOrphanedBlobs(ctx); err != nil {
		i.log.WithError(err).Error("Failed to collect orphaned payloads")
	}

	if err := i.db.CleanupExpiredLocks(ctx); err != nil {
		i.log.WithError(err).Error("Failed to clean up expired locks")
	}

	i.detectRootDisagreements(ctx)
}

// purgeArtifacts drains one kind, a page at a time, until a short page says there is nothing
// left. Rows go first and objects follow: a crash between the two strands an object, which is
// recoverable, rather than an index entry pointing at nothing, which is not.
func (i *Indexer) purgeArtifacts(ctx context.Context, spec purgeSpec) error {
	before := time.Now().Add(-spec.retention)
	start := time.Now()

	var purged int64

	for page := 0; page < maxPagesPerPass; page++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		rows, err := spec.list(ctx, before, purgePageSize)
		if err != nil {
			return err
		}

		if len(rows) == 0 {
			break
		}

		if spec.prepare != nil {
			if perr := spec.prepare(ctx, rows); perr != nil {
				return perr
			}
		}

		deleted, err := i.purgePage(ctx, spec.kind, rows)
		if err != nil {
			return err
		}

		purged += deleted

		if len(rows) < purgePageSize {
			break
		}
	}

	i.metrics.ObservePurgeDuration(spec.kind, time.Since(start).Seconds())

	if purged > 0 {
		i.metrics.ObserveRowsPurged(spec.kind, purged)

		i.log.WithFields(logrus.Fields{
			KeyKind:  spec.kind,
			"before": before,
			"rows":   purged,
		}).Debug("Purged expired rows")
	}

	backlog, err := i.db.CountExpiring(ctx, spec.kind, before)
	if err != nil {
		return err
	}

	i.metrics.SetPurgeBacklog(spec.kind, backlog)

	return nil
}

// purgePage deletes one page of rows and releases what they held. A row that merely referenced
// a shared payload releases a reference; a row that owned its object takes the object with it.
func (i *Indexer) purgePage(ctx context.Context, kind string, rows []*persistence.ExpiringArtifact) (int64, error) {
	linked, err := i.resolveLinks(ctx, kind, rows)
	if err != nil {
		return 0, err
	}

	ids := make([]string, 0, len(rows))
	unlinked := make([]string, 0, len(rows))
	counts := make(map[string]*persistence.BlobRefDecrement)

	for _, row := range rows {
		ids = append(ids, row.ID)

		dedupKey, isLinked := linked[row.ID]
		if !isLinked {
			if row.Location != "" {
				unlinked = append(unlinked, row.Location)
			}

			continue
		}

		key := row.Network + "\x00" + dedupKey

		if existing, ok := counts[key]; ok {
			existing.Count++

			continue
		}

		counts[key] = &persistence.BlobRefDecrement{
			Kind:     kind,
			Network:  row.Network,
			DedupKey: dedupKey,
			Count:    1,
		}
	}

	decrements := make([]persistence.BlobRefDecrement, 0, len(counts))
	for _, d := range counts {
		decrements = append(decrements, *d)
	}

	var deleted int64

	for start := 0; start < len(ids); start += purgePageSize {
		end := start + purgePageSize
		if end > len(ids) {
			end = len(ids)
		}

		// The reference releases ride the first chunk: they are counted over the whole page
		// and applying them twice would double-release.
		chunkDecrements := decrements
		if start > 0 {
			chunkDecrements = nil
		}

		count, err := i.db.DeleteArtifacts(ctx, kind, ids[start:end], chunkDecrements)
		if err != nil {
			return deleted, err
		}

		deleted += count
	}

	i.deleteObjects(ctx, unlinked)

	return deleted, nil
}

// resolveLinks decides which rows in a page merely reference a shared payload. The rule is the
// same in both directions: a row is linked when a payload exists for its dedupe key and that
// payload lives exactly where the row says its bytes are. Anything else — a bad_* kind, or a
// copy stored beside the canonical payload because it disagreed with it — owns its object.
func (i *Indexer) resolveLinks(ctx context.Context, kind string, rows []*persistence.ExpiringArtifact) (map[string]string, error) {
	linked := make(map[string]string)

	byNetwork := make(map[string][]*persistence.ExpiringArtifact)

	for _, row := range rows {
		if row.ContentHash == "" {
			continue
		}

		byNetwork[row.Network] = append(byNetwork[row.Network], row)
	}

	if len(byNetwork) == 0 {
		return linked, nil
	}

	for network, group := range byNetwork {
		switch kind {
		case persistence.KindBeaconState, persistence.KindBeaconBlock, persistence.KindExecutionPayloadEnvelope:
			keys := make([]string, 0, len(group))
			seen := make(map[string]struct{}, len(group))

			for _, row := range group {
				key := persistence.DedupKeyFor(kind, row)
				if _, ok := seen[key]; ok {
					continue
				}

				seen[key] = struct{}{}

				keys = append(keys, key)
			}

			locations, err := i.db.BlobLinksByDedupKey(ctx, kind, network, keys)
			if err != nil {
				return nil, err
			}

			for _, row := range group {
				key := persistence.DedupKeyFor(kind, row)
				if location, ok := locations[key]; ok && location == row.Location {
					linked[row.ID] = key
				}
			}
		case persistence.KindExecutionBlockTrace:
			// A trace's dedupe key carries the agent's trace parameters, which are not columns
			// on the row, so the payload is found through the indexed content hash instead and
			// the location still decides the match.
			hashes := make([]string, 0, len(group))
			seen := make(map[string]struct{}, len(group))

			for _, row := range group {
				if _, ok := seen[row.ContentHash]; ok {
					continue
				}

				seen[row.ContentHash] = struct{}{}

				hashes = append(hashes, row.ContentHash)
			}

			keys, err := i.db.BlobLinksByLocation(ctx, kind, network, hashes)
			if err != nil {
				return nil, err
			}

			for _, row := range group {
				if key, ok := keys[row.Location]; ok {
					linked[row.ID] = key
				}
			}
		default:
			// The bad_* kinds are never deduplicated: their bytes are the evidence.
		}
	}

	return linked, nil
}

// deleteObjects removes objects that nothing references any more, together with anything a
// previous pass could not remove.
func (i *Indexer) deleteObjects(ctx context.Context, locations []string) {
	locations = append(locations, i.reaper.pending()...)

	if len(locations) == 0 {
		return
	}

	err := i.store.DeleteMany(ctx, locations)
	if err == nil {
		i.reaper.settle(locations, nil)

		return
	}

	failed := locations

	var partial *store.DeleteManyError

	if errors.As(err, &partial) {
		failed = partial.Failed
	}

	for _, location := range i.reaper.settle(locations, failed) {
		i.log.WithError(err).WithField(KeyLocation, location).
			Error("Giving up on deleting an object from the store; it is now orphaned")
	}

	i.log.WithError(err).WithField("count", len(failed)).Warn("Failed to delete objects from the store")
}

// archiveBlocksBeforePurge gives the permanent store its chance at a block before the block's
// object goes away.
func (i *Indexer) archiveBlocksBeforePurge(ctx context.Context, rows []*persistence.ExpiringArtifact) error {
	if !i.permanentStore.IsEnabled() {
		return nil
	}

	for _, row := range rows {
		block := PermanentStoreBlock{
			Location:      row.Location,
			BlockRoot:     row.Identifier,
			Network:       row.Network,
			ProcessedChan: make(chan struct{}),
			//nolint:gosec // This is a valid conversion
			Slot: phase0.Slot(row.Slot),
		}

		i.permanentStore.QueueBlock(block)

		select {
		case <-block.ProcessedChan:
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(permanentStoreWaitTimeout):
			i.log.WithField(KeyBlockRoot, row.Identifier).Warn("Timed out waiting for permanent store")
		}
	}

	return nil
}

// purgeOrphanedBlobs collects payloads nothing references any more.
//
// The reference counter only nominates candidates. Each one is then re-derived from the
// artifact tables inside a transaction, tombstoned by a conditional update that must match
// exactly one row, and only then does its object get deleted — outside the transaction,
// because that part cannot be rolled back. The row disappears last, and only if its generation
// is still the one that was tombstoned.
func (i *Indexer) purgeOrphanedBlobs(ctx context.Context) error {
	cutoff := time.Now().Add(-i.config.BlobGCGracePeriod.Duration)

	for page := 0; page < maxPagesPerPass; page++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		candidates, err := i.db.ListCollectableBlobs(ctx, cutoff, purgePageSize)
		if err != nil {
			return err
		}

		if len(candidates) == 0 {
			return nil
		}

		for _, candidate := range candidates {
			i.collectBlob(ctx, candidate)
		}

		if len(candidates) < purgePageSize {
			return nil
		}
	}

	return nil
}

func (i *Indexer) collectBlob(ctx context.Context, candidate *persistence.BlobCandidate) {
	logFields := logrus.Fields{
		KeyKind:     candidate.Kind,
		KeyNetwork:  candidate.Network,
		KeyDedupKey: candidate.DedupKey,
	}

	outcome, err := i.db.TombstoneBlob(ctx, candidate)
	if err != nil {
		i.metrics.ObserveBlobGCSkipped(skipReasonError)
		i.log.WithError(err).WithFields(logFields).Error("Failed to tombstone payload")

		return
	}

	switch outcome {
	case persistence.BlobTombstoneReferenced:
		i.metrics.ObserveBlobGCSkipped(skipReasonReferenced)
		i.log.WithFields(logFields).Debug("Payload is still referenced; repaired its reference count")

		return
	case persistence.BlobTombstoneRaced:
		i.metrics.ObserveBlobGCSkipped(skipReasonRaced)

		return
	case persistence.BlobTombstoneMarked:
	}

	// Outside the transaction on purpose: the object delete cannot be rolled back, so the
	// tombstone has to be durable before it happens.
	if derr := i.store.DeleteMany(ctx, []string{candidate.Location}); derr != nil {
		i.metrics.ObserveBlobGCSkipped(skipReasonStoreError)
		i.log.WithError(derr).WithFields(logFields).Warn("Failed to delete payload object; leaving it tombstoned for the next pass")

		return
	}

	removed, err := i.db.DeleteTombstonedBlob(ctx, candidate)
	if err != nil {
		i.metrics.ObserveBlobGCSkipped(skipReasonError)
		i.log.WithError(err).WithFields(logFields).Error("Failed to remove tombstoned payload")

		return
	}

	if !removed {
		// The payload came back while its object was being deleted. The row belongs to the new
		// generation now and is not ours to remove.
		i.metrics.ObserveBlobGCSkipped(skipReasonResurrect)

		return
	}

	i.metrics.ObserveBlobCollected()
	i.log.WithFields(logFields).Debug("Collected orphaned payload")
}

// detectRootDisagreements looks for recent slots where the network did not settle on a single
// root. Two roots for one slot is either a fork or a node serving something wrong, and both are
// worth knowing about the moment they appear.
func (i *Indexer) detectRootDisagreements(ctx context.Context) {
	for _, kind := range []string{persistence.KindBeaconState, persistence.KindBeaconBlock} {
		disagreements, err := i.db.FindRootDisagreements(ctx, kind, rootDisagreementSlots)
		if err != nil {
			i.log.WithError(err).WithField(KeyKind, kind).Error("Failed to check for root disagreements")

			continue
		}

		for _, d := range disagreements {
			i.metrics.ObserveRootDivergence(d.Network, kind)

			if !i.disagreements.firstTime(kind, d.Network, d.Slot) {
				continue
			}

			i.log.WithFields(logrus.Fields{
				KeyKind:    kind,
				KeyNetwork: d.Network,
				KeySlot:    d.Slot,
				"roots":    d.Roots,
			}).Warn("Nodes disagree on the root for this slot")
		}
	}
}
