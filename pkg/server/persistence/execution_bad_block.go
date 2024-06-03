package persistence

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

type ExecutionBadBlock struct {
	gorm.Model
	ID                      string    `gorm:"primaryKey"`
	Node                    string    `gorm:"index"`
	FetchedAt               time.Time `gorm:"index"`
	ExecutionImplementation string
	NodeVersion             string `gorm:"not null;default:''"`
	Location                string `gorm:"not null;default:''"`
	Network                 string `gorm:"not null;default:''"`
	BlockHash               string `gorm:"not null;default:''"`
	BlockNumber             sql.NullInt64
	BlockExtraData          sql.NullString
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

func (f *ExecutionBadBlockFilter) Validate() error {
	if f.ID == nil &&
		f.Node == nil &&
		f.Before == nil &&
		f.After == nil &&
		f.BlockHash == nil &&
		f.BlockNumber == nil &&
		f.ExecutionImplementation == nil &&
		f.NodeVersion == nil &&
		f.Location == nil &&
		f.Network == nil &&
		f.BlockExtraData == nil {
		return errors.New("no filter specified")
	}

	return nil
}

func (f *ExecutionBadBlockFilter) ApplyToQuery(query *gorm.DB) (*gorm.DB, error) {
	if f.ID != nil {
		query = query.Where("id = ?", f.ID)
	}

	if f.Node != nil {
		query = query.Where("node = ?", f.Node)
	}

	if f.Before != nil {
		query = query.Where("fetched_at <= ?", timestampFormatForDB(*f.Before))
	}

	if f.After != nil {
		query = query.Where("fetched_at >= ?", timestampFormatForDB(*f.After))
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

		query = page.ApplyOrderBy(query)
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

func (i *Indexer) DistinctExecutionBadBlockValues(ctx context.Context, fields []string) (*DistinctExecutionBadBlockValueResults, error) {
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
	query := i.db.WithContext(ctx).Model(&ExecutionBadBlock{}).Select(fields).Group(strings.Join(fields, ", ")).Limit(1000)

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
				case "block_extra_data":
					results.BlockExtraData = append(results.BlockExtraData, values[i].(string))
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
