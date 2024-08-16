package persistence

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

type BeaconBlock struct {
	gorm.Model
	ID   string `gorm:"primaryKey"`
	Node string `gorm:"index;index:idx_beacon_block_node_slot_blockroot_fetchedat_network_deletedat,priority:1"`
	// We have to use int64 here as SQLite doesn't support uint64. This sucks
	// but slot 9223372036854775808 is probably around the heat death
	// of the universe so we should be OK.
	Slot                 int64 `gorm:"index:idx_beacon_block_slot,where:deleted_at IS NULL;index;index:idx_beacon_block_node_slot_blockroot_fetchedat_network_deletedat,priority:2"`
	Epoch                int64
	BlockRoot            string    `gorm:"index;index:idx_beacon_block_node_slot_blockroot_fetchedat_network_deletedat,priority:3"`
	FetchedAt            time.Time `gorm:"index;index:idx_beacon_block_node_slot_blockroot_fetchedat_network_deletedat,priority:4;index:idx_beacon_block_fetchedat_deletedat,priority:1"`
	BeaconImplementation string
	NodeVersion          string         `gorm:"not null;default:''"`
	Location             string         `gorm:"not null;default:''"`
	Network              string         `gorm:"not null;default:'';index;index:idx_beacon_block_node_slot_blockroot_fetchedat_network_deletedat,priority:5"`
	DeletedAt            gorm.DeletedAt `gorm:"index;index:idx_beacon_block_node_slot_blockroot_fetchedat_network_deletedat,priority:6;index:idx_beacon_block_fetchedat_deletedat,priority:2"`
}

type BeaconBlockFilter struct {
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

func (f *BeaconBlockFilter) AddID(id string) {
	f.ID = &id
}

func (f *BeaconBlockFilter) AddNode(node string) {
	f.Node = &node
}

func (f *BeaconBlockFilter) AddBefore(before time.Time) {
	f.Before = &before
}

func (f *BeaconBlockFilter) AddAfter(after time.Time) {
	f.After = &after
}

func (f *BeaconBlockFilter) AddSlot(slot uint64) {
	f.Slot = &slot
}

func (f *BeaconBlockFilter) AddEpoch(epoch uint64) {
	f.Epoch = &epoch
}

func (f *BeaconBlockFilter) AddBlockRoot(blockRoot string) {
	f.BlockRoot = &blockRoot
}

func (f *BeaconBlockFilter) AddNodeVersion(nodeVersion string) {
	f.NodeVersion = &nodeVersion
}

func (f *BeaconBlockFilter) AddLocation(location string) {
	f.Location = &location
}

func (f *BeaconBlockFilter) AddNetwork(network string) {
	f.Network = &network
}

func (f *BeaconBlockFilter) AddBeaconImplementation(beaconImplementation string) {
	f.BeaconImplementation = &beaconImplementation
}

func (f *BeaconBlockFilter) Validate() error {
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

func (f *BeaconBlockFilter) ApplyToQuery(query *gorm.DB) (*gorm.DB, error) {
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

func (i *Indexer) InsertBeaconBlock(ctx context.Context, block *BeaconBlock) error {
	operation := OperationInsertBeaconBlock
	i.metrics.ObserveOperation(operation)

	result := i.db.WithContext(ctx).Create(block)
	if result.Error != nil {
		i.metrics.ObserveOperationError(operation)
	}

	return result.Error
}

func (i *Indexer) RemoveBeaconBlock(ctx context.Context, id string) error {
	operation := OperationDeleteBeaconBlock

	i.metrics.ObserveOperation(operation)

	result := i.db.WithContext(ctx).Unscoped().Where("id = ?", id).Delete(&BeaconBlock{})

	if result.Error != nil {
		i.metrics.ObserveOperationError(operation)
	}

	return result.Error
}

func (i *Indexer) CountBeaconBlock(ctx context.Context, filter *BeaconBlockFilter) (int64, error) {
	operation := OperationCountBeaconBlock

	i.metrics.ObserveOperation(operation)

	var count int64

	query := i.db.WithContext(ctx).Model(&BeaconBlock{})

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

func (i *Indexer) ListBeaconBlock(ctx context.Context, filter *BeaconBlockFilter, page *PaginationCursor) ([]*BeaconBlock, error) {
	operation := OperationListBeaconBlock

	i.metrics.ObserveOperation(operation)

	var BeaconBlocks []*BeaconBlock

	query := i.db.WithContext(ctx).Model(&BeaconBlock{})

	if page != nil {
		query = page.ApplyOffsetLimit(query)

		query = page.ApplyOrderBy(query)
	}

	query, err := filter.ApplyToQuery(query)
	if err != nil {
		i.metrics.ObserveOperationError(operation)

		return nil, err
	}

	result := query.Find(&BeaconBlocks)
	if result.Error != nil {
		i.metrics.ObserveOperationError(operation)

		return nil, result.Error
	}

	return BeaconBlocks, nil
}

type DistinctBeaconBlockValueResults struct {
	Node                 []string
	Slot                 []uint64
	Epoch                []uint64
	BlockRoot            []string
	NodeVersion          []string
	Location             []string
	Network              []string
	BeaconImplementation []string
}

func (i *Indexer) DistinctBeaconBlockValues(ctx context.Context, fields []string) (*DistinctBeaconBlockValueResults, error) {
	operation := OperationDistinctValues

	i.metrics.ObserveOperation(operation)

	results := &DistinctBeaconBlockValueResults{
		Node:                 make([]string, 0),
		Slot:                 make([]uint64, 0),
		Epoch:                make([]uint64, 0),
		BlockRoot:            make([]string, 0),
		NodeVersion:          make([]string, 0),
		Location:             make([]string, 0),
		Network:              make([]string, 0),
		BeaconImplementation: make([]string, 0),
	}
	query := i.db.WithContext(ctx).Model(&BeaconBlock{}).Select(fields).Group(strings.Join(fields, ", ")).Limit(1000)

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

func (i *Indexer) DeleteBeaconBlock(ctx context.Context, id string) error {
	operation := OperationDeleteBeaconBlock

	i.metrics.ObserveOperation(operation)

	query := i.db.WithContext(ctx)

	result := query.Unscoped().Where("id = ?", id).Delete(&BeaconBlock{})
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

func (i *Indexer) UpdateBeaconBlock(ctx context.Context, block *BeaconBlock) error {
	operation := OperationUpdateBeaconBlock

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
