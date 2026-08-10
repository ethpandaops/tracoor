package persistence

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

type BeaconBadBlock struct {
	ID   string `gorm:"primaryKey"`
	Node string `gorm:"not null;default:'';uniqueIndex:ux_beacon_bad_blocks_dedupe,priority:4;index:ix_beacon_bad_blocks_network_node_fetched_at,priority:2"`
	// We have to use int64 here as SQLite doesn't support uint64. This sucks
	// but slot 9223372036854775808 is probably around the heat death
	// of the universe so we should be OK.
	Slot                 int64     `gorm:"not null;default:0;uniqueIndex:ux_beacon_bad_blocks_dedupe,priority:2"`
	Epoch                int64     `gorm:"not null;default:0"`
	BlockRoot            string    `gorm:"not null;default:'';uniqueIndex:ux_beacon_bad_blocks_dedupe,priority:3"`
	FetchedAt            time.Time `gorm:"not null;index:ix_beacon_bad_blocks_fetched_at;index:ix_beacon_bad_blocks_network_node_fetched_at,priority:3;index:ix_beacon_bad_blocks_network_fetched_at,priority:2"`
	BeaconImplementation string    `gorm:"not null;default:''"`
	NodeVersion          string    `gorm:"not null;default:''"`
	ContentEncoding      string    `gorm:"not null;default:''"`
	Location             string    `gorm:"not null;default:''"`
	ContentHash          string    `gorm:"not null;default:'';size:64"`
	// VerifiedAt is set when these bytes were read and hashed from this node.
	VerifiedAt *time.Time
	// ContentMatchedAt is set when the hash was compared against an existing
	// payload and matched. Bad blocks are never linked, so it stays null.
	ContentMatchedAt *time.Time
	Network          string `gorm:"not null;default:'';uniqueIndex:ux_beacon_bad_blocks_dedupe,priority:1;index:ix_beacon_bad_blocks_network_node_fetched_at,priority:1;index:ix_beacon_bad_blocks_network_fetched_at,priority:1"`
}

// BeforeSave keeps every stored timestamp in UTC. The drivers render a time.Time in the zone
// the value itself carries, so a row written by a process in another zone would neither order
// nor compare against the rest of the table.
func (a *BeaconBadBlock) BeforeSave(*gorm.DB) error {
	a.FetchedAt = utcBound(a.FetchedAt)
	a.VerifiedAt = utcBoundPtr(a.VerifiedAt)
	a.ContentMatchedAt = utcBoundPtr(a.ContentMatchedAt)

	return nil
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

func (f *BeaconBadBlockFilter) ApplyToQuery(query *gorm.DB) (*gorm.DB, error) {
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

// beaconBadBlockDistinct declares how each requested field's distinct values are resolved.
// Loose-scannable fields lead an index right after network: node via
// ix_beacon_bad_blocks_network_node_fetched_at(network, node, fetched_at), slot via
// ux_beacon_bad_blocks_dedupe(network, slot, block_root, node), and network leads both.
var beaconBadBlockDistinct = distinctTable{
	name: "beacon_bad_blocks",
	fields: map[string]distinctStrategy{
		KeyNode:                 distinctLooseScan,
		KeySlot:                 distinctLooseScan,
		KeyNetwork:              distinctLooseScan,
		KeyEpoch:                distinctFullScan,
		KeyBlockRoot:            distinctFullScan,
		KeyNodeVersion:          distinctFullScan,
		KeyLocation:             distinctFullScan,
		KeyBeaconImplementation: distinctFullScan,
	},
}

func (i *Indexer) DistinctBeaconBadBlockValues(ctx context.Context, fields []string, network string) (*DistinctBeaconBadBlockValueResults, error) {
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

	seen := make(map[string]bool, len(fields))

	for _, field := range fields {
		if seen[field] {
			continue
		}

		seen[field] = true

		values, err := i.distinctFieldValues(ctx, beaconBadBlockDistinct, field, network)
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
		case KeyBlockRoot:
			results.BlockRoot = distinctStrings(values)
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
