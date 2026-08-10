package persistence

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// PayloadDivergence records that a node served bytes whose hash did not match the
// canonical blob for the same dedup key. The record is the finding; the divergent
// payload may or may not have been re-fetched.
type PayloadDivergence struct {
	ID string `gorm:"primaryKey"`

	ObservedAt time.Time `gorm:"not null;index:ix_payload_divergences_observed_at;index:ix_payload_divergences_network_kind_observed_at,priority:3"`
	Network    string    `gorm:"not null;default:'';index:ix_payload_divergences_network_kind_observed_at,priority:1;index:ix_payload_divergences_network_kind_dedup_key,priority:1"`
	Kind       string    `gorm:"not null;default:'';size:32;index:ix_payload_divergences_network_kind_observed_at,priority:2;index:ix_payload_divergences_network_kind_dedup_key,priority:2"`
	Node       string    `gorm:"not null;default:''"`

	DedupKey string `gorm:"not null;default:'';index:ix_payload_divergences_network_kind_dedup_key,priority:3"`

	ExpectedHash string `gorm:"not null;default:'';size:64"` // the recorded blob's content_hash
	ActualHash   string `gorm:"not null;default:'';size:64"` // what this node served

	// Human/UI-facing identity. Slot is 0 for the execution kinds.
	Slot       int64  `gorm:"not null;default:0"`
	Identifier string `gorm:"not null;default:''"` // state_root / block_root / block_hash

	// 1 = first observation, 2 = the single re-fetch.
	Attempt int32 `gorm:"not null;default:1"`
	// "alarm" for states/blocks/envelopes, "notice" for traces.
	Severity string `gorm:"not null;default:'';size:16"`
	// Location of the divergent copy, empty when the re-fetch failed or the window closed.
	Location string `gorm:"not null;default:''"`
}

type PayloadDivergenceFilter struct {
	Network  *string
	Kind     *string
	DedupKey *string
	Before   *time.Time
	After    *time.Time
}

func (f *PayloadDivergenceFilter) AddNetwork(network string) {
	f.Network = &network
}

func (f *PayloadDivergenceFilter) AddKind(kind string) {
	f.Kind = &kind
}

func (f *PayloadDivergenceFilter) AddDedupKey(dedupKey string) {
	f.DedupKey = &dedupKey
}

func (f *PayloadDivergenceFilter) AddBefore(before time.Time) {
	f.Before = &before
}

func (f *PayloadDivergenceFilter) AddAfter(after time.Time) {
	f.After = &after
}

func (f *PayloadDivergenceFilter) ApplyToQuery(query *gorm.DB) (*gorm.DB, error) {
	if f.Network != nil {
		query = query.Where("network = ?", f.Network)
	}

	if f.Kind != nil {
		query = query.Where("kind = ?", f.Kind)
	}

	if f.DedupKey != nil {
		query = query.Where("dedup_key = ?", f.DedupKey)
	}

	// Bound as time values rather than formatted strings: the drivers store timestamps in
	// their own textual shape, so a hand-formatted literal does not compare reliably.
	if f.Before != nil {
		query = query.Where("observed_at <= ?", *f.Before)
	}

	if f.After != nil {
		query = query.Where("observed_at >= ?", *f.After)
	}

	return query, nil
}

var defaultPayloadDivergenceOrderBy = clause.OrderByColumn{
	Column: clause.Column{Name: "observed_at"},
	Desc:   true,
}

// InsertPayloadDivergence records a divergence observation.
func (i *Indexer) InsertPayloadDivergence(ctx context.Context, divergence *PayloadDivergence) error {
	operation := OperationInsertPayloadDivergence

	i.metrics.ObserveOperation(operation)

	result := i.db.WithContext(ctx).Create(divergence)
	if result.Error != nil {
		i.metrics.ObserveOperationError(operation)
	}

	return result.Error
}

// ListPayloadDivergence lists divergences newest first unless the cursor orders otherwise.
func (i *Indexer) ListPayloadDivergence(ctx context.Context, filter *PayloadDivergenceFilter, page *PaginationCursor) ([]*PayloadDivergence, error) {
	operation := OperationListPayloadDivergence

	i.metrics.ObserveOperation(operation)

	query := i.db.WithContext(ctx).Model(&PayloadDivergence{})

	if page != nil {
		query = page.ApplyOffsetLimit(query)

		// PayloadDivergence has no fetched_at column, so it carries its own default order.
		if page.OrderBy == "" {
			query = query.Order(defaultPayloadDivergenceOrderBy)
		} else {
			ordered, err := page.ApplyOrderBy(query)
			if err != nil {
				i.metrics.ObserveOperationError(operation)

				return nil, err
			}

			query = ordered
		}
	}

	query, err := filter.ApplyToQuery(query)
	if err != nil {
		i.metrics.ObserveOperationError(operation)

		return nil, err
	}

	var divergences []*PayloadDivergence

	result := query.Find(&divergences)
	if result.Error != nil {
		i.metrics.ObserveOperationError(operation)

		return nil, result.Error
	}

	return divergences, nil
}
