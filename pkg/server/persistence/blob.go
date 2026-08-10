package persistence

import (
	"context"
	"errors"
	"strconv"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	// BlobStateReady marks a blob whose object is present and may be linked.
	BlobStateReady = "ready"
	// BlobStateDeleting marks a blob that has been tombstoned for collection. It must
	// never be linked, and its object may already be gone.
	BlobStateDeleting = "deleting"
)

// ErrBlobNotFound is returned when no linkable blob exists for a dedupe key. A tombstoned
// blob is reported as not found: it cannot be linked and the caller must re-upload.
var ErrBlobNotFound = errors.New("blob not found")

// Blob is the deduplicated payload. Keyed on what the lookup actually asks
// for: the dedup key is known before the fetch, the content hash is not.
type Blob struct {
	// The content-hash index carries kind and network ahead of the hash: retention resolves a
	// trace's payload by hash within a (kind, network), and neither planner can combine those
	// equalities with a hash-only index, so a narrower one degrades to a full scan.
	Kind     string `gorm:"primaryKey;size:32;index:ix_blobs_kind_network_content_hash,priority:1"`
	Network  string `gorm:"primaryKey;index:ix_blobs_kind_network_content_hash,priority:2"`
	DedupKey string `gorm:"primaryKey"`

	ContentHash     string `gorm:"not null;default:'';size:64;index:ix_blobs_kind_network_content_hash,priority:3"`
	Location        string `gorm:"not null;default:''"`
	ContentEncoding string `gorm:"not null;default:''"`
	RawSize         int64  `gorm:"not null;default:0"`
	CompressedSize  int64  `gorm:"not null;default:0"`
	RefCount        int64  `gorm:"not null;default:0"`

	// State is the tombstone marker and Generation the resurrect counter: a blob that
	// re-enters use while a delete is pending gets a new generation so the pending delete
	// cannot claim the fresh object.
	State      string `gorm:"not null;default:'ready';size:16;index:ix_blobs_state_created_at,priority:1"`
	Generation int64  `gorm:"not null;default:0"`

	CreatedAt time.Time `gorm:"not null;index:ix_blobs_state_created_at,priority:2"`
}

// BeforeSave keeps created_at in UTC, which is what the collector's grace-period cutoff is
// compared against.
func (b *Blob) BeforeSave(*gorm.DB) error {
	b.CreatedAt = utcBound(b.CreatedAt)

	return nil
}

// BlobCandidate is a blob that looks collectable: no references, old enough that no agent can
// still be mid-upload against it.
type BlobCandidate struct {
	Kind       string
	Network    string
	DedupKey   string
	Location   string
	Generation int64
}

// BlobTombstoneOutcome is what the tombstone attempt decided.
type BlobTombstoneOutcome int

const (
	// BlobTombstoneMarked means the blob is now in the deleting state and its object is the
	// caller's to remove.
	BlobTombstoneMarked BlobTombstoneOutcome = iota
	// BlobTombstoneReferenced means artifact rows still reference the blob. The counter had
	// drifted and has been repaired from the rows themselves.
	BlobTombstoneReferenced
	// BlobTombstoneRaced means the blob changed under the candidate scan — resurrected,
	// already tombstoned, or referenced again — and must be left alone this pass.
	BlobTombstoneRaced
)

// ListCollectableBlobs returns blobs whose reference count has reached zero and that are older
// than the grace period. The count is only a hint: every candidate is re-checked against the
// artifact tables before anything is deleted.
//
// The offset is how the caller steps over candidates it has already decided it cannot collect.
// The order is oldest first and a candidate that is not collected does not go away, so without
// it a single unevaluable row would occupy the head of every page for ever and nothing behind
// it would ever be reached.
func (i *Indexer) ListCollectableBlobs(ctx context.Context, createdBefore time.Time, limit, offset int) ([]*BlobCandidate, error) {
	var candidates []*BlobCandidate

	query := i.db.WithContext(ctx).Model(&Blob{}).
		Select("kind, network, dedup_key, location, generation").
		Where("state = ? AND ref_count <= 0 AND created_at < ?", BlobStateReady, utcBound(createdBefore)).
		Order("created_at ASC").
		Limit(limit)

	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Scan(&candidates).Error; err != nil {
		return nil, err
	}

	return candidates, nil
}

// TombstoneBlob moves a candidate to the deleting state after confirming against the artifact
// tables that nothing references it. Both halves run in one transaction so a create that lands
// between the check and the mark loses the race cleanly rather than losing its payload.
func (i *Indexer) TombstoneBlob(ctx context.Context, c *BlobCandidate) (BlobTombstoneOutcome, error) {
	outcome := BlobTombstoneRaced

	err := i.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		live, err := countLiveReferences(tx, c.Kind, c.Network, c.DedupKey, c.Location)
		if err != nil {
			return err
		}

		if live > 0 {
			outcome = BlobTombstoneReferenced

			return tx.Model(&Blob{}).
				Where("kind = ? AND network = ? AND dedup_key = ?", c.Kind, c.Network, c.DedupKey).
				UpdateColumn("ref_count", live).Error
		}

		result := tx.Model(&Blob{}).
			Where("kind = ? AND network = ? AND dedup_key = ? AND state = ? AND ref_count <= 0 AND generation = ?",
				c.Kind, c.Network, c.DedupKey, BlobStateReady, c.Generation).
			UpdateColumn("state", BlobStateDeleting)
		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected == 1 {
			outcome = BlobTombstoneMarked
		}

		return nil
	})
	if err != nil {
		return BlobTombstoneRaced, err
	}

	return outcome, nil
}

// DeleteTombstonedBlob removes the row once its object is gone. The generation predicate is
// what keeps a resurrected blob alive: a blob that came back has a new generation and a new
// object, and this delete simply does not match it.
func (i *Indexer) DeleteTombstonedBlob(ctx context.Context, c *BlobCandidate) (bool, error) {
	result := i.db.WithContext(ctx).
		Where("kind = ? AND network = ? AND dedup_key = ? AND state = ? AND generation = ?",
			c.Kind, c.Network, c.DedupKey, BlobStateDeleting, c.Generation).
		Delete(&Blob{})
	if result.Error != nil {
		return false, result.Error
	}

	return result.RowsAffected == 1, nil
}

// GetBlob returns the linkable blob for a dedupe key. Only a blob in the ready state is a
// hit; anything else yields ErrBlobNotFound.
func (i *Indexer) GetBlob(ctx context.Context, kind, network, dedupKey string) (*Blob, error) {
	operation := OperationGetBlob

	i.metrics.ObserveOperation(operation)

	var blob Blob

	result := i.db.WithContext(ctx).
		Where("kind = ? AND network = ? AND dedup_key = ? AND state = ?", kind, network, dedupKey, BlobStateReady).
		First(&blob)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, ErrBlobNotFound
		}

		i.metrics.ObserveOperationError(operation)

		return nil, result.Error
	}

	return &blob, nil
}

// InsertBlob inserts a blob and returns the row that won: the freshly inserted one, the one
// that was already there, or a tombstoned one brought back to life. A caller that lost the race
// compares the winner's hash and location against its own and acts on the difference.
func (i *Indexer) InsertBlob(ctx context.Context, blob *Blob) (*Blob, error) {
	operation := OperationInsertBlob

	i.metrics.ObserveOperation(operation)

	if blob.CreatedAt.IsZero() {
		blob.CreatedAt = time.Now().UTC()
	}

	if blob.State == "" {
		blob.State = BlobStateReady
	}

	result := i.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "kind"}, {Name: "network"}, {Name: "dedup_key"}},
		DoNothing: true,
	}).Create(blob)
	if result.Error != nil {
		i.metrics.ObserveOperationError(operation)

		return nil, result.Error
	}

	if result.RowsAffected == 1 {
		return blob, nil
	}

	var existing Blob

	fetch := i.db.WithContext(ctx).
		Where("kind = ? AND network = ? AND dedup_key = ?", blob.Kind, blob.Network, blob.DedupKey).
		First(&existing)
	if fetch.Error != nil {
		if errors.Is(fetch.Error, gorm.ErrRecordNotFound) {
			return nil, ErrBlobNotFound
		}

		i.metrics.ObserveOperationError(operation)

		return nil, fetch.Error
	}

	if existing.State != BlobStateDeleting {
		return &existing, nil
	}

	resurrected, err := i.resurrectBlob(ctx, &existing, blob)
	if err != nil {
		i.metrics.ObserveOperationError(operation)

		return nil, err
	}

	return resurrected, nil
}

// resurrectLocationSuffix separates a location from the generation that owns it.
const resurrectLocationSuffix = "-g"

// resurrectBlob brings a tombstoned blob back into service for a caller that has just
// re-uploaded the payload.
//
// Two things move. The generation advances, so the collection already in flight fails its
// final row delete and leaves the revived row alone. And when the caller's object sits at the
// exact path the collector captured, the revived blob is given a path of its own — the pending
// object delete cannot be recalled, so sharing a path with it would mean handing out a
// location that is about to be emptied. The winning row is returned either way, so a caller
// whose location was moved can see that it must republish there.
func (i *Indexer) resurrectBlob(ctx context.Context, tombstone, wanted *Blob) (*Blob, error) {
	revived := *tombstone
	revived.Generation = tombstone.Generation + 1
	revived.State = BlobStateReady
	revived.RefCount = 0
	revived.ContentHash = wanted.ContentHash
	revived.ContentEncoding = wanted.ContentEncoding
	revived.RawSize = wanted.RawSize
	revived.CompressedSize = wanted.CompressedSize
	revived.CreatedAt = time.Now().UTC()
	revived.Location = wanted.Location

	if revived.Location == tombstone.Location {
		revived.Location += resurrectLocationSuffix + strconv.FormatInt(revived.Generation, 10)
	}

	result := i.db.WithContext(ctx).Model(&Blob{}).
		Where("kind = ? AND network = ? AND dedup_key = ? AND state = ? AND generation = ?",
			tombstone.Kind, tombstone.Network, tombstone.DedupKey, BlobStateDeleting, tombstone.Generation).
		Updates(map[string]any{
			"content_hash":     revived.ContentHash,
			"location":         revived.Location,
			"content_encoding": revived.ContentEncoding,
			"raw_size":         revived.RawSize,
			"compressed_size":  revived.CompressedSize,
			"ref_count":        revived.RefCount,
			"state":            revived.State,
			"generation":       revived.Generation,
			"created_at":       revived.CreatedAt,
		})
	if result.Error != nil {
		return nil, result.Error
	}

	if result.RowsAffected == 1 {
		return &revived, nil
	}

	// Someone else revived it first. Whatever is in the table now is the winner.
	var current Blob

	if err := i.db.WithContext(ctx).
		Where("kind = ? AND network = ? AND dedup_key = ?", tombstone.Kind, tombstone.Network, tombstone.DedupKey).
		First(&current).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrBlobNotFound
		}

		return nil, err
	}

	return &current, nil
}
