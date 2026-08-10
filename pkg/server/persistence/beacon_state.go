package persistence

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

type BeaconState struct {
	ID   string `gorm:"primaryKey"`
	Node string `gorm:"not null;default:'';uniqueIndex:ux_beacon_states_dedupe,priority:4;index:ix_beacon_states_network_node_fetched_at,priority:2"`
	// We have to use int64 here as SQLite doesn't support uint64. This sucks
	// but slot 9223372036854775808 is probably around the heat death
	// of the universe so we should be OK.
	Slot                 int64     `gorm:"not null;default:0;uniqueIndex:ux_beacon_states_dedupe,priority:2"`
	Epoch                int64     `gorm:"not null;default:0"`
	StateRoot            string    `gorm:"not null;default:'';uniqueIndex:ux_beacon_states_dedupe,priority:3"`
	FetchedAt            time.Time `gorm:"not null;index:ix_beacon_states_fetched_at;index:ix_beacon_states_network_node_fetched_at,priority:3;index:ix_beacon_states_network_fetched_at,priority:2"`
	BeaconImplementation string    `gorm:"not null;default:''"`
	NodeVersion          string    `gorm:"not null;default:''"`
	ContentEncoding      string    `gorm:"not null;default:''"`
	Location             string    `gorm:"not null;default:''"`
	ContentHash          string    `gorm:"not null;default:'';size:64"`
	// VerifiedAt is set when these bytes were read and hashed from this node.
	VerifiedAt *time.Time
	// ContentMatchedAt is set when the hash was compared against an existing
	// payload and matched.
	ContentMatchedAt *time.Time
	Network          string `gorm:"not null;default:'';uniqueIndex:ux_beacon_states_dedupe,priority:1;index:ix_beacon_states_network_node_fetched_at,priority:1;index:ix_beacon_states_network_fetched_at,priority:1"`
}

// BeforeSave keeps every stored timestamp in UTC. The drivers render a time.Time in the zone
// the value itself carries, so a row written by a process in another zone would neither order
// nor compare against the rest of the table.
func (a *BeaconState) BeforeSave(*gorm.DB) error {
	a.FetchedAt = utcBound(a.FetchedAt)
	a.VerifiedAt = utcBoundPtr(a.VerifiedAt)
	a.ContentMatchedAt = utcBoundPtr(a.ContentMatchedAt)

	return nil
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

func (f *BeaconStateFilter) ApplyToQuery(query *gorm.DB) (*gorm.DB, error) {
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

// beaconStateDistinct declares how each requested field's distinct values are resolved.
// Loose-scannable fields lead an index right after network: node via
// ix_beacon_states_network_node_fetched_at(network, node, fetched_at), slot via
// ux_beacon_states_dedupe(network, slot, state_root, node), and network leads both.
var beaconStateDistinct = distinctTable{
	name: "beacon_states",
	fields: map[string]distinctStrategy{
		KeyNode:                 distinctLooseScan,
		KeySlot:                 distinctLooseScan,
		KeyNetwork:              distinctLooseScan,
		KeyEpoch:                distinctFullScan,
		KeyStateRoot:            distinctFullScan,
		KeyNodeVersion:          distinctFullScan,
		KeyLocation:             distinctFullScan,
		KeyBeaconImplementation: distinctFullScan,
	},
}

func (i *Indexer) DistinctBeaconStateValues(ctx context.Context, fields []string, network string) (*DistinctBeaconStateValueResults, error) {
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

	seen := make(map[string]bool, len(fields))

	for _, field := range fields {
		if seen[field] {
			continue
		}

		seen[field] = true

		values, err := i.distinctFieldValues(ctx, beaconStateDistinct, field, network)
		if err != nil {
			i.metrics.ObserveOperationError(operation)

			return nil, err
		}

		switch field {
		case KeyNode:
			results.Node = distinctStrings(values)
		case KeySlot:
			results.Slot = distinctUint64s(values)
		case KeyEpoch:
			results.Epoch = distinctUint64s(values)
		case KeyStateRoot:
			results.StateRoot = distinctStrings(values)
		case KeyNodeVersion:
			results.NodeVersion = distinctStrings(values)
		case KeyLocation:
			results.Location = distinctStrings(values)
		case KeyNetwork:
			results.Network = distinctStrings(values)
		case KeyBeaconImplementation:
			results.BeaconImplementation = distinctStrings(values)
		}
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
