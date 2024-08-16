package persistence

import (
	"context"
	"errors"

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

func NewIndexer(namespace string, log logrus.FieldLogger, config Config, opts *Options) (*Indexer, error) {
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
		db.Exec("PRAGMA synchronous = OFF;")
		db.Exec("PRAGMA journal_mode = WAL;")
		db.Exec("PRAGMA cache_size = 100000;")
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
		metrics: NewBasicMetrics(namespace, config.DriverName, opts.MetricsEnabled),
	}, nil
}

func (i *Indexer) Start(ctx context.Context) error {
	i.log.Info("Starting indexer")

	err := i.db.AutoMigrate(&BeaconState{})
	if err != nil {
		return perrors.Wrap(err, "failed to auto migrate beacon state")
	}

	err = i.db.AutoMigrate(&BeaconBlock{})
	if err != nil {
		return perrors.Wrap(err, "failed to auto migrate beacon block")
	}

	err = i.db.AutoMigrate(&BeaconBadBlock{})
	if err != nil {
		return perrors.Wrap(err, "failed to auto migrate beacon bad block")
	}

	err = i.db.AutoMigrate(&BeaconBadBlob{})
	if err != nil {
		return perrors.Wrap(err, "failed to auto migrate beacon bad blob")
	}

	err = i.db.AutoMigrate(&ExecutionBlockTrace{})
	if err != nil {
		return perrors.Wrap(err, "failed to auto migrate execution block trace")
	}

	err = i.db.AutoMigrate(&ExecutionBadBlock{})
	if err != nil {
		return perrors.Wrap(err, "failed to auto migrate execution bad block")
	}

	return nil
}

func (i *Indexer) Stop(ctx context.Context) error {
	i.log.Info("Stopping indexer")

	return nil
}
