package persistence

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

type ExecutionPayloadEnvelope struct {
	ID   string `gorm:"primaryKey"`
	Node string `gorm:"not null;default:'';uniqueIndex:ux_execution_payload_envelopes_dedupe,priority:4;index:ix_execution_payload_envelopes_network_node_fetched_at,priority:2"`
	// We have to use int64 here as SQLite doesn't support uint64. This sucks
	// but slot 9223372036854775808 is probably around the heat death
	// of the universe so we should be OK.
	Slot                 int64     `gorm:"not null;default:0;uniqueIndex:ux_execution_payload_envelopes_dedupe,priority:2"`
	Epoch                int64     `gorm:"not null;default:0"`
	BlockRoot            string    `gorm:"not null;default:'';uniqueIndex:ux_execution_payload_envelopes_dedupe,priority:3"`
	FetchedAt            time.Time `gorm:"not null;index:ix_execution_payload_envelopes_fetched_at;index:ix_execution_payload_envelopes_network_node_fetched_at,priority:3;index:ix_execution_payload_envelopes_network_fetched_at,priority:2"`
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
	Network          string `gorm:"not null;default:'';uniqueIndex:ux_execution_payload_envelopes_dedupe,priority:1;index:ix_execution_payload_envelopes_network_node_fetched_at,priority:1;index:ix_execution_payload_envelopes_network_fetched_at,priority:1"`
}

type ExecutionPayloadEnvelopeFilter struct {
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

func (f *ExecutionPayloadEnvelopeFilter) AddID(id string) {
	f.ID = &id
}

func (f *ExecutionPayloadEnvelopeFilter) AddNode(node string) {
	f.Node = &node
}

func (f *ExecutionPayloadEnvelopeFilter) AddBefore(before time.Time) {
	f.Before = &before
}

func (f *ExecutionPayloadEnvelopeFilter) AddAfter(after time.Time) {
	f.After = &after
}

func (f *ExecutionPayloadEnvelopeFilter) AddSlot(slot uint64) {
	f.Slot = &slot
}

func (f *ExecutionPayloadEnvelopeFilter) AddEpoch(epoch uint64) {
	f.Epoch = &epoch
}

func (f *ExecutionPayloadEnvelopeFilter) AddBlockRoot(blockRoot string) {
	f.BlockRoot = &blockRoot
}

func (f *ExecutionPayloadEnvelopeFilter) AddNodeVersion(nodeVersion string) {
	f.NodeVersion = &nodeVersion
}

func (f *ExecutionPayloadEnvelopeFilter) AddLocation(location string) {
	f.Location = &location
}

func (f *ExecutionPayloadEnvelopeFilter) AddNetwork(network string) {
	f.Network = &network
}

func (f *ExecutionPayloadEnvelopeFilter) AddBeaconImplementation(beaconImplementation string) {
	f.BeaconImplementation = &beaconImplementation
}

func (f *ExecutionPayloadEnvelopeFilter) ApplyToQuery(query *gorm.DB) (*gorm.DB, error) {
	if f.ID != nil {
		query = query.Where("id = ?", f.ID)
	}

	if f.Node != nil {
		query = query.Where("node = ?", f.Node)
	}

	if f.Before != nil {
		query = query.Where("fetched_at <= ?", *f.Before)
	}

	if f.After != nil {
		query = query.Where("fetched_at >= ?", *f.After)
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

func (i *Indexer) InsertExecutionPayloadEnvelope(ctx context.Context, envelope *ExecutionPayloadEnvelope) error {
	operation := OperationInsertExecutionPayloadEnvelope
	i.metrics.ObserveOperation(operation)

	result := i.db.WithContext(ctx).Create(envelope)
	if result.Error != nil {
		i.metrics.ObserveOperationError(operation)
	}

	return result.Error
}

func (i *Indexer) RemoveExecutionPayloadEnvelope(ctx context.Context, id string) error {
	operation := OperationDeleteExecutionPayloadEnvelope

	i.metrics.ObserveOperation(operation)

	result := i.db.WithContext(ctx).Unscoped().Where("id = ?", id).Delete(&ExecutionPayloadEnvelope{})

	if result.Error != nil {
		i.metrics.ObserveOperationError(operation)
	}

	return result.Error
}

func (i *Indexer) CountExecutionPayloadEnvelope(ctx context.Context, filter *ExecutionPayloadEnvelopeFilter) (int64, error) {
	operation := OperationCountExecutionPayloadEnvelope

	i.metrics.ObserveOperation(operation)

	var count int64

	query := i.db.WithContext(ctx).Model(&ExecutionPayloadEnvelope{})

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

func (i *Indexer) ListExecutionPayloadEnvelope(ctx context.Context, filter *ExecutionPayloadEnvelopeFilter, page *PaginationCursor) ([]*ExecutionPayloadEnvelope, error) {
	operation := OperationListExecutionPayloadEnvelope

	i.metrics.ObserveOperation(operation)

	var ExecutionPayloadEnvelopes []*ExecutionPayloadEnvelope

	query := i.db.WithContext(ctx).Model(&ExecutionPayloadEnvelope{})

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

	result := query.Find(&ExecutionPayloadEnvelopes)
	if result.Error != nil {
		i.metrics.ObserveOperationError(operation)

		return nil, result.Error
	}

	return ExecutionPayloadEnvelopes, nil
}

type DistinctExecutionPayloadEnvelopeValueResults struct {
	Node                 []string
	Slot                 []uint64
	Epoch                []uint64
	BlockRoot            []string
	NodeVersion          []string
	Location             []string
	Network              []string
	BeaconImplementation []string
}

//nolint:errcheck // casting fine here.
func (i *Indexer) DistinctExecutionPayloadEnvelopeValues(ctx context.Context, fields []string, network string) (*DistinctExecutionPayloadEnvelopeValueResults, error) {
	operation := OperationDistinctValues

	i.metrics.ObserveOperation(operation)

	results := &DistinctExecutionPayloadEnvelopeValueResults{
		Node:                 make([]string, 0),
		Slot:                 make([]uint64, 0),
		Epoch:                make([]uint64, 0),
		BlockRoot:            make([]string, 0),
		NodeVersion:          make([]string, 0),
		Location:             make([]string, 0),
		Network:              make([]string, 0),
		BeaconImplementation: make([]string, 0),
	}
	query := i.db.WithContext(ctx).Model(&ExecutionPayloadEnvelope{})

	if network != "" {
		query = query.Where("network = ?", network)
	}

	query = query.Select(fields).Group(strings.Join(fields, ", ")).Limit(1000)

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
				case KeyNode:
					results.Node = append(results.Node, values[i].(string))
				case KeySlot:
					//nolint:gosec // not worried about int64 overflow here
					results.Slot = append(results.Slot, uint64(values[i].(int64)))
				case KeyEpoch:
					//nolint:gosec // not worried about int64 overflow here
					results.Epoch = append(results.Epoch, uint64(values[i].(int64)))
				case KeyBlockRoot:
					results.BlockRoot = append(results.BlockRoot, values[i].(string))
				case KeyNodeVersion:
					results.NodeVersion = append(results.NodeVersion, values[i].(string))
				case KeyLocation:
					results.Location = append(results.Location, values[i].(string))
				case KeyNetwork:
					results.Network = append(results.Network, values[i].(string))
				case KeyBeaconImplementation:
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

func (i *Indexer) DeleteExecutionPayloadEnvelope(ctx context.Context, id string) error {
	operation := OperationDeleteExecutionPayloadEnvelope

	i.metrics.ObserveOperation(operation)

	query := i.db.WithContext(ctx)

	result := query.Unscoped().Where("id = ?", id).Delete(&ExecutionPayloadEnvelope{})
	if result.Error != nil {
		i.metrics.ObserveOperationError(operation)

		return result.Error
	}

	if result.RowsAffected == 0 {
		i.metrics.ObserveOperationError(operation)

		return errors.New("execution payload envelope not found")
	}

	return nil
}

func (i *Indexer) UpdateExecutionPayloadEnvelope(ctx context.Context, envelope *ExecutionPayloadEnvelope) error {
	operation := OperationUpdateExecutionPayloadEnvelope

	i.metrics.ObserveOperation(operation)

	query := i.db.WithContext(ctx)

	result := query.Save(envelope)
	if result.Error != nil {
		i.metrics.ObserveOperationError(operation)

		return result.Error
	}

	if result.RowsAffected == 0 {
		i.metrics.ObserveOperationError(operation)

		return errors.New("execution payload envelope not found")
	}

	if result.RowsAffected != 1 {
		i.metrics.ObserveOperationError(operation)

		return errors.New("execution payload envelope update affected more than one row")
	}

	return nil
}
