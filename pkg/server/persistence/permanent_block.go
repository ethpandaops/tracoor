package persistence

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"
)

// PermanentBlock represents a permanently stored block in the database.
// This provides a mapping between slot, block_root, and network for
// blocks that have been copied to permanent storage.
type PermanentBlock struct {
	gorm.Model
	// We have to use int64 here as SQLite doesn't support uint64
	Slot      int64  `gorm:"index:idx_permanent_block_slot,where:deleted_at IS NULL;index:idx_permanent_block_slot_blockroot_network,where:deleted_at IS NULL,priority:1"`
	BlockRoot string `gorm:"index:idx_permanent_block_blockroot,where:deleted_at IS NULL;index:idx_permanent_block_slot_blockroot_network,where:deleted_at IS NULL,priority:2"`
	Network   string `gorm:"index:idx_permanent_block_network,where:deleted_at IS NULL;index:idx_permanent_block_slot_blockroot_network,where:deleted_at IS NULL,priority:3"`
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

// InsertPermanentBlock inserts a permanent block record
func (i *Indexer) InsertPermanentBlock(ctx context.Context, block *PermanentBlock) error {
	operation := OperationInsertPermanentBlock
	i.metrics.ObserveOperation(operation)

	query := i.db.WithContext(ctx)

	result := query.Create(block)
	if result.Error != nil {
		i.metrics.ObserveOperationError(operation)

		return result.Error
	}

	return nil
}

// ListPermanentBlock lists permanent blocks based on the filter
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
		if pagination.OrderBy != "" {
			query = query.Order(pagination.OrderBy)
		}

		if pagination.Limit > 0 {
			query = query.Limit(pagination.Limit)
		}

		if pagination.Offset > 0 {
			query = query.Offset(pagination.Offset)
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

// CountPermanentBlock counts permanent blocks based on the filter
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

// GetPermanentBlockByBlockRoot retrieves a permanent block by block root and network
func (i *Indexer) GetPermanentBlockByBlockRoot(ctx context.Context, blockRoot, network string) (*PermanentBlock, error) {
	operation := OperationGetPermanentBlock
	i.metrics.ObserveOperation(operation)

	query := i.db.WithContext(ctx).Model(&PermanentBlock{})

	var permanentBlock PermanentBlock

	result := query.Where("block_root = ? AND network = ?", blockRoot, network).First(&permanentBlock)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		i.metrics.ObserveOperationError(operation)

		return nil, result.Error
	}

	return &permanentBlock, nil
}

// DistinctPermanentBlockValues returns distinct values for permanent blocks
type DistinctPermanentBlockValues struct {
	Slot      []uint64
	BlockRoot []string
	Network   []string
}

// DistinctPermanentBlockValues gets distinct values for permanent blocks
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
	valueSets := make(map[string]map[interface{}]bool)

	for _, field := range fields {
		valueSets[field] = make(map[interface{}]bool)
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

	values := make([]interface{}, len(fields))

	for rows.Next() {
		values = make([]interface{}, len(fields))
		valuePtrs := make([]interface{}, len(fields))

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
				case "slot":
					//nolint:gosec // not worried about int64
					results.Slot = append(results.Slot, uint64(values[i].(int64)))
				case "block_root":
					results.BlockRoot = append(results.BlockRoot, values[i].(string))
				case "network":
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
