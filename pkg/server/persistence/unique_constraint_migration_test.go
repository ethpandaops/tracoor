package persistence_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/ethpandaops/tracoor/pkg/server/persistence"
	"github.com/glebarez/sqlite"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// legacyBeaconState mirrors BeaconState's shape from before this change --
// same table, same columns, but no unique constraint -- so it can be used to
// stand up a database that looks exactly like one a pre-fix version of this
// code would have produced, via gorm's own migrator rather than a hand
// written CREATE TABLE that would need to be kept in sync by hand.
type legacyBeaconState struct {
	gorm.Model
	ID                   string `gorm:"primaryKey"`
	Node                 string
	Slot                 int64
	Epoch                int64
	StateRoot            string
	FetchedAt            time.Time
	BeaconImplementation string
	NodeVersion          string `gorm:"not null;default:''"`
	ContentEncoding      string `gorm:"not null;default:''"`
	Location             string `gorm:"not null;default:''"`
	Network              string `gorm:"not null;default:''"`
}

func (legacyBeaconState) TableName() string { return "beacon_states" }

func newFileBackedTestDB(t *testing.T) (string, func()) {
	t.Helper()

	dbFile, err := os.CreateTemp("", "unique_constraint_migration_*.db")
	require.NoError(t, err)

	dbPath := dbFile.Name()
	dbFile.Close()
	os.Remove(dbPath)

	cleanup := func() {
		os.Remove(dbPath)
		os.Remove(dbPath + "-wal")
		os.Remove(dbPath + "-shm")
	}

	return dbPath, cleanup
}

// TestMigration_SucceedsOnFreshDatabase is the baseline: a database that has
// never been migrated before must still start up cleanly. The dedup step
// must not error out just because the tables it's looking for don't exist
// yet.
func TestMigration_SucceedsOnFreshDatabase(t *testing.T) {
	dbPath, cleanup := newFileBackedTestDB(t)
	defer cleanup()

	idx, err := persistence.NewIndexer("migration-test", logrus.New(), persistence.Config{
		DSN:        fmt.Sprintf("file:%s?parseTime=True", dbPath),
		DriverName: "sqlite",
	}, persistence.DefaultOptions().SetMetricsEnabled(false))
	require.NoError(t, err)

	require.NoError(t, idx.Start(context.Background()))
}

// TestMigration_DeduplicatesExistingRowsBeforeAddingConstraint is the
// highest-stakes test for this change: a database that already has
// duplicate rows -- exactly what the pre-existing TOCTOU race and the
// execution handlers' missing dedup check are known to produce -- must not
// fail to start when it upgrades to a version that adds a unique
// constraint. If this broke, every deployment that had ever hit the
// original race would fail to start after upgrading.
func TestMigration_DeduplicatesExistingRowsBeforeAddingConstraint(t *testing.T) {
	dbPath, cleanup := newFileBackedTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Seed the table exactly as the PRE-this-change schema would have left
	// it: created via gorm's own migrator against the legacy (no unique
	// constraint) shape, with duplicate rows already present, standing in
	// for what the original TOCTOU race actually produced in a real
	// deployment over time. This can't be done by calling idx.Start()
	// first, since that already runs the current (patched) migration -- the
	// whole point of this test is to simulate a database that predates the
	// fix.
	legacyDB, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?parseTime=True", dbPath)), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, legacyDB.AutoMigrate(&legacyBeaconState{}))

	now := time.Now()

	for i := 0; i < 3; i++ {
		require.NoError(t, legacyDB.Create(&legacyBeaconState{
			ID:                   fmt.Sprintf("dup-%d", i),
			Node:                 "race-node",
			Network:              "mainnet",
			Slot:                 100,
			Epoch:                3,
			StateRoot:            "0xduplicated",
			FetchedAt:            now.Add(time.Duration(i) * time.Millisecond),
			BeaconImplementation: "teku",
			NodeVersion:          "1.0.0",
			Location:             "beacon_state/dup.ssz",
		}).Error)
	}

	require.NoError(t, legacyDB.Create(&legacyBeaconState{
		ID:                   "distinct-row",
		Node:                 "other-node",
		Network:              "mainnet",
		Slot:                 200,
		Epoch:                6,
		StateRoot:            "0xdistinct",
		FetchedAt:            now,
		BeaconImplementation: "teku",
		NodeVersion:          "1.0.0",
		Location:             "beacon_state/distinct.ssz",
	}).Error)

	rawConn, err := legacyDB.DB()
	require.NoError(t, err)
	require.NoError(t, rawConn.Close())

	// Startup with the current (patched) code, against this pre-existing,
	// duplicate-containing database. This is the exact code path a real
	// deployment hits on upgrade.
	idx2, err := persistence.NewIndexer("migration-test-upgrade", logrus.New(), persistence.Config{
		DSN:        fmt.Sprintf("file:%s?parseTime=True", dbPath),
		DriverName: "sqlite",
	}, persistence.DefaultOptions().SetMetricsEnabled(false))
	require.NoError(t, err)

	err = idx2.Start(ctx)
	require.NoError(t, err, "migration must succeed even when pre-existing duplicate rows are present")

	// Exactly one of the three duplicates must remain, and it must be the
	// earliest one (deterministic, not arbitrary).
	remaining, err := idx2.ListBeaconState(ctx, &persistence.BeaconStateFilter{Node: strPtr("race-node")}, nil)
	require.NoError(t, err)
	require.Len(t, remaining, 1, "expected exactly one surviving row per natural key after dedup")
	require.Equal(t, "dup-0", remaining[0].ID, "expected the earliest row (by FetchedAt/insertion order) to survive")

	// The unrelated, non-duplicated row must be completely untouched.
	distinct, err := idx2.ListBeaconState(ctx, &persistence.BeaconStateFilter{ID: strPtr("distinct-row")}, nil)
	require.NoError(t, err)
	require.Len(t, distinct, 1, "the non-duplicated row must survive untouched")

	// And the constraint must now actually be enforced: a fresh attempt to
	// insert a fourth duplicate must fail.
	dupErr := idx2.InsertBeaconState(ctx, &persistence.BeaconState{
		ID:                   "dup-attempt-after-migration",
		Node:                 "race-node",
		Network:              "mainnet",
		Slot:                 100,
		Epoch:                3,
		StateRoot:            "0xduplicated",
		FetchedAt:            now,
		BeaconImplementation: "teku",
		NodeVersion:          "1.0.0",
		Location:             "beacon_state/dup.ssz",
	})
	require.Error(t, dupErr, "expected the unique constraint to reject a new duplicate after migration")
	require.True(t, persistence.IsUniqueConstraintError(dupErr))
}

func strPtr(s string) *string { return &s }
