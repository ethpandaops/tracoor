package persistence

import (
	"context"
	"time"

	"gorm.io/gorm"
)

type ExecutionBlockTrace struct {
	ID                      string    `gorm:"primaryKey"`
	Node                    string    `gorm:"not null;default:'';uniqueIndex:ux_execution_block_traces_dedupe,priority:3;index:ix_execution_block_traces_network_node_fetched_at,priority:2"`
	FetchedAt               time.Time `gorm:"not null;index:ix_execution_block_traces_fetched_at;index:ix_execution_block_traces_network_node_fetched_at,priority:3;index:ix_execution_block_traces_network_fetched_at,priority:2"`
	ExecutionImplementation string    `gorm:"not null;default:''"`
	NodeVersion             string    `gorm:"not null;default:''"`
	ContentEncoding         string    `gorm:"not null;default:''"`
	Location                string    `gorm:"not null;default:''"`
	ContentHash             string    `gorm:"not null;default:'';size:64"`
	// VerifiedAt is set when these bytes were read and hashed from this node.
	VerifiedAt *time.Time
	// ContentMatchedAt is set when the hash was compared against an existing
	// payload and matched.
	ContentMatchedAt *time.Time
	Network          string `gorm:"not null;default:'';uniqueIndex:ux_execution_block_traces_dedupe,priority:1;index:ix_execution_block_traces_network_node_fetched_at,priority:1;index:ix_execution_block_traces_network_fetched_at,priority:1"`
	BlockHash        string `gorm:"not null;default:'';uniqueIndex:ux_execution_block_traces_dedupe,priority:2"`
	BlockNumber      int64  `gorm:"not null;default:0"`
}

// BeforeSave keeps every stored timestamp in UTC. The drivers render a time.Time in the zone
// the value itself carries, so a row written by a process in another zone would neither order
// nor compare against the rest of the table.
func (a *ExecutionBlockTrace) BeforeSave(*gorm.DB) error {
	a.FetchedAt = utcBound(a.FetchedAt)
	a.VerifiedAt = utcBoundPtr(a.VerifiedAt)
	a.ContentMatchedAt = utcBoundPtr(a.ContentMatchedAt)

	return nil
}

type ExecutionBlockTraceFilter struct {
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
}

func (f *ExecutionBlockTraceFilter) AddID(id string) {
	f.ID = &id
}

func (f *ExecutionBlockTraceFilter) AddNode(node string) {
	f.Node = &node
}

func (f *ExecutionBlockTraceFilter) AddBefore(before time.Time) {
	f.Before = &before
}

func (f *ExecutionBlockTraceFilter) AddAfter(after time.Time) {
	f.After = &after
}

func (f *ExecutionBlockTraceFilter) AddNodeVersion(nodeVersion string) {
	f.NodeVersion = &nodeVersion
}

func (f *ExecutionBlockTraceFilter) AddLocation(location string) {
	f.Location = &location
}

func (f *ExecutionBlockTraceFilter) AddNetwork(network string) {
	f.Network = &network
}

func (f *ExecutionBlockTraceFilter) AddExecutionImplementation(impl string) {
	f.ExecutionImplementation = &impl
}

func (f *ExecutionBlockTraceFilter) AddBlockHash(hash string) {
	f.BlockHash = &hash
}

func (f *ExecutionBlockTraceFilter) AddBlockNumber(number int64) {
	f.BlockNumber = &number
}

func (f *ExecutionBlockTraceFilter) ApplyToQuery(query *gorm.DB) (*gorm.DB, error) {
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

func (i *Indexer) InsertExecutionBlockTrace(ctx context.Context, trace *ExecutionBlockTrace) error {
	operation := OperationInsertExecutionBlockTrace
	i.metrics.ObserveOperation(operation)

	result := i.db.WithContext(ctx).Create(trace)
	if result.Error != nil {
		i.metrics.ObserveOperationError(operation)
	}

	return result.Error
}

func (i *Indexer) DeleteExecutionBlockTrace(ctx context.Context, id string) error {
	operation := OperationDeleteExecutionBlockTrace

	i.metrics.ObserveOperation(operation)

	result := i.db.WithContext(ctx).Unscoped().Where("id = ?", id).Delete(&ExecutionBlockTrace{})

	if result.Error != nil {
		i.metrics.ObserveOperationError(operation)
	}

	return result.Error
}

func (i *Indexer) CountExecutionBlockTrace(ctx context.Context, filter *ExecutionBlockTraceFilter) (int64, error) {
	operation := OperationCountExecutionBlockTrace

	i.metrics.ObserveOperation(operation)

	var count int64

	query := i.db.WithContext(ctx).Model(&ExecutionBlockTrace{})

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

func (i *Indexer) ListExecutionBlockTrace(ctx context.Context, filter *ExecutionBlockTraceFilter, page *PaginationCursor) ([]*ExecutionBlockTrace, error) {
	operation := OperationListExecutionBlockTrace

	i.metrics.ObserveOperation(operation)

	var ExecutionBlockTraces []*ExecutionBlockTrace

	query := i.db.WithContext(ctx).Model(&ExecutionBlockTrace{})

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

	result := query.Find(&ExecutionBlockTraces)
	if result.Error != nil {
		i.metrics.ObserveOperationError(operation)

		return nil, result.Error
	}

	return ExecutionBlockTraces, nil
}

type DistinctExecutionBlockTraceValueResults struct {
	Node                    []string
	BlockHash               []string
	BlockNumber             []int64
	Location                []string
	Network                 []string
	ExecutionImplementation []string
	NodeVersion             []string
}

// executionBlockTraceDistinct declares how each requested field's distinct values are
// resolved. Loose-scannable fields lead an index right after network: node via
// ix_execution_block_traces_network_node_fetched_at(network, node, fetched_at), block_hash
// via ux_execution_block_traces_dedupe(network, block_hash, node), and network leads both.
var executionBlockTraceDistinct = distinctTable{
	name: "execution_block_traces",
	fields: map[string]distinctStrategy{
		KeyNode:                    distinctLooseScan,
		KeyBlockHash:               distinctLooseScan,
		KeyNetwork:                 distinctLooseScan,
		KeyBlockNumber:             distinctFullScan,
		KeyLocation:                distinctFullScan,
		KeyExecutionImplementation: distinctFullScan,
		KeyNodeVersion:             distinctFullScan,
	},
}

func (i *Indexer) DistinctExecutionBlockTraceValues(ctx context.Context, fields []string, network string) (*DistinctExecutionBlockTraceValueResults, error) {
	operation := OperationDistinctValues

	i.metrics.ObserveOperation(operation)

	results := &DistinctExecutionBlockTraceValueResults{
		Node:                    make([]string, 0),
		BlockHash:               make([]string, 0),
		BlockNumber:             make([]int64, 0),
		Location:                make([]string, 0),
		Network:                 make([]string, 0),
		ExecutionImplementation: make([]string, 0),
		NodeVersion:             make([]string, 0),
	}

	seen := make(map[string]bool, len(fields))

	for _, field := range fields {
		if seen[field] {
			continue
		}

		seen[field] = true

		values, err := i.distinctFieldValues(ctx, executionBlockTraceDistinct, field, network)
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
		}
	}

	return results, nil
}
