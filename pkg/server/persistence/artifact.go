package persistence

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Artifact kinds. The vocabulary is the store's data-type vocabulary verbatim so that a kind
// read out of a blob row can be handed straight to the store without translation.
const (
	KindBeaconState              = "beacon_state"
	KindBeaconBlock              = "beacon_block"
	KindExecutionPayloadEnvelope = "execution_payload_envelope"
	KindBeaconBadBlock           = "beacon_bad_block"
	KindBeaconBadBlob            = "beacon_bad_blob"
	KindExecutionBlockTrace      = "execution_block_trace"
	KindExecutionBadBlock        = "execution_bad_block"
)

// ArtifactInsertOutcome is what an insert did, which decides both the RPC status and whether
// the caller still owes the store a HEAD request.
type ArtifactInsertOutcome int

const (
	// ArtifactInsertDuplicate means the unique index already held an identical observation.
	ArtifactInsertDuplicate ArtifactInsertOutcome = iota
	// ArtifactInsertLinked means the row was inserted and took a reference on a ready blob.
	ArtifactInsertLinked
	// ArtifactInsertUnlinked means the row was inserted and owns its object outright: a
	// bad_* kind, or a divergent copy stored beside the canonical blob.
	ArtifactInsertUnlinked
	// ArtifactInsertUnlinkable means the row claimed a payload that is not linkable — no blob
	// for the dedupe key, or one that is already being collected. Nothing was inserted.
	ArtifactInsertUnlinkable
)

// errUnlinkable rolls back an insert whose payload turned out not to exist. It never escapes
// InsertArtifact.
var errUnlinkable = errors.New("blob not linkable")

// InsertArtifactParams describes one artifact write.
type InsertArtifactParams struct {
	// Row is the model to insert; ConflictColumns must name that table's unique index
	// exactly, as a set.
	Row             any
	ConflictColumns []string

	// Kind, Network and DedupKey identify the blob the row may reference. An empty DedupKey
	// or ContentHash means the row references no blob at all.
	Kind        string
	Network     string
	DedupKey    string
	ContentHash string

	// Location is where the row's own bytes live. A reference is only taken when the blob
	// agrees on it: the same dedupe key stored at a different location is a divergent copy,
	// not a second reference.
	Location string
}

// InsertArtifact inserts one artifact row and, when the row references a blob, takes the
// reference in the same transaction. The unique index is the arbiter of duplication; the
// conditional reference bump is the arbiter of linkage. Nothing here reads before it writes.
func (i *Indexer) InsertArtifact(ctx context.Context, p *InsertArtifactParams) (ArtifactInsertOutcome, error) {
	columns := make([]clause.Column, 0, len(p.ConflictColumns))
	for _, name := range p.ConflictColumns {
		columns = append(columns, clause.Column{Name: name})
	}

	outcome := ArtifactInsertDuplicate

	err := i.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		insert := tx.Clauses(clause.OnConflict{Columns: columns, DoNothing: true}).Create(p.Row)
		if insert.Error != nil {
			return insert.Error
		}

		if insert.RowsAffected == 0 {
			outcome = ArtifactInsertDuplicate

			return nil
		}

		if p.DedupKey == "" || p.ContentHash == "" {
			outcome = ArtifactInsertUnlinked

			return nil
		}

		link := tx.Model(&Blob{}).
			Where("kind = ? AND network = ? AND dedup_key = ? AND state = ? AND location = ?",
				p.Kind, p.Network, p.DedupKey, BlobStateReady, p.Location).
			UpdateColumn("ref_count", gorm.Expr("ref_count + 1"))
		if link.Error != nil {
			return link.Error
		}

		if link.RowsAffected > 0 {
			outcome = ArtifactInsertLinked

			return nil
		}

		// Nothing was linked. A ready blob recorded under this dedupe key at some other
		// location means these bytes disagree with the canonical copy, which is a legitimate
		// row that owns its own object. Anything else means the payload the row points at is
		// gone or going, and the row must not survive.
		var elsewhere int64

		if err := tx.Model(&Blob{}).
			Where("kind = ? AND network = ? AND dedup_key = ? AND state = ?",
				p.Kind, p.Network, p.DedupKey, BlobStateReady).
			Count(&elsewhere).Error; err != nil {
			return err
		}

		if elsewhere > 0 {
			outcome = ArtifactInsertUnlinked

			return nil
		}

		outcome = ArtifactInsertUnlinkable

		return errUnlinkable
	})
	if err != nil && !errors.Is(err, errUnlinkable) {
		return outcome, err
	}

	return outcome, nil
}

// ExpiringArtifact is the narrow projection retention needs: enough to identify the row, find
// its object and work out which blob it references. Deliberately not the whole row.
type ExpiringArtifact struct {
	ID          string
	Location    string
	ContentHash string
	Network     string
	Slot        int64
	// Identifier is the root or hash that completes the dedupe key: state_root, block_root
	// or block_hash depending on the kind.
	Identifier  string
	BlockNumber int64
}

// DedupKeyFor builds the dedupe key for the kinds whose key is derivable from the row itself.
// Execution block traces are not: their key carries the agent's trace parameters, which are
// deliberately not columns on the artifact table, so their blob is resolved by content hash
// instead.
func DedupKeyFor(kind string, row *ExpiringArtifact) string {
	switch kind {
	case KindBeaconState, KindBeaconBlock, KindExecutionPayloadEnvelope:
		return strconv.FormatInt(row.Slot, 10) + "/" + row.Identifier
	default:
		return ""
	}
}

func (i *Indexer) listExpiring(ctx context.Context, operation Operation, model any, projection string, before time.Time, limit, offset int) ([]*ExpiringArtifact, error) {
	i.metrics.ObserveOperation(operation)

	var rows []*ExpiringArtifact

	result := i.db.WithContext(ctx).
		Model(model).
		Select(projection).
		Where("fetched_at <= ?", utcBound(before)).
		Order("fetched_at ASC").
		Limit(limit).
		Offset(offset).
		Scan(&rows)
	if result.Error != nil {
		i.metrics.ObserveOperationError(operation)

		return nil, result.Error
	}

	return rows, nil
}

func (i *Indexer) ListExpiringBeaconStates(ctx context.Context, before time.Time, limit, offset int) ([]*ExpiringArtifact, error) {
	return i.listExpiring(ctx, OperationListBeaconState, &BeaconState{},
		"id, location, content_hash, network, slot, state_root AS identifier", before, limit, offset)
}

func (i *Indexer) ListExpiringBeaconBlocks(ctx context.Context, before time.Time, limit, offset int) ([]*ExpiringArtifact, error) {
	return i.listExpiring(ctx, OperationListBeaconBlock, &BeaconBlock{},
		"id, location, content_hash, network, slot, block_root AS identifier", before, limit, offset)
}

func (i *Indexer) ListExpiringExecutionPayloadEnvelopes(ctx context.Context, before time.Time, limit, offset int) ([]*ExpiringArtifact, error) {
	return i.listExpiring(ctx, OperationListExecutionPayloadEnvelope, &ExecutionPayloadEnvelope{},
		"id, location, content_hash, network, slot, block_root AS identifier", before, limit, offset)
}

func (i *Indexer) ListExpiringBeaconBadBlocks(ctx context.Context, before time.Time, limit, offset int) ([]*ExpiringArtifact, error) {
	return i.listExpiring(ctx, OperationListBeaconBadBlock, &BeaconBadBlock{},
		"id, location, content_hash, network, slot, block_root AS identifier", before, limit, offset)
}

func (i *Indexer) ListExpiringBeaconBadBlobs(ctx context.Context, before time.Time, limit, offset int) ([]*ExpiringArtifact, error) {
	return i.listExpiring(ctx, OperationListBeaconBadBlob, &BeaconBadBlob{},
		"id, location, content_hash, network, slot, block_root AS identifier", before, limit, offset)
}

func (i *Indexer) ListExpiringExecutionBlockTraces(ctx context.Context, before time.Time, limit, offset int) ([]*ExpiringArtifact, error) {
	return i.listExpiring(ctx, OperationListExecutionBlockTrace, &ExecutionBlockTrace{},
		"id, location, content_hash, network, block_number, block_hash AS identifier", before, limit, offset)
}

func (i *Indexer) ListExpiringExecutionBadBlocks(ctx context.Context, before time.Time, limit, offset int) ([]*ExpiringArtifact, error) {
	return i.listExpiring(ctx, OperationListExecutionBadBlock, &ExecutionBadBlock{},
		"id, location, content_hash, network, block_hash AS identifier", before, limit, offset)
}

// CountExpiring reports how many rows are still past their retention cutoff. It is the purge
// backlog, and it is a plain indexed count so it can be sampled every cycle.
func (i *Indexer) CountExpiring(ctx context.Context, kind string, before time.Time) (int64, error) {
	model, err := artifactModel(kind)
	if err != nil {
		return 0, err
	}

	var count int64

	if err := i.db.WithContext(ctx).Model(model).Where("fetched_at <= ?", utcBound(before)).Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}

// BlobRefDecrement is a reference release: one blob, and how many of its referencing rows are
// going away in this batch.
type BlobRefDecrement struct {
	Kind     string
	Network  string
	DedupKey string
	Count    int64
}

// DeleteArtifacts removes a batch of rows and releases the references they held, in one
// transaction. Rows go first and objects follow, so a crash between the two leaves an orphaned
// object — recoverable — rather than a row pointing at nothing.
func (i *Indexer) DeleteArtifacts(ctx context.Context, kind string, ids []string, decrements []BlobRefDecrement) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	model, err := artifactModel(kind)
	if err != nil {
		return 0, err
	}

	var deleted int64

	err = i.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Where("id IN ?", ids).Delete(model)
		if result.Error != nil {
			return result.Error
		}

		deleted = result.RowsAffected

		for _, d := range decrements {
			if d.Count <= 0 {
				continue
			}

			// The floor is applied in SQL so a drifted counter cannot go negative and turn
			// every later blob into a permanent GC candidate.
			if derr := tx.Model(&Blob{}).
				Where("kind = ? AND network = ? AND dedup_key = ?", d.Kind, d.Network, d.DedupKey).
				UpdateColumn("ref_count", gorm.Expr("CASE WHEN ref_count > ? THEN ref_count - ? ELSE 0 END", d.Count, d.Count)).
				Error; derr != nil {
				return derr
			}
		}

		return nil
	})
	if err != nil {
		return 0, err
	}

	return deleted, nil
}

// LocationsStillReferenced narrows a set of locations to the ones rows of this kind still
// point at.
//
// An unlinked row usually owns its object outright, but not always: a copy stored because it
// disagreed with the canonical payload is addressed by its content alone, with no node in the
// path, so every node that served those same bytes wrote the same object and every one of
// their rows is unlinked. Deleting on the strength of one page would empty a location the rest
// of that group still advertises.
func (i *Indexer) LocationsStillReferenced(ctx context.Context, kind string, locations []string) (map[string]struct{}, error) {
	referenced := make(map[string]struct{}, len(locations))

	if len(locations) == 0 {
		return referenced, nil
	}

	model, err := artifactModel(kind)
	if err != nil {
		return nil, err
	}

	for start := 0; start < len(locations); start += lookupChunkSize {
		end := start + lookupChunkSize
		if end > len(locations) {
			end = len(locations)
		}

		var found []string

		if err := i.db.WithContext(ctx).Model(model).
			Where("location IN ?", locations[start:end]).
			Distinct().
			Pluck("location", &found).Error; err != nil {
			return nil, err
		}

		for _, location := range found {
			referenced[location] = struct{}{}
		}
	}

	return referenced, nil
}

// BlobLink is the linkage evidence for one artifact row: which blob it belongs to and where
// that blob's object actually lives.
type BlobLink struct {
	DedupKey string
	Location string
}

// BlobLinksByDedupKey resolves blobs by their primary key. Tombstoned blobs are included: a
// row still counts as linked to a blob that is being collected, and must not delete the object
// out from under the collector.
func (i *Indexer) BlobLinksByDedupKey(ctx context.Context, kind, network string, dedupKeys []string) (map[string]string, error) {
	links := make(map[string]string, len(dedupKeys))

	for start := 0; start < len(dedupKeys); start += lookupChunkSize {
		end := start + lookupChunkSize
		if end > len(dedupKeys) {
			end = len(dedupKeys)
		}

		var rows []BlobLink

		if err := i.db.WithContext(ctx).Model(&Blob{}).
			Select("dedup_key, location").
			Where("kind = ? AND network = ? AND dedup_key IN ?", kind, network, dedupKeys[start:end]).
			Scan(&rows).Error; err != nil {
			return nil, err
		}

		for _, row := range rows {
			links[row.DedupKey] = row.Location
		}
	}

	return links, nil
}

// BlobLinksByLocation resolves blobs for kinds whose dedupe key cannot be rebuilt from the
// artifact row. The content hash index narrows the search and the location decides the match,
// which is the same linkage rule stated the other way round.
func (i *Indexer) BlobLinksByLocation(ctx context.Context, kind, network string, contentHashes []string) (map[string]string, error) {
	links := make(map[string]string, len(contentHashes))

	for start := 0; start < len(contentHashes); start += lookupChunkSize {
		end := start + lookupChunkSize
		if end > len(contentHashes) {
			end = len(contentHashes)
		}

		var rows []BlobLink

		if err := i.db.WithContext(ctx).Model(&Blob{}).
			Select("dedup_key, location").
			Where("kind = ? AND network = ? AND content_hash IN ?", kind, network, contentHashes[start:end]).
			Scan(&rows).Error; err != nil {
			return nil, err
		}

		for _, row := range rows {
			links[row.Location] = row.DedupKey
		}
	}

	return links, nil
}

// AgreementCountsByLocation returns, per blob location, how many rows currently reference the
// ready blob stored there — the number of nodes whose verified payload agrees. A linked row
// carries its blob's location, so the location is the row→blob join that stays inside one
// dedupe key: grouping by content hash instead would merge blobs that happen to share bytes
// across keys — routine for traces, where an empty trace is identical JSON on every client
// build, yet the count is presented per build. A location with no ready blob is absent from
// the map: its agreement is unknown, not zero, and the caller must not dress it up as either.
func (i *Indexer) AgreementCountsByLocation(ctx context.Context, kind, network string, locations []string) (map[string]int64, error) {
	counts := make(map[string]int64, len(locations))

	for start := 0; start < len(locations); start += lookupChunkSize {
		end := start + lookupChunkSize
		if end > len(locations) {
			end = len(locations)
		}

		var rows []struct {
			Location  string
			Agreement int64
		}

		if err := i.db.WithContext(ctx).Model(&Blob{}).
			Select("location, SUM(ref_count) AS agreement").
			Where("kind = ? AND network = ? AND location IN ? AND state = ?",
				kind, network, locations[start:end], BlobStateReady).
			Group("location").
			Scan(&rows).Error; err != nil {
			return nil, err
		}

		for _, row := range rows {
			counts[row.Location] = row.Agreement
		}
	}

	return counts, nil
}

// RootDisagreement is one slot on which nodes reported more than one root.
type RootDisagreement struct {
	Network string
	Slot    int64
	Roots   int64
}

// FindRootDisagreements reports slots within the recent window where the network did not agree
// on a single root. The group rides the leading columns of the dedupe index.
func (i *Indexer) FindRootDisagreements(ctx context.Context, kind string, slotWindow int64) ([]*RootDisagreement, error) {
	model, err := artifactModel(kind)
	if err != nil {
		return nil, err
	}

	var rootColumn string

	switch kind {
	case KindBeaconState:
		rootColumn = KeyStateRoot
	case KindBeaconBlock, KindExecutionPayloadEnvelope:
		rootColumn = KeyBlockRoot
	default:
		return nil, fmt.Errorf("kind %q has no root to disagree on", kind)
	}

	type networkHead struct {
		Network string
		MaxSlot int64
	}

	var heads []networkHead

	if err := i.db.WithContext(ctx).Model(model).
		Select("network, MAX(slot) AS max_slot").
		Group(KeyNetwork).
		Scan(&heads).Error; err != nil {
		return nil, err
	}

	disagreements := make([]*RootDisagreement, 0)

	for _, head := range heads {
		from := head.MaxSlot - slotWindow
		if from < 0 {
			from = 0
		}

		var found []*RootDisagreement

		if err := i.db.WithContext(ctx).Model(model).
			Select("network, slot, COUNT(DISTINCT "+rootColumn+") AS roots").
			Where("network = ? AND slot >= ?", head.Network, from).
			Group("network, slot").
			Having("COUNT(DISTINCT " + rootColumn + ") > 1").
			Scan(&found).Error; err != nil {
			return nil, err
		}

		disagreements = append(disagreements, found...)
	}

	return disagreements, nil
}

// CleanupExpiredLocks removes lock rows whose lease has run out. Acquisition does not depend on
// it; it only keeps abandoned rows from accumulating.
func (i *Indexer) CleanupExpiredLocks(ctx context.Context) error {
	return i.cleanupExpiredLocks(ctx)
}

// lookupChunkSize bounds how many bind variables a lookup uses, well under SQLite's limit.
const lookupChunkSize = 500

func artifactModel(kind string) (any, error) {
	switch kind {
	case KindBeaconState:
		return &BeaconState{}, nil
	case KindBeaconBlock:
		return &BeaconBlock{}, nil
	case KindExecutionPayloadEnvelope:
		return &ExecutionPayloadEnvelope{}, nil
	case KindBeaconBadBlock:
		return &BeaconBadBlock{}, nil
	case KindBeaconBadBlob:
		return &BeaconBadBlob{}, nil
	case KindExecutionBlockTrace:
		return &ExecutionBlockTrace{}, nil
	case KindExecutionBadBlock:
		return &ExecutionBadBlock{}, nil
	default:
		return nil, fmt.Errorf("unknown artifact kind: %s", kind)
	}
}

// countLiveReferences re-derives how many artifact rows actually reference a blob. It is the
// authority the reference counter is only a hint for, so it matches on the same evidence the
// link path used: the dedupe key's own columns plus the location.
func countLiveReferences(tx *gorm.DB, kind, network, dedupKey, location string) (int64, error) {
	model, err := artifactModel(kind)
	if err != nil {
		return 0, err
	}

	query := tx.Model(model).Where("network = ? AND location = ?", network, location)

	switch kind {
	case KindBeaconState, KindBeaconBlock, KindExecutionPayloadEnvelope:
		slot, identifier, ok := strings.Cut(dedupKey, "/")
		if !ok {
			return 0, fmt.Errorf("malformed dedup key for %s: %s", kind, dedupKey)
		}

		parsed, perr := strconv.ParseInt(slot, 10, 64)
		if perr != nil {
			return 0, fmt.Errorf("malformed slot in dedup key %s: %w", dedupKey, perr)
		}

		column := KeyBlockRoot
		if kind == KindBeaconState {
			column = KeyStateRoot
		}

		query = query.Where("slot = ? AND "+column+" = ?", parsed, identifier)
	case KindExecutionBlockTrace:
		// <block_number>/<block_hash>/<impl>/<version>/<flags>
		parts := strings.Split(dedupKey, "/")
		if len(parts) < 2 {
			return 0, fmt.Errorf("malformed dedup key for %s: %s", kind, dedupKey)
		}

		query = query.Where("block_hash = ?", parts[1])
	default:
		return 0, fmt.Errorf("kind %s does not reference blobs", kind)
	}

	var count int64

	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}
