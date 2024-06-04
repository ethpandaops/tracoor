package persistence

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

type BeaconBadBlob struct {
	gorm.Model
	ID   string `gorm:"primaryKey"`
	Node string `gorm:"index"`
	// We have to use int64 here as SQLite doesn't support uint64. This sucks
	// but slot 9223372036854775808 is probably around the heat death
	// of the universe so we should be OK.
	Slot                 int64 `gorm:"where:deleted_at IS NULL"`
	Epoch                int64
	BlockRoot            string
	FetchedAt            time.Time `gorm:"index"`
	BeaconImplementation string
	NodeVersion          string `gorm:"not null;default:''"`
	Location             string `gorm:"not null;default:''"`
	Network              string `gorm:"not null;default:''"`
	Index                int64
}

type BeaconBadBlobFilter struct {
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
	Index                *uint64
}

func (f *BeaconBadBlobFilter) AddID(id string) {
	f.ID = &id
}

func (f *BeaconBadBlobFilter) AddNode(node string) {
	f.Node = &node
}

func (f *BeaconBadBlobFilter) AddBefore(before time.Time) {
	f.Before = &before
}

func (f *BeaconBadBlobFilter) AddAfter(after time.Time) {
	f.After = &after
}

func (f *BeaconBadBlobFilter) AddSlot(slot uint64) {
	f.Slot = &slot
}

func (f *BeaconBadBlobFilter) AddEpoch(epoch uint64) {
	f.Epoch = &epoch
}

func (f *BeaconBadBlobFilter) AddBlockRoot(blockRoot string) {
	f.BlockRoot = &blockRoot
}

func (f *BeaconBadBlobFilter) AddNodeVersion(nodeVersion string) {
	f.NodeVersion = &nodeVersion
}

func (f *BeaconBadBlobFilter) AddLocation(location string) {
	f.Location = &location
}

func (f *BeaconBadBlobFilter) AddNetwork(network string) {
	f.Network = &network
}

func (f *BeaconBadBlobFilter) AddBeaconImplementation(beaconImplementation string) {
	f.BeaconImplementation = &beaconImplementation
}

func (f *BeaconBadBlobFilter) AddIndex(index uint64) {
	f.Index = &index
}

func (f *BeaconBadBlobFilter) Validate() error {
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
		f.Network == nil &&
		f.Index == nil {
		return errors.New("no filter specified")
	}

	return nil
}

func (f *BeaconBadBlobFilter) ApplyToQuery(query *gorm.DB) (*gorm.DB, error) {
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

	if f.Index != nil {
		query = query.Where("`index` = ?", f.Index)
	}

	return query, nil
}

func (i *Indexer) InsertBeaconBadBlob(ctx context.Context, blob *BeaconBadBlob) error {
	operation := OperationInsertBeaconBadBlob
	i.metrics.ObserveOperation(operation)

	result := i.db.WithContext(ctx).Create(blob)
	if result.Error != nil {
		i.metrics.ObserveOperationError(operation)
	}

	return result.Error
}

func (i *Indexer) RemoveBeaconBadBlob(ctx context.Context, id string) error {
	operation := OperationDeleteBeaconBadBlob

	i.metrics.ObserveOperation(operation)

	result := i.db.WithContext(ctx).Unscoped().Where("id = ?", id).Delete(&BeaconBadBlob{})

	if result.Error != nil {
		i.metrics.ObserveOperationError(operation)
	}

	return result.Error
}

func (i *Indexer) CountBeaconBadBlob(ctx context.Context, filter *BeaconBadBlobFilter) (int64, error) {
	operation := OperationCountBeaconBadBlob

	i.metrics.ObserveOperation(operation)

	var count int64

	query := i.db.WithContext(ctx).Model(&BeaconBadBlob{})

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

func (i *Indexer) ListBeaconBadBlob(ctx context.Context, filter *BeaconBadBlobFilter, page *PaginationCursor) ([]*BeaconBadBlob, error) {
	operation := OperationListBeaconBadBlob

	i.metrics.ObserveOperation(operation)

	var BeaconBadBlobs []*BeaconBadBlob

	query := i.db.WithContext(ctx).Model(&BeaconBadBlob{})

	if page != nil {
		query = page.ApplyOffsetLimit(query)

		query = page.ApplyOrderBy(query)
	}

	query, err := filter.ApplyToQuery(query)
	if err != nil {
		i.metrics.ObserveOperationError(operation)

		return nil, err
	}

	result := query.Find(&BeaconBadBlobs)
	if result.Error != nil {
		i.metrics.ObserveOperationError(operation)

		return nil, result.Error
	}

	return BeaconBadBlobs, nil
}

type DistinctBeaconBadBlobValueResults struct {
	Node                 []string
	Slot                 []uint64
	Epoch                []uint64
	BlockRoot            []string
	NodeVersion          []string
	Location             []string
	Network              []string
	BeaconImplementation []string
	Index                []uint64
}

func (i *Indexer) DistinctBeaconBadBlobValues(ctx context.Context, fields []string) (*DistinctBeaconBadBlobValueResults, error) {
	operation := OperationDistinctValues

	i.metrics.ObserveOperation(operation)

	results := &DistinctBeaconBadBlobValueResults{
		Node:                 make([]string, 0),
		Slot:                 make([]uint64, 0),
		Epoch:                make([]uint64, 0),
		BlockRoot:            make([]string, 0),
		NodeVersion:          make([]string, 0),
		Location:             make([]string, 0),
		Network:              make([]string, 0),
		BeaconImplementation: make([]string, 0),
		Index:                make([]uint64, 0),
	}
	query := i.db.WithContext(ctx).Model(&BeaconBadBlob{}).Select(fields).Group(strings.Join(fields, ", ")).Limit(1000)

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
				case "index":
					results.Index = append(results.Index, uint64(values[i].(int64)))
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

func (i *Indexer) DeleteBeaconBadBlob(ctx context.Context, id string) error {
	operation := OperationDeleteBeaconBadBlob

	i.metrics.ObserveOperation(operation)

	query := i.db.WithContext(ctx)

	result := query.Unscoped().Where("id = ?", id).Delete(&BeaconBadBlob{})
	if result.Error != nil {
		i.metrics.ObserveOperationError(operation)

		return result.Error
	}

	if result.RowsAffected == 0 {
		i.metrics.ObserveOperationError(operation)

		return errors.New("beacon blob not found")
	}

	return nil
}

func (i *Indexer) UpdateBeaconBadBlob(ctx context.Context, blob *BeaconBadBlob) error {
	operation := OperationUpdateBeaconBadBlob

	i.metrics.ObserveOperation(operation)

	query := i.db.WithContext(ctx)

	result := query.Save(blob)
	if result.Error != nil {
		i.metrics.ObserveOperationError(operation)

		return result.Error
	}

	if result.RowsAffected == 0 {
		i.metrics.ObserveOperationError(operation)

		return errors.New("beacon blob not found")
	}

	if result.RowsAffected != 1 {
		i.metrics.ObserveOperationError(operation)

		return errors.New("beacon blob update affected more than one row")
	}

	return nil
}
