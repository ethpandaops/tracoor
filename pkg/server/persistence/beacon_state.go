package persistence

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

type BeaconState struct {
	gorm.Model
	ID   string `gorm:"primaryKey"`
	Node string `gorm:"index"`
	// We have to use int64 here as SQLite doesn't support uint64. This sucks
	// but slot 9223372036854775808 is probably around the heat death
	// of the universe so we should be OK.
	Slot                 int64 `gorm:"index:idx_slot,where:deleted_at IS NULL"`
	Epoch                int64
	StateRoot            string
	FetchedAt            time.Time `gorm:"index"`
	BeaconImplementation string
	NodeVersion          string `gorm:"not null;default:''"`
	Location             string `gorm:"not null;default:''"`
	Network              string `gorm:"not null;default:''"`
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

func (f *BeaconStateFilter) AddBeaconImplementation(BeaconImplementation string) {
	f.BeaconImplementation = &BeaconImplementation
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
