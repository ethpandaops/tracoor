package persistence

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/glebarez/sqlite"
	perrors "github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/plugin/prometheus"
)

// Connection pool sizing.
//
// SQLite: one connection, always. The pure-Go driver serialises writers anyway, and a DSN of
// the `file:name?mode=memory` form gives every *connection* its own database, so a pool wider
// than one is not a throughput knob but a correctness bug. The connection is never retired for
// the same reason.
//
// Postgres: sized for an indexer that runs a handful of concurrent gRPC handlers plus the
// retention loop, with a lifetime short enough that a rolling database upgrade drains.
const (
	sqliteMaxOpenConns = 1
	sqliteMaxIdleConns = 1
	pgMaxOpenConns     = 16
	pgMaxIdleConns     = 8
	pgConnMaxLifetime  = time.Hour
)

// sqlitePragmas are carried on the DSN rather than executed after open: a pragma run through
// the pool lands on whichever connection served it, while DSN pragmas are applied to every
// connection the driver makes.
//
//	synchronous(0)      - the index is rebuildable from the object store; fsync per commit is
//	                      not worth the write amplification.
//	cache_size(-400000) - 400 MB of page cache, in KiB as SQLite's negative form.
//	busy_timeout(10000) - wait out a writer instead of failing the request immediately.
//	journal_mode(WAL)   - readers do not block the writer.
const sqlitePragmas = "_pragma=synchronous(0)&_pragma=cache_size(-400000)&_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)"

type Indexer struct {
	db      *gorm.DB
	log     logrus.FieldLogger
	metrics *BasicMetrics
}

// withSQLitePragmas appends the standard pragmas to a DSN that does not already carry any. An
// operator who has spelled out their own pragmas gets them respected verbatim.
func withSQLitePragmas(dsn string) string {
	if strings.Contains(dsn, "_pragma=") {
		return dsn
	}

	separator := "?"
	if strings.Contains(dsn, "?") {
		separator = "&"
	}

	return dsn + separator + sqlitePragmas
}

func configureConnectionPool(db *gorm.DB, driverName string) error {
	sqlDB, err := db.DB()
	if err != nil {
		return perrors.Wrap(err, "failed to get underlying sql db")
	}

	switch driverName {
	case "sqlite":
		sqlDB.SetMaxOpenConns(sqliteMaxOpenConns)
		sqlDB.SetMaxIdleConns(sqliteMaxIdleConns)
		sqlDB.SetConnMaxLifetime(0)
	default:
		sqlDB.SetMaxOpenConns(pgMaxOpenConns)
		sqlDB.SetMaxIdleConns(pgMaxIdleConns)
		sqlDB.SetConnMaxLifetime(pgConnMaxLifetime)
	}

	return nil
}

func NewIndexer(namespace string, log logrus.FieldLogger, config Config, opts *Options) (*Indexer, error) {
	namespace += "_indexer"

	var db *gorm.DB

	var err error

	// Statements are reused across the process: every query this package issues comes from a
	// fixed set of shapes, so caching the prepared form is a pure win on both engines.
	//
	// NowFunc is UTC for the same reason every bound timestamp is: the SQLite driver stores a
	// time.Time as text carrying its own offset, so a timestamp gorm fills in from the host
	// clock would not order against the UTC values this package writes.
	gormConfig := &gorm.Config{
		PrepareStmt: true,
		NowFunc:     func() time.Time { return time.Now().UTC() },
	}

	switch config.DriverName {
	case "postgres":
		conf := postgres.Config{
			DSN:        config.DSN,
			DriverName: "postgres",
		}

		dialect := postgres.New(conf)

		db, err = gorm.Open(dialect, gormConfig)
	case "sqlite":
		db, err = gorm.Open(sqlite.Open(withSQLitePragmas(config.DSN)), gormConfig)
	default:
		return nil, errors.New("invalid driver name: " + config.DriverName)
	}

	if err != nil {
		return nil, err
	}

	if err = configureConnectionPool(db, config.DriverName); err != nil {
		return nil, err
	}

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

	err = i.db.AutoMigrate(&ExecutionPayloadEnvelope{})
	if err != nil {
		return perrors.Wrap(err, "failed to auto migrate execution payload envelope")
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

	err = i.db.AutoMigrate(&Blob{})
	if err != nil {
		return perrors.Wrap(err, "failed to auto migrate blob")
	}

	err = i.db.AutoMigrate(&PayloadDivergence{})
	if err != nil {
		return perrors.Wrap(err, "failed to auto migrate payload divergence")
	}

	err = i.db.AutoMigrate(&DistributedLock{})
	if err != nil {
		return perrors.Wrap(err, "failed to auto migrate distributed lock")
	}

	err = i.db.AutoMigrate(&PermanentBlock{})
	if err != nil {
		return perrors.Wrap(err, "failed to auto migrate permanent block")
	}

	return nil
}

func (i *Indexer) Stop(ctx context.Context) error {
	i.log.Info("Stopping indexer")

	return nil
}
