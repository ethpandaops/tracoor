package persistence

import (
	"context"
	"errors"
	"fmt"
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

	// Each of these tables is gaining a unique constraint on its natural key
	// as part of this migration. A deployment that has been running for a
	// while may already have duplicate rows for the same natural key (that
	// is the exact defect the constraint is being added to prevent), and
	// AutoMigrate would fail outright trying to create a unique index over
	// data that violates it. Deduplicating first, keeping the earliest row
	// per natural key, makes the migration safe to run on an existing
	// database instead of requiring a manual cleanup before upgrading.
	if err := i.dedupeBeforeUniqueIndex(&BeaconState{}, "beacon_states", "node", "network", "slot", "state_root"); err != nil {
		return perrors.Wrap(err, "failed to remove duplicate beacon states")
	}

	err := i.db.AutoMigrate(&BeaconState{})
	if err != nil {
		return perrors.Wrap(err, "failed to auto migrate beacon state")
	}

	if err := i.dedupeBeforeUniqueIndex(&BeaconBlock{}, "beacon_blocks", "node", "network", "slot", "block_root"); err != nil {
		return perrors.Wrap(err, "failed to remove duplicate beacon blocks")
	}

	err = i.db.AutoMigrate(&BeaconBlock{})
	if err != nil {
		return perrors.Wrap(err, "failed to auto migrate beacon block")
	}

	if err := i.dedupeBeforeUniqueIndex(&BeaconBadBlock{}, "beacon_bad_blocks", "node", "network", "slot", "block_root"); err != nil {
		return perrors.Wrap(err, "failed to remove duplicate beacon bad blocks")
	}

	err = i.db.AutoMigrate(&BeaconBadBlock{})
	if err != nil {
		return perrors.Wrap(err, "failed to auto migrate beacon bad block")
	}

	if err := i.dedupeBeforeUniqueIndex(&BeaconBadBlob{}, "beacon_bad_blobs", "node", "network", "slot", "block_root", "index"); err != nil {
		return perrors.Wrap(err, "failed to remove duplicate beacon bad blobs")
	}

	err = i.db.AutoMigrate(&BeaconBadBlob{})
	if err != nil {
		return perrors.Wrap(err, "failed to auto migrate beacon bad blob")
	}

	if err := i.dedupeBeforeUniqueIndex(&ExecutionBlockTrace{}, "execution_block_traces", "node", "network", "block_hash"); err != nil {
		return perrors.Wrap(err, "failed to remove duplicate execution block traces")
	}

	err = i.db.AutoMigrate(&ExecutionBlockTrace{})
	if err != nil {
		return perrors.Wrap(err, "failed to auto migrate execution block trace")
	}

	if err := i.dedupeBeforeUniqueIndex(&ExecutionBadBlock{}, "execution_bad_blocks", "node", "network", "block_hash"); err != nil {
		return perrors.Wrap(err, "failed to remove duplicate execution bad blocks")
	}

	err = i.db.AutoMigrate(&ExecutionBadBlock{})
	if err != nil {
		return perrors.Wrap(err, "failed to auto migrate execution bad block")
	}

	err = i.db.AutoMigrate(&DistributedLock{})
	if err != nil {
		return perrors.Wrap(err, "failed to auto migrate distributed lock")
	}

	if err := i.dedupeBeforeUniqueIndex(&PermanentBlock{}, "permanent_blocks", "block_root", "network"); err != nil {
		return perrors.Wrap(err, "failed to remove duplicate permanent blocks")
	}

	err = i.db.AutoMigrate(&PermanentBlock{})
	if err != nil {
		return perrors.Wrap(err, "failed to auto migrate permanent block")
	}

	return nil
}

// dedupeBeforeUniqueIndex removes all but the earliest row (by created_at,
// falling back to id for a deterministic tiebreak) within each group of rows
// that share the given natural-key columns. It is a no-op if the table
// doesn't exist yet, since a fresh database has nothing to deduplicate.
func (i *Indexer) dedupeBeforeUniqueIndex(dst interface{}, table string, naturalKeyColumns ...string) error {
	if !i.db.Migrator().HasTable(dst) {
		return nil
	}

	quotedColumns := make([]string, len(naturalKeyColumns))
	for idx, column := range naturalKeyColumns {
		quotedColumns[idx] = `"` + column + `"`
	}

	query := fmt.Sprintf(
		`DELETE FROM "%s" WHERE "deleted_at" IS NULL AND "id" NOT IN (`+
			`SELECT "id" FROM (`+
			`SELECT "id", ROW_NUMBER() OVER (PARTITION BY %s ORDER BY "created_at" ASC, "id" ASC) AS row_num `+
			`FROM "%s" WHERE "deleted_at" IS NULL`+
			`) ranked WHERE row_num = 1)`,
		table, strings.Join(quotedColumns, ", "), table,
	)

	result := i.db.Exec(query)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected > 0 {
		i.log.WithFields(logrus.Fields{
			"table": table,
			"rows":  result.RowsAffected,
		}).Warn("Removed duplicate rows before adding a unique constraint")
	}

	return nil
}

func (i *Indexer) Stop(ctx context.Context) error {
	i.log.Info("Stopping indexer")

	return nil
}
