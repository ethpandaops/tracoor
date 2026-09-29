package persistence

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// legacyPermanentBlockIndex is the pre-kind unique index on (block_root, network),
// superseded by ux_permanent_blocks_kind_block_root_network and dropped at migration.
const legacyPermanentBlockIndex = "ux_permanent_blocks_block_root_network"

// ErrPermanentBlockNotFound is returned when no row records the artifact. It is an
// ordinary answer on the archive path - almost every artifact is new - so callers must
// be able to tell it apart from a database failure.
var ErrPermanentBlockNotFound = errors.New("permanent block not found")

// PermanentBlock represents a permanently stored artifact in the database.
// This provides a mapping between kind, slot, block_root, and network for
// artifacts that have been copied to permanent storage.
//
// It has no retention on purpose: it is the index of what was kept for ever, so a row that
// expired would leave a permanent object nothing points at.
//
// Kind is part of the identity, not decoration. Gloas splits a slot across two artifacts -
// the block and its execution payload envelope - which share a block root, so without kind
// the second one to arrive would collide with the first and never be archived.
//
// The only production lookup is by (kind, block_root, network), so that triple carries the
// one index, and unique makes it the integrity guarantee the get-before-insert flow in the
// permanent store otherwise only approximates.
type PermanentBlock struct {
	ID uint `gorm:"primaryKey"`
	// We have to use int64 here as SQLite doesn't support uint64
	Slot      int64  `gorm:"not null;default:0"`
	Kind      string `gorm:"not null;default:'beacon_block';uniqueIndex:ux_permanent_blocks_kind_block_root_network,priority:1"`
	BlockRoot string `gorm:"not null;default:'';uniqueIndex:ux_permanent_blocks_kind_block_root_network,priority:2"`
	Network   string `gorm:"not null;default:'';uniqueIndex:ux_permanent_blocks_kind_block_root_network,priority:3"`
}

type PermanentBlockFilter struct {
	Slot      *int64
	Kind      *string
	BlockRoot *string
	Network   *string
}

func (f *PermanentBlockFilter) AddKind(kind string) {
	f.Kind = &kind
}

func (f *PermanentBlockFilter) AddSlot(slot int64) {
	f.Slot = &slot
}

func (f *PermanentBlockFilter) AddBlockRoot(blockRoot string) {
	f.BlockRoot = &blockRoot
}

func (f *PermanentBlockFilter) AddNetwork(network string) {
	f.Network = &network
}

func (f *PermanentBlockFilter) ApplyToQuery(query *gorm.DB) (*gorm.DB, error) {
	if f.Slot != nil {
		query = query.Where("slot = ?", f.Slot)
	}

	if f.Kind != nil {
		query = query.Where("kind = ?", f.Kind)
	}

	if f.BlockRoot != nil {
		query = query.Where("block_root = ?", f.BlockRoot)
	}

	if f.Network != nil {
		query = query.Where("network = ?", f.Network)
	}

	return query, nil
}

// InsertPermanentBlock inserts a permanent block record. A record that already
// exists is left alone rather than erroring: two replicas racing past the
// distributed lock both believe they inserted, and both are right.
func (i *Indexer) InsertPermanentBlock(ctx context.Context, block *PermanentBlock) error {
	operation := OperationInsertPermanentBlock
	i.metrics.ObserveOperation(operation)

	query := i.db.WithContext(ctx)

	result := query.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "kind"}, {Name: "block_root"}, {Name: "network"}},
		DoNothing: true,
	}).Create(block)
	if result.Error != nil {
		i.metrics.ObserveOperationError(operation)

		return result.Error
	}

	return nil
}

// ListPermanentBlock lists permanent blocks based on the filter.
func (i *Indexer) ListPermanentBlock(ctx context.Context, filter *PermanentBlockFilter, pagination *PaginationCursor) ([]*PermanentBlock, error) {
	operation := OperationListPermanentBlock
	i.metrics.ObserveOperation(operation)

	query := i.db.WithContext(ctx).Model(&PermanentBlock{})

	query, err := filter.ApplyToQuery(query)
	if err != nil {
		i.metrics.ObserveOperationError(operation)

		return nil, err
	}

	if pagination != nil {
		query = pagination.ApplyOffsetLimit(query)

		// PermanentBlock has no fetched_at column, so the default ordering does not apply here.
		if pagination.OrderBy != "" {
			ordered, oerr := pagination.ApplyOrderBy(query)
			if oerr != nil {
				i.metrics.ObserveOperationError(operation)

				return nil, oerr
			}

			query = ordered
		}
	}

	var permanentBlocks []*PermanentBlock

	result := query.Find(&permanentBlocks)
	if result.Error != nil {
		i.metrics.ObserveOperationError(operation)

		return nil, result.Error
	}

	return permanentBlocks, nil
}

// CountPermanentBlock counts permanent blocks based on the filter.
func (i *Indexer) CountPermanentBlock(ctx context.Context, filter *PermanentBlockFilter) (int64, error) {
	operation := OperationCountPermanentBlock
	i.metrics.ObserveOperation(operation)

	query := i.db.WithContext(ctx).Model(&PermanentBlock{})

	query, err := filter.ApplyToQuery(query)
	if err != nil {
		i.metrics.ObserveOperationError(operation)

		return 0, err
	}

	var count int64

	result := query.Count(&count)
	if result.Error != nil {
		i.metrics.ObserveOperationError(operation)

		return 0, result.Error
	}

	return count, nil
}

// GetPermanentBlockByBlockRoot retrieves a permanent artifact by kind, block root and
// network. A row that is not there returns ErrPermanentBlockNotFound, which is the normal
// answer and not a database error.
func (i *Indexer) GetPermanentBlockByBlockRoot(ctx context.Context, kind, blockRoot, network string) (*PermanentBlock, error) {
	operation := OperationGetPermanentBlock
	i.metrics.ObserveOperation(operation)

	query := i.db.WithContext(ctx).Model(&PermanentBlock{})

	var permanentBlock PermanentBlock

	result := query.Where("kind = ? AND block_root = ? AND network = ?", kind, blockRoot, network).First(&permanentBlock)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, ErrPermanentBlockNotFound
		}

		i.metrics.ObserveOperationError(operation)

		return nil, result.Error
	}

	return &permanentBlock, nil
}

// DistinctPermanentBlockValues returns distinct values for permanent blocks.
type DistinctPermanentBlockValues struct {
	Slot      []uint64
	BlockRoot []string
	Network   []string
}

// DistinctPermanentBlockValues gets distinct values for permanent blocks.
//
//nolint:errcheck // casting fine here.
func (i *Indexer) DistinctPermanentBlockValues(ctx context.Context, fields []string) (*DistinctPermanentBlockValues, error) {
	operation := OperationDistinctValues
	i.metrics.ObserveOperation(operation)

	results := &DistinctPermanentBlockValues{
		Slot:      []uint64{},
		BlockRoot: []string{},
		Network:   []string{},
	}

	if len(fields) == 0 {
		return results, nil
	}

	// Create maps to track values we've already seen
	valueSets := make(map[string]map[any]bool)

	for _, field := range fields {
		valueSets[field] = make(map[any]bool)
	}

	// Create the SQL query with all fields
	query := i.db.WithContext(ctx).
		Model(&PermanentBlock{}).
		Distinct(strings.Join(fields, ", "))

	rows, err := query.Rows()
	if err != nil {
		i.metrics.ObserveOperationError(operation)

		return nil, err
	}
	defer rows.Close()

	values := make([]any, len(fields))

	for rows.Next() {
		valuePtrs := make([]any, len(fields))

		for i := range values {
			valuePtrs[i] = &values[i]
		}

		err := rows.Scan(valuePtrs...)
		if err != nil {
			i.metrics.ObserveOperationError(operation)

			return nil, err
		}

		for i, field := range fields {
			if !valueSets[field][values[i]] {
				switch field {
				case KeySlot:
					//nolint:gosec // not worried about int64
					results.Slot = append(results.Slot, uint64(values[i].(int64)))
				case KeyBlockRoot:
					results.BlockRoot = append(results.BlockRoot, values[i].(string))
				case KeyNetwork:
					results.Network = append(results.Network, values[i].(string))
				}

				valueSets[field][values[i]] = true
			}
		}
	}

	if err := rows.Err(); err != nil {
		i.metrics.ObserveOperationError(operation)

		return nil, err
	}

	return results, nil
}
