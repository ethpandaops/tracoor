package persistence

import (
	"context"
	"database/sql"
	"time"

	"gorm.io/gorm"
)

type ExecutionBadBlock struct {
	ID                      string    `gorm:"primaryKey"`
	Node                    string    `gorm:"not null;default:'';uniqueIndex:ux_execution_bad_blocks_dedupe,priority:3;index:ix_execution_bad_blocks_network_node_fetched_at,priority:2"`
	FetchedAt               time.Time `gorm:"not null;index:ix_execution_bad_blocks_fetched_at;index:ix_execution_bad_blocks_network_node_fetched_at,priority:3;index:ix_execution_bad_blocks_network_fetched_at,priority:2"`
	ExecutionImplementation string    `gorm:"not null;default:''"`
	NodeVersion             string    `gorm:"not null;default:''"`
	ContentEncoding         string    `gorm:"not null;default:''"`
	Location                string    `gorm:"not null;default:''"`
	ContentHash             string    `gorm:"not null;default:'';size:64"`
	// VerifiedAt is set when these bytes were read and hashed from this node.
	VerifiedAt *time.Time
	// ContentMatchedAt is set when the hash was compared against an existing
	// payload and matched. Bad blocks are never linked, so it stays null.
	ContentMatchedAt *time.Time
	Network          string `gorm:"not null;default:'';uniqueIndex:ux_execution_bad_blocks_dedupe,priority:1;index:ix_execution_bad_blocks_network_node_fetched_at,priority:1;index:ix_execution_bad_blocks_network_fetched_at,priority:1"`
	BlockHash        string `gorm:"not null;default:'';uniqueIndex:ux_execution_bad_blocks_dedupe,priority:2"`
	BlockNumber      sql.NullInt64
	BlockExtraData   sql.NullString
}

// BeforeSave keeps every stored timestamp in UTC. The drivers render a time.Time in the zone
// the value itself carries, so a row written by a process in another zone would neither order
// nor compare against the rest of the table.
func (a *ExecutionBadBlock) BeforeSave(*gorm.DB) error {
	a.FetchedAt = utcBound(a.FetchedAt)
	a.VerifiedAt = utcBoundPtr(a.VerifiedAt)
	a.ContentMatchedAt = utcBoundPtr(a.ContentMatchedAt)

	return nil
}

type ExecutionBadBlockFilter struct {
	ID                      *string
	Node                    *string
	Before                  *time.Time
	After                   *time.Time
	NodeVersion             *string
	Location                *string
	Network                 *string
	ExecutionImplementation *string
	BlockHash               *string
	BlockNumber             *int64
	BlockExtraData          *string
}

func (f *ExecutionBadBlockFilter) AddID(id string) {
	f.ID = &id
}

func (f *ExecutionBadBlockFilter) AddNode(node string) {
	f.Node = &node
}

func (f *ExecutionBadBlockFilter) AddBefore(before time.Time) {
	f.Before = &before
}

func (f *ExecutionBadBlockFilter) AddAfter(after time.Time) {
	f.After = &after
}

func (f *ExecutionBadBlockFilter) AddNodeVersion(nodeVersion string) {
	f.NodeVersion = &nodeVersion
}

func (f *ExecutionBadBlockFilter) AddLocation(location string) {
	f.Location = &location
}

func (f *ExecutionBadBlockFilter) AddNetwork(network string) {
	f.Network = &network
}

func (f *ExecutionBadBlockFilter) AddExecutionImplementation(impl string) {
	f.ExecutionImplementation = &impl
}

func (f *ExecutionBadBlockFilter) AddBlockHash(hash string) {
	f.BlockHash = &hash
}

func (f *ExecutionBadBlockFilter) AddBlockNumber(number int64) {
	f.BlockNumber = &number
}

func (f *ExecutionBadBlockFilter) AddBlockExtraData(data string) {
	f.BlockExtraData = &data
}

func (f *ExecutionBadBlockFilter) ApplyToQuery(query *gorm.DB) (*gorm.DB, error) {
	if f.ID != nil {
		query = query.Where("id = ?", f.ID)
	}

	if f.Node != nil {
		query = query.Where("node = ?", f.Node)
	}

	if f.Before != nil {
		query = query.Where("fetched_at <= ?", utcBound(*f.Before))
	}

	if f.After != nil {
		query = query.Where("fetched_at >= ?", utcBound(*f.After))
	}

	if f.BlockHash != nil {
		query = query.Where("block_hash = ?", f.BlockHash)
	}

	if f.BlockNumber != nil {
		query = query.Where("block_number = ?", f.BlockNumber)
	}

	if f.BlockExtraData != nil {
		query = query.Where("block_extra_data = ?", f.BlockExtraData)
	}

	if f.NodeVersion != nil {
		query = query.Where("node_version = ?", f.NodeVersion)
	}

	if f.Location != nil {
		query = query.Where("location = ?", f.Location)
	}

	if f.Network != nil {
		query = query.Where("network = ?", f.Network)
	}

	if f.ExecutionImplementation != nil {
		query = query.Where("execution_implementation = ?", f.ExecutionImplementation)
	}

	return query, nil
}

func (i *Indexer) InsertExecutionBadBlock(ctx context.Context, trace *ExecutionBadBlock) error {
	operation := OperationInsertExecutionBadBlock
	i.metrics.ObserveOperation(operation)

	result := i.db.WithContext(ctx).Create(trace)
	if result.Error != nil {
		i.metrics.ObserveOperationError(operation)
	}

	return result.Error
}

func (i *Indexer) DeleteExecutionBadBlock(ctx context.Context, id string) error {
	operation := OperationDeleteExecutionBadBlock

	i.metrics.ObserveOperation(operation)

	result := i.db.WithContext(ctx).Unscoped().Where("id = ?", id).Delete(&ExecutionBadBlock{})

	if result.Error != nil {
		i.metrics.ObserveOperationError(operation)
	}

	return result.Error
}

func (i *Indexer) CountExecutionBadBlock(ctx context.Context, filter *ExecutionBadBlockFilter) (int64, error) {
	operation := OperationCountExecutionBadBlock

	i.metrics.ObserveOperation(operation)

	var count int64

	query := i.db.WithContext(ctx).Model(&ExecutionBadBlock{})

	query, err := filter.ApplyToQuery(query)
	if err != nil {
		i.metrics.ObserveOperationError(operation)

		return 0, err
	}

	result := query.Count(&count)
	if result.Error != nil {
		i.metrics.ObserveOperationError(operation)

		return 0, result.Error
	}

	return count, nil
}

func (i *Indexer) ListExecutionBadBlock(ctx context.Context, filter *ExecutionBadBlockFilter, page *PaginationCursor) ([]*ExecutionBadBlock, error) {
	operation := OperationListExecutionBadBlock

	i.metrics.ObserveOperation(operation)

	var ExecutionBadBlocks []*ExecutionBadBlock

	query := i.db.WithContext(ctx).Model(&ExecutionBadBlock{})

	if page != nil {
		query = page.ApplyOffsetLimit(query)

		ordered, err := page.ApplyOrderBy(query)
		if err != nil {
			i.metrics.ObserveOperationError(operation)

			return nil, err
		}

		query = ordered
	}

	query, err := filter.ApplyToQuery(query)
	if err != nil {
		i.metrics.ObserveOperationError(operation)

		return nil, err
	}

	result := query.Find(&ExecutionBadBlocks)
	if result.Error != nil {
		i.metrics.ObserveOperationError(operation)

		return nil, result.Error
	}

	return ExecutionBadBlocks, nil
}

type DistinctExecutionBadBlockValueResults struct {
	Node                    []string
	BlockHash               []string
	BlockNumber             []int64
	Location                []string
	Network                 []string
	ExecutionImplementation []string
	NodeVersion             []string
	BlockExtraData          []string
}

// executionBadBlockDistinct declares how each requested field's distinct values are
// resolved. Loose-scannable fields lead an index right after network: node via
// ix_execution_bad_blocks_network_node_fetched_at(network, node, fetched_at), block_hash
// via ux_execution_bad_blocks_dedupe(network, block_hash, node), and network leads both.
var executionBadBlockDistinct = distinctTable{
	name: "execution_bad_blocks",
	fields: map[string]distinctStrategy{
		KeyNode:                    distinctLooseScan,
		KeyBlockHash:               distinctLooseScan,
		KeyNetwork:                 distinctLooseScan,
		KeyBlockNumber:             distinctFullScan,
		KeyLocation:                distinctFullScan,
		KeyExecutionImplementation: distinctFullScan,
		KeyNodeVersion:             distinctFullScan,
		KeyBlockExtraData:          distinctFullScan,
	},
}

func (i *Indexer) DistinctExecutionBadBlockValues(ctx context.Context, fields []string, network string) (*DistinctExecutionBadBlockValueResults, error) {
	operation := OperationDistinctValues

	i.metrics.ObserveOperation(operation)

	results := &DistinctExecutionBadBlockValueResults{
		Node:                    make([]string, 0),
		BlockHash:               make([]string, 0),
		BlockNumber:             make([]int64, 0),
		Location:                make([]string, 0),
		Network:                 make([]string, 0),
		ExecutionImplementation: make([]string, 0),
		NodeVersion:             make([]string, 0),
		BlockExtraData:          make([]string, 0),
	}

	seen := make(map[string]bool, len(fields))

	for _, field := range fields {
		if seen[field] {
			continue
		}

		seen[field] = true

		values, err := i.distinctFieldValues(ctx, executionBadBlockDistinct, field, network)
		if err != nil {
			i.metrics.ObserveOperationError(operation)

			return nil, err
		}

		switch field {
		case KeyNode:
			results.Node = distinctStrings(values)
		case KeyBlockHash:
			results.BlockHash = distinctStrings(values)
		case KeyBlockNumber:
			results.BlockNumber = distinctInt64s(values)
		case KeyLocation:
			results.Location = distinctStrings(values)
		case KeyNetwork:
			results.Network = distinctStrings(values)
		case KeyExecutionImplementation:
			results.ExecutionImplementation = distinctStrings(values)
		case KeyNodeVersion:
			results.NodeVersion = distinctStrings(values)
		case KeyBlockExtraData:
			results.BlockExtraData = distinctStrings(values)
		}
	}

	return results, nil
}
