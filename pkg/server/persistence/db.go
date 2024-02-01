package persistence

import (
	"context"
	"errors"
	"strings"

	"github.com/glebarez/sqlite"
	perrors "github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/plugin/prometheus"
)

type Indexer struct {
	db      *gorm.DB
	log     logrus.FieldLogger
	metrics *BasicMetrics
}

func NewIndexer(namespace string, log logrus.FieldLogger, config Config) (*Indexer, error) {
	namespace += "_indexer"

	var db *gorm.DB

	var err error

	switch config.DriverName {
	case "postgres":
		conf := postgres.Config{
			DSN:        config.DSN,
			DriverName: "postgres",
		}

		dialect := postgres.New(conf)

		db, err = gorm.Open(dialect, &gorm.Config{})
	case "sqlite":
		db, err = gorm.Open(sqlite.Open(config.DSN), &gorm.Config{})
	default:
		return nil, errors.New("invalid driver name: " + config.DriverName)
	}

	if err != nil {
		return nil, err
	}

	db = db.Session(&gorm.Session{FullSaveAssociations: true})

	if err = db.Use(
		prometheus.New(prometheus.Config{
			DBName:          "tracoor",
			RefreshInterval: 15,
			StartServer:     false,
		}),
	); err != nil {
		return nil, perrors.Wrap(err, "failed to register prometheus plugin")
	}

	return &Indexer{
		db:      db,
		log:     log.WithField("component", "indexer"),
		metrics: NewBasicMetrics(namespace, config.DriverName, true),
	}, nil
}

func (i *Indexer) Start(ctx context.Context) error {
	i.log.Info("Starting indexer")

	err := i.db.AutoMigrate(&BeaconState{})
	if err != nil {
		return perrors.Wrap(err, "failed to auto migrate beacon state")
	}

	return nil
}

func (i *Indexer) Stop(ctx context.Context) error {
	i.log.Info("Stopping indexer")

	return nil
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

func contains(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
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
