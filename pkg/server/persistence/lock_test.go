package persistence

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *Indexer {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

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

		// Verify lock exists in DB
		var lock DistributedLock
		result := indexer.db.Where("key = ?", "test-key-1").First(&lock)
		require.NoError(t, result.Error)
		assert.Equal(t, "test-key-1", lock.Key)
		assert.Equal(t, "owner-1", lock.Owner)
		assert.True(t, lock.ExpiresAt.After(time.Now()))
	})

	t.Run("acquire existing lock by same owner", func(t *testing.T) {
		// First acquisition
		acquired, err := indexer.AcquireLock(ctx, "test-key-2", "owner-2", 10*time.Second)
		require.NoError(t, err)
		assert.True(t, acquired)

		// Get the original expiry time
		var lock1 DistributedLock

		result := indexer.db.Where("key = ?", "test-key-2").First(&lock1)
		require.NoError(t, result.Error)

		originalExpiry := lock1.ExpiresAt

		// Wait a bit to ensure the new expiry will be different
		time.Sleep(10 * time.Millisecond)

		// Second acquisition by same owner should extend the lock
		acquired, err = indexer.AcquireLock(ctx, "test-key-2", "owner-2", 10*time.Second)
		require.NoError(t, err)
		assert.True(t, acquired)

		// Verify lock was extended
		var lock2 DistributedLock

		result = indexer.db.Where("key = ?", "test-key-2").First(&lock2)
		require.NoError(t, result.Error)

		assert.Equal(t, "owner-2", lock2.Owner)
		assert.True(t, lock2.ExpiresAt.After(originalExpiry))
	})

	t.Run("fail to acquire existing lock by different owner", func(t *testing.T) {
		// First acquisition
		acquired, err := indexer.AcquireLock(ctx, "test-key-3", "owner-3", 10*time.Second)
		require.NoError(t, err)
		assert.True(t, acquired)

		// Second acquisition by different owner should fail
		acquired, err = indexer.AcquireLock(ctx, "test-key-3", "owner-4", 10*time.Second)
		require.Error(t, err)
		assert.False(t, acquired)

		// Verify lock still belongs to original owner
		var lock DistributedLock
		result := indexer.db.Where("key = ?", "test-key-3").First(&lock)
		require.NoError(t, result.Error)
		assert.Equal(t, "owner-3", lock.Owner)
	})

	t.Run("acquire expired lock", func(t *testing.T) {
		// Create an expired lock
		expiredLock := &DistributedLock{
			Key:       "test-key-4",
			Owner:     "owner-5",
			ExpiresAt: time.Now().Add(-1 * time.Second),
		}
		result := indexer.db.Create(expiredLock)
		require.NoError(t, result.Error)

		// New owner should be able to acquire the expired lock
		acquired, err := indexer.AcquireLock(ctx, "test-key-4", "owner-6", 10*time.Second)
		require.NoError(t, err)
		assert.True(t, acquired)

		// Verify lock now belongs to new owner
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
		// First acquire the lock
		acquired, err := indexer.AcquireLock(ctx, "test-key-5", "owner-7", 10*time.Second)
		require.NoError(t, err)
		assert.True(t, acquired)

		// Release the lock
		err = indexer.ReleaseLock(ctx, "test-key-5", "owner-7")
		require.NoError(t, err)

		// Verify lock no longer exists
		var lock DistributedLock
		result := indexer.db.Where("key = ?", "test-key-5").First(&lock)
		assert.Error(t, result.Error)
		assert.True(t, result.Error == gorm.ErrRecordNotFound)
	})

	t.Run("release non-existent lock", func(t *testing.T) {
		// Release a lock that doesn't exist
		err := indexer.ReleaseLock(ctx, "non-existent-key", "owner-8")
		require.NoError(t, err)
	})

	t.Run("release lock owned by different owner", func(t *testing.T) {
		// First acquire the lock
		acquired, err := indexer.AcquireLock(ctx, "test-key-6", "owner-9", 10*time.Second)
		require.NoError(t, err)
		assert.True(t, acquired)

		// Try to release the lock as a different owner
		err = indexer.ReleaseLock(ctx, "test-key-6", "owner-10")
		require.NoError(t, err)

		// Verify lock still exists
		var lock DistributedLock
		result := indexer.db.Where("key = ?", "test-key-6").First(&lock)
		require.NoError(t, result.Error)
		assert.Equal(t, "owner-9", lock.Owner)
	})
}

func TestCleanupExpiredLocks(t *testing.T) {
	ctx := context.Background()
	indexer := setupTestDB(t)

	// Create some expired locks
	for i := range 5 {
		expiredLock := &DistributedLock{
			Key:       "expired-key-" + string(rune(i+'0')),
			Owner:     "owner-expired",
			ExpiresAt: time.Now().Add(-1 * time.Second),
		}
		result := indexer.db.Create(expiredLock)
		require.NoError(t, result.Error)
	}

	// Create some valid locks
	for i := range 3 {
		validLock := &DistributedLock{
			Key:       "valid-key-" + string(rune(i+'0')),
			Owner:     "owner-valid",
			ExpiresAt: time.Now().Add(10 * time.Second),
		}
		result := indexer.db.Create(validLock)
		require.NoError(t, result.Error)
	}

	// Count total locks before cleanup
	var countBefore int64

	indexer.db.Model(&DistributedLock{}).Count(&countBefore)

	assert.Equal(t, int64(8), countBefore)

	// Run cleanup
	err := indexer.cleanupExpiredLocks(ctx)
	require.NoError(t, err)

	// Count remaining locks after cleanup
	var countAfter int64

	indexer.db.Model(&DistributedLock{}).Count(&countAfter)

	assert.Equal(t, int64(3), countAfter)

	// Verify only valid locks remain
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

	// Test concurrent lock acquisition by different owners
	t.Run("concurrent acquisition by different owners", func(t *testing.T) {
		// First owner acquires the lock
		acquired1, err := indexer.AcquireLock(ctx, "concurrent-key", "owner-11", 10*time.Second)
		require.NoError(t, err)
		assert.True(t, acquired1)

		// Second owner tries to acquire the same lock
		acquired2, err := indexer.AcquireLock(ctx, "concurrent-key", "owner-12", 10*time.Second)
		require.Error(t, err)
		assert.False(t, acquired2)

		// First owner releases the lock
		err = indexer.ReleaseLock(ctx, "concurrent-key", "owner-11")
		require.NoError(t, err)

		// Now second owner should be able to acquire the lock
		acquired3, err := indexer.AcquireLock(ctx, "concurrent-key", "owner-12", 10*time.Second)
		require.NoError(t, err)
		assert.True(t, acquired3)
	})
}
