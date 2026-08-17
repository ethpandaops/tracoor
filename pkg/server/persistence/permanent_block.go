package persistence

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// PermanentBlock represents a permanently stored block in the database.
// This provides a mapping between slot, block_root, and network for
// blocks that have been copied to permanent storage.
//
// It has no retention on purpose: it is the index of what was kept for ever, so a row that
// expired would leave a permanent object nothing points at.
//
// The only production lookup is by (block_root, network), so that pair carries the one
// index, and unique makes it the integrity guarantee the get-before-insert flow in the
// permanent store otherwise only approximates.
type PermanentBlock struct {
	ID uint `gorm:"primaryKey"`
	// We have to use int64 here as SQLite doesn't support uint64
	Slot      int64  `gorm:"not null;default:0"`
	BlockRoot string `gorm:"not null;default:'';uniqueIndex:ux_permanent_blocks_block_root_network,priority:1"`
	Network   string `gorm:"not null;default:'';uniqueIndex:ux_permanent_blocks_block_root_network,priority:2"`
}

type PermanentBlockFilter struct {
	Slot      *int64
	BlockRoot *string
	Network   *string
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
		Columns:   []clause.Column{{Name: "block_root"}, {Name: "network"}},
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

// GetPermanentBlockByBlockRoot retrieves a permanent block by block root and network.
func (i *Indexer) GetPermanentBlockByBlockRoot(ctx context.Context, blockRoot, network string) (*PermanentBlock, error) {
	operation := OperationGetPermanentBlock
	i.metrics.ObserveOperation(operation)

	query := i.db.WithContext(ctx).Model(&PermanentBlock{})

	var permanentBlock PermanentBlock

	result := query.Where("block_root = ? AND network = ?", blockRoot, network).First(&permanentBlock)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("permanent block not found")
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
