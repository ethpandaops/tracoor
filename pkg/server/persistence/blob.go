package persistence

import (
	"context"
	"errors"
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
	Kind     string `gorm:"primaryKey;size:32"`
	Network  string `gorm:"primaryKey"`
	DedupKey string `gorm:"primaryKey"`

	ContentHash     string `gorm:"not null;default:'';size:64;index:ix_blobs_content_hash"`
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

// InsertBlob inserts a blob and returns the row that won: the freshly inserted one, or the
// one that was already there. A caller that lost the race compares hashes against it.
func (i *Indexer) InsertBlob(ctx context.Context, blob *Blob) (*Blob, error) {
	operation := OperationInsertBlob

	i.metrics.ObserveOperation(operation)

	if blob.CreatedAt.IsZero() {
		blob.CreatedAt = time.Now()
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

	return &existing, nil
}
