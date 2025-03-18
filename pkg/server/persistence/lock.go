package persistence

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// DistributedLock represents a lock in the database.
type DistributedLock struct {
	gorm.Model
	Key       string `gorm:"uniqueIndex"`
	Owner     string
	ExpiresAt time.Time
}

// AcquireLock attempts to acquire a lock with the given key.
// It returns true if the lock was acquired, false otherwise.
func (i *Indexer) AcquireLock(ctx context.Context, key, owner string, ttl time.Duration) (bool, error) {
	// First, clean up expired locks
	if err := i.cleanupExpiredLocks(ctx); err != nil {
		return false, errors.Wrap(err, "failed to cleanup expired locks")
	}

	// Try to acquire the lock
	expiresAt := time.Now().Add(ttl)
	lock := &DistributedLock{
		Key:       key,
		Owner:     owner,
		ExpiresAt: expiresAt,
	}

	// Use a transaction to ensure atomicity
	err := i.db.Transaction(func(tx *gorm.DB) error {
		// Check if the lock exists (including soft-deleted records)
		var existingLock DistributedLock

		result := tx.Unscoped().Where("key = ?", key).First(&existingLock)
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				// Lock doesn't exist, create it
				if err := tx.Create(lock).Error; err != nil {
					return errors.Wrap(err, "failed to create lock")
				}

				return nil
			}

			return errors.Wrap(result.Error, "failed to check if lock exists")
		}

		// If the lock was soft-deleted, permanently delete it and create a new one
		if existingLock.DeletedAt.Valid {
			if err := tx.Unscoped().Delete(&existingLock).Error; err != nil {
				return errors.Wrap(err, "failed to delete soft-deleted lock")
			}

			if err := tx.Create(lock).Error; err != nil {
				return errors.Wrap(err, "failed to create lock after deleting soft-deleted lock")
			}

			return nil
		}

		// Lock exists, check if it's expired
		if existingLock.ExpiresAt.Before(time.Now()) {
			// Lock is expired, update it
			existingLock.Owner = owner
			existingLock.ExpiresAt = expiresAt

			if err := tx.Save(&existingLock).Error; err != nil {
				return errors.Wrap(err, "failed to update expired lock")
			}

			return nil
		}

		// Lock exists and is not expired, check if we own it
		if existingLock.Owner == owner {
			// We own the lock, extend it
			existingLock.ExpiresAt = expiresAt
			if err := tx.Save(&existingLock).Error; err != nil {
				return errors.Wrap(err, "failed to extend lock")
			}

			return nil
		}

		// Lock is owned by someone else
		return fmt.Errorf("lock is owned by %s until %s", existingLock.Owner, existingLock.ExpiresAt)
	})

	if err != nil {
		i.log.WithFields(logrus.Fields{
			"key":   key,
			"owner": owner,
			"error": err.Error(),
		}).Debug("Failed to acquire lock")

		return false, errors.Wrap(err, "failed to acquire lock")
	}

	i.log.WithFields(logrus.Fields{
		"key":   key,
		"owner": owner,
		"ttl":   ttl,
	}).Debug("Acquired lock")

	return true, nil
}

// ReleaseLock releases a lock with the given key if it's owned by the given owner.
func (i *Indexer) ReleaseLock(ctx context.Context, key, owner string) error {
	result := i.db.Where("key = ? AND owner = ?", key, owner).Delete(&DistributedLock{})
	if result.Error != nil {
		return errors.Wrap(result.Error, "failed to release lock")
	}

	if result.RowsAffected == 0 {
		i.log.WithFields(logrus.Fields{
			"key":   key,
			"owner": owner,
		}).Debug("Lock not found or not owned by the given owner")

		return nil
	}

	i.log.WithFields(logrus.Fields{
		"key":   key,
		"owner": owner,
	}).Debug("Released lock")

	return nil
}

// cleanupExpiredLocks removes all expired locks from the database.
func (i *Indexer) cleanupExpiredLocks(_ context.Context) error {
	// Use Unscoped() to permanently delete the records instead of soft-deleting them
	result := i.db.Unscoped().Where("expires_at < ?", time.Now()).Delete(&DistributedLock{})
	if result.Error != nil {
		return errors.Wrap(result.Error, "failed to cleanup expired locks")
	}

	if result.RowsAffected > 0 {
		i.log.WithField("count", result.RowsAffected).Debug("Cleaned up expired locks")
	}

	return nil
}
