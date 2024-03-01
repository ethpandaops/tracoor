package persistence

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

type BeaconBadBlock struct {
	gorm.Model
	ID   string `gorm:"primaryKey"`
	Node string `gorm:"index"`
	// We have to use int64 here as SQLite doesn't support uint64. This sucks
	// but slot 9223372036854775808 is probably around the heat death
	// of the universe so we should be OK.
	Slot                 int64 `gorm:"index:idx_slot,where:deleted_at IS NULL"`
	Epoch                int64
	BlockRoot            string
	FetchedAt            time.Time `gorm:"index"`
	BeaconImplementation string
	NodeVersion          string `gorm:"not null;default:''"`
	Location             string `gorm:"not null;default:''"`
	Network              string `gorm:"not null;default:''"`
}

type BeaconBadBlockFilter struct {
	ID                   *string
	Node                 *string
	Before               *time.Time
	After                *time.Time
	Slot                 *uint64
	Epoch                *uint64
	BlockRoot            *string
	NodeVersion          *string
	Location             *string
	Network              *string
	BeaconImplementation *string
}

func (f *BeaconBadBlockFilter) AddID(id string) {
	f.ID = &id
}

func (f *BeaconBadBlockFilter) AddNode(node string) {
	f.Node = &node
}

func (f *BeaconBadBlockFilter) AddBefore(before time.Time) {
	f.Before = &before
}

func (f *BeaconBadBlockFilter) AddAfter(after time.Time) {
	f.After = &after
}

func (f *BeaconBadBlockFilter) AddSlot(slot uint64) {
	f.Slot = &slot
}

func (f *BeaconBadBlockFilter) AddEpoch(epoch uint64) {
	f.Epoch = &epoch
}

func (f *BeaconBadBlockFilter) AddBlockRoot(blockRoot string) {
	f.BlockRoot = &blockRoot
}

func (f *BeaconBadBlockFilter) AddNodeVersion(nodeVersion string) {
	f.NodeVersion = &nodeVersion
}

func (f *BeaconBadBlockFilter) AddLocation(location string) {
	f.Location = &location
}

func (f *BeaconBadBlockFilter) AddNetwork(network string) {
	f.Network = &network
}

func (f *BeaconBadBlockFilter) AddBeaconImplementation(beaconImplementation string) {
	f.BeaconImplementation = &beaconImplementation
}

func (f *BeaconBadBlockFilter) Validate() error {
	if f.ID == nil &&
		f.Node == nil &&
		f.Before == nil &&
		f.After == nil &&
		f.Slot == nil &&
		f.Epoch == nil &&
		f.BlockRoot == nil &&
		f.NodeVersion == nil &&
		f.Location == nil &&
		f.BeaconImplementation == nil &&
		f.Network == nil {
		return errors.New("no filter specified")
	}

	return nil
}

func (f *BeaconBadBlockFilter) ApplyToQuery(query *gorm.DB) (*gorm.DB, error) {
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

	if f.Slot != nil {
		query = query.Where("slot = ?", f.Slot)
	}

	if f.Epoch != nil {
		query = query.Where("epoch = ?", f.Epoch)
	}

	if f.BlockRoot != nil {
		query = query.Where("block_root = ?", f.BlockRoot)
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

	if f.BeaconImplementation != nil {
		query = query.Where("beacon_implementation = ?", f.BeaconImplementation)
	}

	return query, nil
}

func (i *Indexer) InsertBeaconBadBlock(ctx context.Context, block *BeaconBadBlock) error {
	operation := OperationInsertBeaconBadBlock
	i.metrics.ObserveOperation(operation)

	result := i.db.WithContext(ctx).Create(block)
	if result.Error != nil {
		i.metrics.ObserveOperationError(operation)
	}

	return result.Error
}

func (i *Indexer) RemoveBeaconBadBlock(ctx context.Context, id string) error {
	operation := OperationDeleteBeaconBadBlock

	i.metrics.ObserveOperation(operation)

	result := i.db.WithContext(ctx).Unscoped().Where("id = ?", id).Delete(&BeaconBadBlock{})

	if result.Error != nil {
		i.metrics.ObserveOperationError(operation)
	}

	return result.Error
}

func (i *Indexer) CountBeaconBadBlock(ctx context.Context, filter *BeaconBadBlockFilter) (int64, error) {
	operation := OperationCountBeaconBadBlock

	i.metrics.ObserveOperation(operation)

	var count int64

	query := i.db.WithContext(ctx).Model(&BeaconBadBlock{})

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

func (i *Indexer) ListBeaconBadBlock(ctx context.Context, filter *BeaconBadBlockFilter, page *PaginationCursor) ([]*BeaconBadBlock, error) {
	operation := OperationListBeaconBadBlock

	i.metrics.ObserveOperation(operation)

	var BeaconBadBlocks []*BeaconBadBlock

	query := i.db.WithContext(ctx).Model(&BeaconBadBlock{})

	if page != nil {
		query = page.ApplyOffsetLimit(query)

		query = page.ApplyOrderBy(query)
	}

	query, err := filter.ApplyToQuery(query)
	if err != nil {
		i.metrics.ObserveOperationError(operation)

		return nil, err
	}

	result := query.Find(&BeaconBadBlocks)
	if result.Error != nil {
		i.metrics.ObserveOperationError(operation)

		return nil, result.Error
	}

	return BeaconBadBlocks, nil
}

type DistinctBeaconBadBlockValueResults struct {
	Node                 []string
	Slot                 []uint64
	Epoch                []uint64
	BlockRoot            []string
	NodeVersion          []string
	Location             []string
	Network              []string
	BeaconImplementation []string
}

func (i *Indexer) DistinctBeaconBadBlockValues(ctx context.Context, fields []string) (*DistinctBeaconBadBlockValueResults, error) {
	operation := OperationDistinctValues

	i.metrics.ObserveOperation(operation)

	results := &DistinctBeaconBadBlockValueResults{
		Node:                 make([]string, 0),
		Slot:                 make([]uint64, 0),
		Epoch:                make([]uint64, 0),
		BlockRoot:            make([]string, 0),
		NodeVersion:          make([]string, 0),
		Location:             make([]string, 0),
		Network:              make([]string, 0),
		BeaconImplementation: make([]string, 0),
	}
	query := i.db.WithContext(ctx).Model(&BeaconBadBlock{}).Select(fields).Group(strings.Join(fields, ", ")).Limit(1000)

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
				case "slot":
					results.Slot = append(results.Slot, uint64(values[i].(int64)))
				case "epoch":
					results.Epoch = append(results.Epoch, uint64(values[i].(int64)))
				case "block_root":
					results.BlockRoot = append(results.BlockRoot, values[i].(string))
				case "node_version":
					results.NodeVersion = append(results.NodeVersion, values[i].(string))
				case "location":
					results.Location = append(results.Location, values[i].(string))
				case "network":
					results.Network = append(results.Network, values[i].(string))
				case "beacon_implementation":
					results.BeaconImplementation = append(results.BeaconImplementation, values[i].(string))
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

func (i *Indexer) DeleteBeaconBadBlock(ctx context.Context, id string) error {
	operation := OperationDeleteBeaconBadBlock

	i.metrics.ObserveOperation(operation)

	query := i.db.WithContext(ctx)

	result := query.Unscoped().Where("id = ?", id).Delete(&BeaconBadBlock{})
	if result.Error != nil {
		i.metrics.ObserveOperationError(operation)

		return result.Error
	}

	if result.RowsAffected == 0 {
		i.metrics.ObserveOperationError(operation)

		return errors.New("beacon block not found")
	}

	return nil
}

func (i *Indexer) UpdateBeaconBadBlock(ctx context.Context, block *BeaconBadBlock) error {
	operation := OperationUpdateBeaconBadBlock

	i.metrics.ObserveOperation(operation)

	query := i.db.WithContext(ctx)

	result := query.Save(block)
	if result.Error != nil {
		i.metrics.ObserveOperationError(operation)

		return result.Error
	}

	if result.RowsAffected == 0 {
		i.metrics.ObserveOperationError(operation)

		return errors.New("beacon block not found")
	}

	if result.RowsAffected != 1 {
		i.metrics.ObserveOperationError(operation)

		return errors.New("beacon block update affected more than one row")
	}

	return nil
}
