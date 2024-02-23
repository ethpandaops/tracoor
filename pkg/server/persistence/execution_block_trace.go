package persistence

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

type ExecutionBlockTrace struct {
	gorm.Model
	ID                      string    `gorm:"primaryKey"`
	Node                    string    `gorm:"index"`
	FetchedAt               time.Time `gorm:"index"`
	ExecutionImplementation string
	NodeVersion             string `gorm:"not null;default:''"`
	Location                string `gorm:"not null;default:''"`
	Network                 string `gorm:"not null;default:''"`
	BlockHash               string `gorm:"not null;default:''"`
	BlockNumber             int64
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

func (f *ExecutionBlockTraceFilter) Validate() error {
	if f.ID == nil &&
		f.Node == nil &&
		f.Before == nil &&
		f.After == nil &&
		f.BlockHash == nil &&
		f.BlockNumber == nil &&
		f.ExecutionImplementation == nil &&
		f.NodeVersion == nil &&
		f.Location == nil &&
		f.Network == nil {
		return errors.New("no filter specified")
	}

	return nil
}

func (f *ExecutionBlockTraceFilter) ApplyToQuery(query *gorm.DB) (*gorm.DB, error) {
	if f.ID != nil {
		query = query.Where("id = ?", f.ID)
	}

	if f.Node != nil {
		query = query.Where("node = ?", f.Node)
	}

	if f.Before != nil {
		query = query.Where("fetched_at <= ?", f.Before)
	}

	if f.After != nil {
		query = query.Where("fetched_at >= ?", f.After)
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

		query = page.ApplyOrderBy(query)
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

func (i *Indexer) DistinctExecutionBlockTraceValues(ctx context.Context, fields []string) (*DistinctExecutionBlockTraceValueResults, error) {
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
	query := i.db.WithContext(ctx).Model(&ExecutionBlockTrace{}).Select(fields).Group(strings.Join(fields, ", ")).Limit(1000)

	rows, err := query.Rows()
	if err != nil {
		i.metrics.ObserveOperationError(operation)

		return nil, err
	}
	defer rows.Close()

	valueSets := make(map[string]map[interface{}]bool)
	for _, field := range fields {
		valueSets[field] = make(map[interface{}]bool)
	}

	var values []interface{}
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
				case "node":
					results.Node = append(results.Node, values[i].(string))
				case "block_hash":
					results.BlockHash = append(results.BlockHash, values[i].(string))
				case "block_number":
					results.BlockNumber = append(results.BlockNumber, values[i].(int64))
				case "location":
					results.Location = append(results.Location, values[i].(string))
				case "network":
					results.Network = append(results.Network, values[i].(string))
				case "execution_implementation":
					results.ExecutionImplementation = append(results.ExecutionImplementation, values[i].(string))
				case "node_version":
					results.NodeVersion = append(results.NodeVersion, values[i].(string))
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
