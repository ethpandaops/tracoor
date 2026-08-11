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

// permanentStoreArchiveBudget bounds how long one page may spend in total handing blocks to
// the permanent store. Without it a page of rows against a contended store waits the per-block
// timeout hundreds of times over, and because the cycle is sequential nothing else purges, no
// payloads are collected and no expired locks are cleaned for hours. Rows that do not get
// their turn inside the budget keep their objects and are offered again next cycle.
const permanentStoreArchiveBudget = 30 * time.Second

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
	// maxBlobCollectAttempts is how often a payload that cannot be evaluated is retried before
	// it is quarantined and reported. It mirrors the object reaper's allowance.
	maxBlobCollectAttempts = 3
	// kindPayloadDivergence labels the divergence log in the retention metrics. It is not an
	// artifact kind, but it is purged on the same cadence and reported the same way.
	kindPayloadDivergence = "payload_divergence"
)

// Blob collection skip reasons.
const (
	skipReasonReferenced  = "referenced"
	skipReasonRaced       = "raced"
	skipReasonStoreError  = "store_error"
	skipReasonResurrect   = "resurrected"
	skipReasonError       = "error"
	skipReasonQuarantined = "quarantined"
)

// Why a block row was held back from purge instead of released with its page.
const (
	holdbackReasonBudgetExhausted = "budget_exhausted"
	holdbackReasonDivergent       = "divergent"
	holdbackReasonUnconfirmed     = "unconfirmed"
)

// Why a slot ended up with more than one root.
const (
	// causeReorg is the ordinary explanation: the chain reorganised and nodes recorded both
	// sides of it. Expected on a healthy network.
	causeReorg = "reorg"
	// causeDivergence means some node also served bytes that disagreed with the canonical
	// payload for that slot. That is not a fork; that is a node to go and look at.
	causeDivergence = "divergence"
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

// blobQuarantine bounds how often a payload that cannot be evaluated is retried.
//
// The reference re-derivation parses the blob's kind and dedupe key, so a row written with a
// shape it does not understand fails every time it is looked at. Candidates come back oldest
// first and a failed one does not go away, so without an allowance such a row is re-attempted
// for ever and never stops being reported.
type blobQuarantine struct {
	mu       sync.Mutex
	failures map[string]int
}

func newBlobQuarantine() *blobQuarantine {
	return &blobQuarantine{failures: make(map[string]int)}
}

func blobQuarantineKey(c *persistence.BlobCandidate) string {
	return c.Kind + "\x00" + c.Network + "\x00" + c.DedupKey
}

// quarantined reports whether a payload has already used up its attempts.
func (q *blobQuarantine) quarantined(key string) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	return q.failures[key] >= maxBlobCollectAttempts
}

// fail counts a failed evaluation and reports whether this attempt was the one that exhausted
// the allowance, so the caller can report it exactly once.
func (q *blobQuarantine) fail(key string) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	// A store outage must not let the list grow without end. Dropping it costs at most one
	// repeated report per quarantined payload.
	if len(q.failures) >= maxQuarantineEntries {
		q.failures = make(map[string]int)
	}

	q.failures[key]++

	return q.failures[key] == maxBlobCollectAttempts
}

// forget drops a payload's history once it has been evaluated without error.
func (q *blobQuarantine) forget(key string) {
	q.mu.Lock()
	defer q.mu.Unlock()

	delete(q.failures, key)
}

// disagreementReporter remembers which (kind, network, slot, root count) has already been
// reported, so a disagreement that persists for hours is counted and logged once rather than
// once per cycle. The root count is part of the key: a slot that grows a third root is a new
// observation, not the same one again.
type disagreementReporter struct {
	mu   sync.Mutex
	seen map[string]struct{}
}

func newDisagreementReporter() *disagreementReporter {
	return &disagreementReporter{seen: make(map[string]struct{})}
}

func (d *disagreementReporter) firstTime(kind, network string, slot, roots int64) bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	key := kind + "/" + network + "/" + strconv.FormatInt(slot, 10) + "/" + strconv.FormatInt(roots, 10)

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
	// first. It returns the rows that may now be purged, which is not always all of them: a
	// row whose block did not get its turn keeps its object and is offered again next cycle.
	prepare func(ctx context.Context, rows []*persistence.ExpiringArtifact) ([]*persistence.ExpiringArtifact, error)
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
		kindPayloadDivergence:                    i.config.Retention.PayloadDivergences.Duration,
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

	if err := i.purgePayloadDivergences(ctx); err != nil {
		i.log.WithError(err).Error("Failed to purge expired payload divergences")
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
	before := time.Now().UTC().Add(-spec.retention)
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

		purgeable := rows

		if spec.prepare != nil {
			purgeable, err = spec.prepare(ctx, rows)
			if err != nil {
				return err
			}
		}

		deleted, err := i.purgePage(ctx, spec.kind, purgeable)
		if err != nil {
			return err
		}

		purged += deleted

		// A page that prepare could not clear in full has to stop the pass: the rows it held
		// back are the same ones the next page would read.
		if len(purgeable) < len(rows) {
			break
		}

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
	unlinkedSeen := make(map[string]struct{}, len(rows))
	counts := make(map[string]*persistence.BlobRefDecrement)

	for _, row := range rows {
		ids = append(ids, row.ID)

		dedupKey, isLinked := linked[row.ID]
		if !isLinked {
			if _, seen := unlinkedSeen[row.Location]; row.Location != "" && !seen {
				unlinkedSeen[row.Location] = struct{}{}

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

		count, derr := i.db.DeleteArtifacts(ctx, kind, ids[start:end], chunkDecrements)
		if derr != nil {
			return deleted, derr
		}

		deleted += count
	}

	orphaned, err := i.unreferencedLocations(ctx, kind, unlinked)
	if err != nil {
		return deleted, err
	}

	i.deleteObjects(ctx, orphaned)

	return deleted, nil
}

// unreferencedLocations drops from a page's owned locations any that a surviving row still
// points at. Ownership is per row until the location turns out to be shared, which is exactly
// what a divergent copy is: content-addressed, no node in the path, one object written by
// every node that served those bytes. The rows are already gone by the time this runs, so
// anything the query still finds is a sibling that outlived this page.
func (i *Indexer) unreferencedLocations(ctx context.Context, kind string, locations []string) ([]string, error) {
	if len(locations) == 0 {
		return nil, nil
	}

	referenced, err := i.db.LocationsStillReferenced(ctx, kind, locations)
	if err != nil {
		return nil, err
	}

	orphaned := make([]string, 0, len(locations))

	for _, location := range locations {
		if _, ok := referenced[location]; ok {
			continue
		}

		orphaned = append(orphaned, location)
	}

	return orphaned, nil
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
// object goes away, and returns only the rows whose archive is confirmed.
//
// The whole page shares one budget. A row that is not reached inside it — or whose wait times
// out, whose block the queue refused, or whose worker failed — is left out of the purge
// entirely rather than deleted unarchived, so the only cost of a slow permanent store is that
// these blocks wait for the next cycle, not that the archive silently loses them.
func (i *Indexer) archiveBlocksBeforePurge(ctx context.Context, rows []*persistence.ExpiringArtifact) ([]*persistence.ExpiringArtifact, error) {
	if !i.permanentStore.IsEnabled() {
		return rows, nil
	}

	deadline := time.Now().Add(i.archiveBudget)

	archived := make([]*persistence.ExpiringArtifact, 0, len(rows))

	for index, row := range rows {
		if remaining := time.Until(deadline); remaining <= 0 {
			i.metrics.ObserveArchiveHoldback(holdbackReasonBudgetExhausted, int64(len(rows)-index))
			i.log.WithFields(logrus.Fields{
				"remaining": len(rows) - index,
				"budget":    i.archiveBudget,
			}).Warn("Permanent store archiving ran out of budget; the rest of the page keeps its objects for now")

			break
		}

		divergent, err := i.rowDivergesFromBlob(ctx, row)
		if err != nil {
			return nil, err
		}

		if divergent {
			i.metrics.ObserveArchiveHoldback(holdbackReasonDivergent, 1)
			i.log.WithFields(logrus.Fields{
				KeyBlockRoot: row.Identifier,
				KeyNetwork:   row.Network,
			}).Warn("Holding back a block whose bytes disagree with the canonical payload; a canonical row archives this root instead")

			continue
		}

		block := PermanentStoreBlock{
			Location:  row.Location,
			BlockRoot: row.Identifier,
			Network:   row.Network,
			// Buffered so the worker's result survives a waiter that has already timed out.
			ProcessedChan: make(chan PermanentStoreResult, 1),
			//nolint:gosec // This is a valid conversion
			Slot: phase0.Slot(row.Slot),
		}

		i.permanentStore.QueueBlock(block)

		wait := time.NewTimer(min(permanentStoreWaitTimeout, time.Until(deadline)))

		confirmed := false

		select {
		case result := <-block.ProcessedChan:
			confirmed = result.Archived
		case <-ctx.Done():
			wait.Stop()

			return nil, ctx.Err()
		case <-wait.C:
			i.log.WithField(KeyBlockRoot, row.Identifier).Warn("Timed out waiting for permanent store; the row keeps its object for now")
		}

		wait.Stop()

		if !confirmed {
			i.metrics.ObserveArchiveHoldback(holdbackReasonUnconfirmed, 1)

			continue
		}

		archived = append(archived, row)
	}

	return archived, nil
}

// rowDivergesFromBlob reports whether a block row's bytes are known to disagree with the
// canonical payload for its root, in which case archiving it would preserve a divergent copy
// as the permanent record. A row with no recorded hash predates verification and is taken at
// face value. A hashed row whose blob is already gone is archived as-is: when only divergent
// copies were ever captured, keeping those bytes beats losing the block entirely — at the
// cost that the archive cannot tell such a copy apart from a clean one.
func (i *Indexer) rowDivergesFromBlob(ctx context.Context, row *persistence.ExpiringArtifact) (bool, error) {
	if row.ContentHash == "" {
		return false, nil
	}

	dedupKey := persistence.DedupKeyFor(persistence.KindBeaconBlock, row)

	blob, err := i.db.GetBlob(ctx, persistence.KindBeaconBlock, row.Network, dedupKey)
	if err != nil {
		if errors.Is(err, persistence.ErrBlobNotFound) {
			return false, nil
		}

		return false, err
	}

	return blob.ContentHash != "" && blob.ContentHash != row.ContentHash, nil
}

// purgeOrphanedBlobs collects payloads nothing references any more.
//
// The reference counter only nominates candidates. Each one is then re-derived from the
// artifact tables inside a transaction, tombstoned by a conditional update that must match
// exactly one row, and only then does its object get deleted — outside the transaction,
// because that part cannot be rolled back. The row disappears last, and only if its generation
// is still the one that was tombstoned.
func (i *Indexer) purgeOrphanedBlobs(ctx context.Context) error {
	cutoff := time.Now().UTC().Add(-i.config.BlobGCGracePeriod.Duration)

	// Candidates that were looked at and left in place are still candidates, and they sort
	// ahead of everything else. Stepping over them is what lets a later page be reached at all.
	offset := 0

	for page := 0; page < maxPagesPerPass; page++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		candidates, err := i.db.ListCollectableBlobs(ctx, cutoff, purgePageSize, offset)
		if err != nil {
			return err
		}

		if len(candidates) == 0 {
			return nil
		}

		for _, candidate := range candidates {
			if !i.collectBlob(ctx, candidate) {
				offset++
			}
		}

		if len(candidates) < purgePageSize {
			return nil
		}
	}

	return nil
}

// collectBlob evaluates one candidate and reports whether it left the candidate set.
func (i *Indexer) collectBlob(ctx context.Context, candidate *persistence.BlobCandidate) bool {
	logFields := logrus.Fields{
		KeyKind:     candidate.Kind,
		KeyNetwork:  candidate.Network,
		KeyDedupKey: candidate.DedupKey,
	}

	quarantineKey := blobQuarantineKey(candidate)

	if i.blobs.quarantined(quarantineKey) {
		i.metrics.ObserveBlobGCSkipped(skipReasonQuarantined)

		return false
	}

	// A candidate already in the deleting state was tombstoned on an earlier pass whose object
	// delete failed; it owes the store another attempt, not another tombstone. Its generation
	// was captured back then, so the resurrect guard below still holds.
	if candidate.State != persistence.BlobStateDeleting {
		outcome, err := i.db.TombstoneBlob(ctx, candidate)
		if err != nil {
			i.metrics.ObserveBlobGCSkipped(skipReasonError)

			if i.blobs.fail(quarantineKey) {
				i.log.WithError(err).WithFields(logFields).
					Error("Giving up on collecting this payload; it will be skipped until the process restarts")
			} else {
				i.log.WithError(err).WithFields(logFields).Error("Failed to tombstone payload")
			}

			return false
		}

		i.blobs.forget(quarantineKey)

		switch outcome {
		case persistence.BlobTombstoneReferenced:
			i.metrics.ObserveBlobGCSkipped(skipReasonReferenced)
			i.log.WithFields(logFields).Debug("Payload is still referenced; repaired its reference count")

			return true
		case persistence.BlobTombstoneRaced:
			i.metrics.ObserveBlobGCSkipped(skipReasonRaced)

			return false
		case persistence.BlobTombstoneMarked:
		}
	}

	// Outside the transaction on purpose: the object delete cannot be rolled back, so the
	// tombstone has to be durable before it happens.
	if derr := i.store.DeleteMany(ctx, []string{candidate.Location}); derr != nil {
		i.metrics.ObserveBlobGCSkipped(skipReasonStoreError)

		// The row stays tombstoned and is offered again, within the same bounded allowance a
		// payload that cannot be evaluated gets.
		if i.blobs.fail(quarantineKey) {
			i.log.WithError(derr).WithFields(logFields).
				Error("Giving up on deleting this payload's object; it will be skipped until the process restarts")
		} else {
			i.log.WithError(derr).WithFields(logFields).Warn("Failed to delete payload object; leaving it tombstoned for the next pass")
		}

		return false
	}

	i.blobs.forget(quarantineKey)

	removed, err := i.db.DeleteTombstonedBlob(ctx, candidate)
	if err != nil {
		i.metrics.ObserveBlobGCSkipped(skipReasonError)
		i.log.WithError(err).WithFields(logFields).Error("Failed to remove tombstoned payload")

		// The row stays tombstoned; the next pass retries the delete, which tolerates the
		// object already being gone.
		return false
	}

	if !removed {
		// The payload came back while its object was being deleted. The row belongs to the new
		// generation now and is not ours to remove.
		i.metrics.ObserveBlobGCSkipped(skipReasonResurrect)

		return true
	}

	i.metrics.ObserveBlobCollected()
	i.log.WithFields(logFields).Debug("Collected orphaned payload")

	return true
}

// purgePayloadDivergences trims the divergence log. It has no objects and no references, so it
// is a plain paged delete on the indexed timestamp.
func (i *Indexer) purgePayloadDivergences(ctx context.Context) error {
	before := time.Now().UTC().Add(-i.config.Retention.PayloadDivergences.Duration)
	start := time.Now()

	var purged int64

	for page := 0; page < maxPagesPerPass; page++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		deleted, err := i.db.DeleteExpiredPayloadDivergences(ctx, before, purgePageSize)
		if err != nil {
			return err
		}

		purged += deleted

		if deleted < purgePageSize {
			break
		}
	}

	i.metrics.ObservePurgeDuration(kindPayloadDivergence, time.Since(start).Seconds())

	if purged > 0 {
		i.metrics.ObserveRowsPurged(kindPayloadDivergence, purged)
	}

	backlog, err := i.db.CountExpiringPayloadDivergences(ctx, before)
	if err != nil {
		return err
	}

	i.metrics.SetPurgeBacklog(kindPayloadDivergence, backlog)

	return nil
}

// detectRootDisagreements looks for recent slots where the network did not settle on a single
// root.
//
// The counter is advanced once per distinct observation, not once per pass: the detector runs
// every cycle over a window many minutes wide, so counting every pass would make the metric a
// measure of the cadence rather than of events. Each observation is also given a cause, because
// two roots for a slot is the ordinary result of a reorg and must not be confused with a node
// serving bytes nobody else agrees with.
func (i *Indexer) detectRootDisagreements(ctx context.Context) {
	for _, kind := range []string{persistence.KindBeaconState, persistence.KindBeaconBlock} {
		disagreements, err := i.db.FindRootDisagreements(ctx, kind, rootDisagreementSlots)
		if err != nil {
			i.log.WithError(err).WithField(KeyKind, kind).Error("Failed to check for root disagreements")

			continue
		}

		unexplained := i.divergentSlots(ctx, kind, disagreements)

		for _, d := range disagreements {
			if !i.disagreements.firstTime(kind, d.Network, d.Slot, d.Roots) {
				continue
			}

			cause := causeReorg

			if slots, ok := unexplained[d.Network]; ok {
				if _, found := slots[d.Slot]; found {
					cause = causeDivergence
				}
			}

			i.metrics.ObserveRootDivergence(d.Network, kind, cause)

			i.log.WithFields(logrus.Fields{
				KeyKind:    kind,
				KeyNetwork: d.Network,
				KeySlot:    d.Slot,
				"roots":    d.Roots,
				"cause":    cause,
			}).Warn("Nodes disagree on the root for this slot")
		}
	}
}

// divergentSlots returns, per network, the slots a node was caught serving bytes that did not
// match the canonical payload. A reorg produces distinct roots and therefore distinct dedupe
// keys, so it never leaves a divergence record; anything that does is a node to look at. The
// lookup only runs when there is a disagreement to explain.
func (i *Indexer) divergentSlots(ctx context.Context, kind string, disagreements []*persistence.RootDisagreement) map[string]map[int64]struct{} {
	from := make(map[string]int64, len(disagreements))

	for _, d := range disagreements {
		if lowest, ok := from[d.Network]; !ok || d.Slot < lowest {
			from[d.Network] = d.Slot
		}
	}

	divergent := make(map[string]map[int64]struct{}, len(from))

	for network, lowest := range from {
		slots, err := i.db.SlotsWithDivergence(ctx, network, kind, lowest)
		if err != nil {
			// Without the evidence the disagreement is simply unexplained, which is the
			// ordinary case. Reporting it as a divergence on a failed lookup would page.
			i.log.WithError(err).WithFields(logrus.Fields{
				KeyKind:    kind,
				KeyNetwork: network,
			}).Warn("Failed to look up payload divergences for a disagreeing slot")

			continue
		}

		divergent[network] = slots
	}

	return divergent
}
