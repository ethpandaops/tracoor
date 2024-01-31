package persistence

import (
	"context"
	"errors"
	"strconv"

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

	result := query.Order("fetched_at ASC").Find(&BeaconStates).Limit(1000)
	if result.Error != nil {
		i.metrics.ObserveOperationError(operation)

		return nil, result.Error
	}

	return BeaconStates, nil
}

type DistinctValueResults struct {
	Node        []string `json:"node"`
	Slot        []uint64 `json:"slot"`
	Epoch       []uint64 `json:"epoch"`
	StateRoot   []string `json:"state_root"`
	NodeVersion []string `json:"node_version"`
	Location    []string `json:"location"`
	Network     []string `json:"network"`
}

func (i *Indexer) DistinctValues(ctx context.Context, entity interface{}, fields []string) (*DistinctValueResults, error) {
	operation := OperationDistinctValues

	i.metrics.ObserveOperation(operation)

	results := &DistinctValueResults{}
	query := i.db.WithContext(ctx).Model(entity).Select(fields).Distinct()

	rows, err := query.Rows()
	if err != nil {
		i.metrics.ObserveOperationError(operation)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			i.metrics.ObserveOperationError(operation)
			return nil, err
		}
		switch {
		case contains(fields, "node"):
			results.Node = append(results.Node, value)
		case contains(fields, "slot"):
			slot, err := strconv.ParseUint(value, 10, 64)
			if err != nil {
				i.metrics.ObserveOperationError(operation)

				return nil, err
			}

			results.Slot = append(results.Slot, slot)
		case contains(fields, "epoch"):
			epoch, err := strconv.ParseUint(value, 10, 64)
			if err != nil {
				i.metrics.ObserveOperationError(operation)

				return nil, err
			}

			results.Epoch = append(results.Epoch, epoch)
		case contains(fields, "state_root"):
			results.StateRoot = append(results.StateRoot, value)
		case contains(fields, "node_version"):
			results.NodeVersion = append(results.NodeVersion, value)
		case contains(fields, "location"):
			results.Location = append(results.Location, value)
		case contains(fields, "network"):
			results.Network = append(results.Network, value)
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
func (i *Indexer) ListNodesWithBeaconStates(ctx context.Context, filter *BeaconStateFilter, page *PaginationCursor) ([]string, error) {
	operation := OperationListBeaconState

	i.metrics.ObserveOperation(operation)

	var nodes []string

	query := i.db.WithContext(ctx).Model(&BeaconState{})

	query, err := filter.ApplyToQuery(query)
	if err != nil {
		i.metrics.ObserveOperationError(operation)

		return nil, err
	}

	if page != nil {
		query = page.ApplyOffsetLimit(query)
	}

	result := query.Distinct("node").Order("node ASC").Find(&nodes)
	if result.Error != nil {
		i.metrics.ObserveOperationError(operation)

		return nil, result.Error
	}

	return nodes, nil
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
