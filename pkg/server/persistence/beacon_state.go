package persistence

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

type BeaconState struct {
	gorm.Model
	ID   string `gorm:"primaryKey"`
	Node string `gorm:"index;index:idx_beacon_state_node_slot_stateroot_network_fetchedat,where:deleted_at IS NULL,priority:1"`
	// We have to use int64 here as SQLite doesn't support uint64. This sucks
	// but slot 9223372036854775808 is probably around the heat death
	// of the universe so we should be OK.
	Slot                 int64 `gorm:"index:idx_beacon_state_slot,where:deleted_at IS NULL;index;index:idx_beacon_state_node_slot_stateroot_network_fetchedat,where:deleted_at IS NULL,priority:2"`
	Epoch                int64
	StateRoot            string    `gorm:"index;index:idx_beacon_state_node_slot_stateroot_network_fetchedat,where:deleted_at IS NULL,priority:3"`
	FetchedAt            time.Time `gorm:"index;index:idx_beacon_state_node_slot_stateroot_network_fetchedat,where:deleted_at IS NULL,priority:5;index:idx_beacon_state_fetchedat,where:deleted_at IS NULL;index:idx_beacon_state_fetchedat_network,where:deleted_at IS NULL,priority:1"`
	BeaconImplementation string
	NodeVersion          string `gorm:"not null;default:''"`
	Location             string `gorm:"not null;default:''"`
	Network              string `gorm:"not null;default:'';index;index:idx_beacon_state_node_slot_stateroot_network_fetchedat,where:deleted_at IS NULL,priority:4;index:idx_beacon_state_network,where:deleted_at IS NULL;index:idx_beacon_state_network,where:deleted_at IS NULL;index:idx_beacon_state_fetchedat_network,where:deleted_at IS NULL,priority:2"`
}

type BeaconStateFilter struct {
	ID                   *string
	Node                 *string
	Before               *time.Time
	After                *time.Time
	Slot                 *uint64
	Epoch                *uint64
	StateRoot            *string
	NodeVersion          *string
	Location             *string
	Network              *string
	BeaconImplementation *string
}

func (f *BeaconStateFilter) AddID(id string) {
	f.ID = &id
}

func (f *BeaconStateFilter) AddNode(node string) {
	f.Node = &node
}

func (f *BeaconStateFilter) AddBefore(before time.Time) {
	f.Before = &before
}

func (f *BeaconStateFilter) AddAfter(after time.Time) {
	f.After = &after
}

func (f *BeaconStateFilter) AddSlot(slot uint64) {
	f.Slot = &slot
}

func (f *BeaconStateFilter) AddEpoch(epoch uint64) {
	f.Epoch = &epoch
}

func (f *BeaconStateFilter) AddStateRoot(stateRoot string) {
	f.StateRoot = &stateRoot
}

func (f *BeaconStateFilter) AddNodeVersion(nodeVersion string) {
	f.NodeVersion = &nodeVersion
}

func (f *BeaconStateFilter) AddLocation(location string) {
	f.Location = &location
}

func (f *BeaconStateFilter) AddNetwork(network string) {
	f.Network = &network
}

func (f *BeaconStateFilter) AddBeaconImplementation(beaconImplementation string) {
	f.BeaconImplementation = &beaconImplementation
}

func (f *BeaconStateFilter) Validate() error {
	if f.ID == nil &&
		f.Node == nil &&
		f.Before == nil &&
		f.After == nil &&
		f.Slot == nil &&
		f.Epoch == nil &&
		f.StateRoot == nil &&
		f.NodeVersion == nil &&
		f.Location == nil &&
		f.BeaconImplementation == nil &&
		f.Network == nil {
		return errors.New("no filter specified")
	}

	return nil
}

func (f *BeaconStateFilter) ApplyToQuery(query *gorm.DB) (*gorm.DB, error) {
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

	if f.StateRoot != nil {
		query = query.Where("state_root = ?", f.StateRoot)
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

func (i *Indexer) InsertBeaconState(ctx context.Context, state *BeaconState) error {
	operation := OperationInsertBeaconState
	i.metrics.ObserveOperation(operation)

	result := i.db.WithContext(ctx).Create(state)
	if result.Error != nil {
		i.metrics.ObserveOperationError(operation)
	}

	return result.Error
}

func (i *Indexer) RemoveBeaconState(ctx context.Context, id string) error {
	operation := OperationDeleteBeaconState

	i.metrics.ObserveOperation(operation)

	result := i.db.WithContext(ctx).Unscoped().Where("id = ?", id).Delete(&BeaconState{})

	if result.Error != nil {
		i.metrics.ObserveOperationError(operation)
	}

	return result.Error
}

func (i *Indexer) CountBeaconState(ctx context.Context, filter *BeaconStateFilter) (int64, error) {
	operation := OperationCountBeaconState

	i.metrics.ObserveOperation(operation)

	var count int64

	query := i.db.WithContext(ctx).Model(&BeaconState{})

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

func (i *Indexer) ListBeaconState(ctx context.Context, filter *BeaconStateFilter, page *PaginationCursor) ([]*BeaconState, error) {
	operation := OperationListBeaconState

	i.metrics.ObserveOperation(operation)

	var BeaconStates []*BeaconState

	query := i.db.WithContext(ctx).Model(&BeaconState{})

	if page != nil {
		query = page.ApplyOffsetLimit(query)

		query = page.ApplyOrderBy(query)
	}

	query, err := filter.ApplyToQuery(query)
	if err != nil {
		i.metrics.ObserveOperationError(operation)

		return nil, err
	}

	result := query.Find(&BeaconStates)
	if result.Error != nil {
		i.metrics.ObserveOperationError(operation)

		return nil, result.Error
	}

	return BeaconStates, nil
}

type DistinctBeaconStateValueResults struct {
	Node                 []string
	Slot                 []uint64
	Epoch                []uint64
	StateRoot            []string
	NodeVersion          []string
	Location             []string
	Network              []string
	BeaconImplementation []string
}

func (i *Indexer) DistinctBeaconStateValues(ctx context.Context, fields []string) (*DistinctBeaconStateValueResults, error) {
	operation := OperationDistinctValues

	i.metrics.ObserveOperation(operation)

	results := &DistinctBeaconStateValueResults{
		Node:                 make([]string, 0),
		Slot:                 make([]uint64, 0),
		Epoch:                make([]uint64, 0),
		StateRoot:            make([]string, 0),
		NodeVersion:          make([]string, 0),
		Location:             make([]string, 0),
		Network:              make([]string, 0),
		BeaconImplementation: make([]string, 0),
	}
	query := i.db.WithContext(ctx).Model(&BeaconState{}).Select(fields).Group(strings.Join(fields, ", ")).Limit(1000)

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
				case "state_root":
					results.StateRoot = append(results.StateRoot, values[i].(string))
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

func (i *Indexer) DeleteBeaconState(ctx context.Context, id string) error {
	operation := OperationDeleteBeaconState

	i.metrics.ObserveOperation(operation)

	query := i.db.WithContext(ctx)

	result := query.Unscoped().Where("id = ?", id).Delete(&BeaconState{})
	if result.Error != nil {
		i.metrics.ObserveOperationError(operation)

		return result.Error
	}

	if result.RowsAffected == 0 {
		i.metrics.ObserveOperationError(operation)

		return errors.New("beacon state not found")
	}

	return nil
}

func (i *Indexer) UpdateBeaconState(ctx context.Context, state *BeaconState) error {
	operation := OperationUpdateBeaconState

	i.metrics.ObserveOperation(operation)

	query := i.db.WithContext(ctx)

	result := query.Save(state)
	if result.Error != nil {
		i.metrics.ObserveOperationError(operation)

		return result.Error
	}

	if result.RowsAffected == 0 {
		i.metrics.ObserveOperationError(operation)

		return errors.New("beacon state not found")
	}

	if result.RowsAffected != 1 {
		i.metrics.ObserveOperationError(operation)

		return errors.New("beacon state update affected more than one row")
	}

	return nil
}
