package persistence

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *Indexer {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "lock.db")), &gorm.Config{})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)

	// Serialise at the pool so concurrent writers contend on the conditional upsert rather
	// than on SQLite's file lock.
	sqlDB.SetMaxOpenConns(1)

	err = db.AutoMigrate(&DistributedLock{})
	require.NoError(t, err)

	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	return &Indexer{
		db:  db,
		log: log,
	}
}

func TestAcquireLock(t *testing.T) {
	ctx := context.Background()
	indexer := setupTestDB(t)

	t.Run("acquire new lock", func(t *testing.T) {
		acquired, err := indexer.AcquireLock(ctx, "test-key-1", "owner-1", 10*time.Second)
		require.NoError(t, err)
		assert.True(t, acquired)

		var lock DistributedLock

		result := indexer.db.Where("key = ?", "test-key-1").First(&lock)
		require.NoError(t, result.Error)

		assert.Equal(t, "test-key-1", lock.Key)
		assert.Equal(t, "owner-1", lock.Owner)
		assert.True(t, lock.ExpiresAt.After(time.Now()))
	})

	t.Run("same owner extends the lock", func(t *testing.T) {
		acquired, err := indexer.AcquireLock(ctx, "test-key-2", "owner-2", 10*time.Second)
		require.NoError(t, err)
		assert.True(t, acquired)

		var before DistributedLock

		result := indexer.db.Where("key = ?", "test-key-2").First(&before)
		require.NoError(t, result.Error)

		time.Sleep(10 * time.Millisecond)

		acquired, err = indexer.AcquireLock(ctx, "test-key-2", "owner-2", 10*time.Second)
		require.NoError(t, err)
		assert.True(t, acquired)

		var after DistributedLock

		result = indexer.db.Where("key = ?", "test-key-2").First(&after)
		require.NoError(t, result.Error)

		assert.Equal(t, "owner-2", after.Owner)
		assert.True(t, after.ExpiresAt.After(before.ExpiresAt))
	})

	t.Run("a held lock is not an error", func(t *testing.T) {
		acquired, err := indexer.AcquireLock(ctx, "test-key-3", "owner-3", 10*time.Second)
		require.NoError(t, err)
		assert.True(t, acquired)

		acquired, err = indexer.AcquireLock(ctx, "test-key-3", "owner-4", 10*time.Second)
		require.NoError(t, err)
		assert.False(t, acquired)

		var lock DistributedLock

		result := indexer.db.Where("key = ?", "test-key-3").First(&lock)
		require.NoError(t, result.Error)

		assert.Equal(t, "owner-3", lock.Owner)
	})

	t.Run("expired lock is taken over", func(t *testing.T) {
		result := indexer.db.Create(&DistributedLock{
			Key:       "test-key-4",
			Owner:     "owner-5",
			ExpiresAt: time.Now().Add(-1 * time.Second),
		})
		require.NoError(t, result.Error)

		acquired, err := indexer.AcquireLock(ctx, "test-key-4", "owner-6", 10*time.Second)
		require.NoError(t, err)
		assert.True(t, acquired)

		var lock DistributedLock

		result = indexer.db.Where("key = ?", "test-key-4").First(&lock)
		require.NoError(t, result.Error)

		assert.Equal(t, "owner-6", lock.Owner)
		assert.True(t, lock.ExpiresAt.After(time.Now()))
	})
}

func TestReleaseLock(t *testing.T) {
	ctx := context.Background()
	indexer := setupTestDB(t)

	t.Run("release owned lock", func(t *testing.T) {
		acquired, err := indexer.AcquireLock(ctx, "test-key-5", "owner-7", 10*time.Second)
		require.NoError(t, err)
		assert.True(t, acquired)

		err = indexer.ReleaseLock(ctx, "test-key-5", "owner-7")
		require.NoError(t, err)

		var lock DistributedLock

		result := indexer.db.Where("key = ?", "test-key-5").First(&lock)
		require.ErrorIs(t, result.Error, gorm.ErrRecordNotFound)
	})

	t.Run("release non-existent lock", func(t *testing.T) {
		err := indexer.ReleaseLock(ctx, "non-existent-key", "owner-8")
		require.NoError(t, err)
	})

	t.Run("release lock owned by different owner", func(t *testing.T) {
		acquired, err := indexer.AcquireLock(ctx, "test-key-6", "owner-9", 10*time.Second)
		require.NoError(t, err)
		assert.True(t, acquired)

		err = indexer.ReleaseLock(ctx, "test-key-6", "owner-10")
		require.NoError(t, err)

		var lock DistributedLock

		result := indexer.db.Where("key = ?", "test-key-6").First(&lock)
		require.NoError(t, result.Error)
		assert.Equal(t, "owner-9", lock.Owner)
	})

	t.Run("released lock can be reacquired by anyone", func(t *testing.T) {
		acquired, err := indexer.AcquireLock(ctx, "test-key-7", "owner-11", 10*time.Second)
		require.NoError(t, err)
		assert.True(t, acquired)

		acquired, err = indexer.AcquireLock(ctx, "test-key-7", "owner-12", 10*time.Second)
		require.NoError(t, err)
		assert.False(t, acquired)

		require.NoError(t, indexer.ReleaseLock(ctx, "test-key-7", "owner-11"))

		acquired, err = indexer.AcquireLock(ctx, "test-key-7", "owner-12", 10*time.Second)
		require.NoError(t, err)
		assert.True(t, acquired)

		var lock DistributedLock

		result := indexer.db.Where("key = ?", "test-key-7").First(&lock)
		require.NoError(t, result.Error)
		assert.Equal(t, "owner-12", lock.Owner)
	})
}

func TestCleanupExpiredLocks(t *testing.T) {
	ctx := context.Background()
	indexer := setupTestDB(t)

	for i := range 5 {
		result := indexer.db.Create(&DistributedLock{
			Key:       "expired-key-" + string(rune(i+'0')),
			Owner:     "owner-expired",
			ExpiresAt: time.Now().Add(-1 * time.Second),
		})
		require.NoError(t, result.Error)
	}

	for i := range 3 {
		result := indexer.db.Create(&DistributedLock{
			Key:       "valid-key-" + string(rune(i+'0')),
			Owner:     "owner-valid",
			ExpiresAt: time.Now().Add(10 * time.Second),
		})
		require.NoError(t, result.Error)
	}

	var countBefore int64

	indexer.db.Model(&DistributedLock{}).Count(&countBefore)

	assert.Equal(t, int64(8), countBefore)

	require.NoError(t, indexer.cleanupExpiredLocks(ctx))

	var locks []DistributedLock

	result := indexer.db.Find(&locks)
	require.NoError(t, result.Error)

	assert.Len(t, locks, 3)

	for _, lock := range locks {
		assert.True(t, lock.ExpiresAt.After(time.Now()))
		assert.Contains(t, lock.Key, "valid-key-")
	}
}

func TestConcurrentLockAcquisition(t *testing.T) {
	ctx := context.Background()
	indexer := setupTestDB(t)

	const contenders = 8

	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		winners []string
		start   = make(chan struct{})
	)

	for i := range contenders {
		owner := "owner-" + string(rune(i+'a'))

		wg.Add(1)

		go func() {
			defer wg.Done()

			<-start

			acquired, err := indexer.AcquireLock(ctx, "contended-key", owner, 10*time.Second)
			if err != nil {
				mu.Lock()
				defer mu.Unlock()

				t.Errorf("unexpected error acquiring lock: %v", err)

				return
			}

			if acquired {
				mu.Lock()
				defer mu.Unlock()

				winners = append(winners, owner)
			}
		}()
	}

	close(start)
	wg.Wait()

	require.Len(t, winners, 1, "exactly one contender must win the lock")

	var lock DistributedLock

	result := indexer.db.Where("key = ?", "contended-key").First(&lock)
	require.NoError(t, result.Error)
	assert.Equal(t, winners[0], lock.Owner)
}
