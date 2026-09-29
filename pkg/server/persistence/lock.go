package persistence

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	logKeyLock  = "key"
	logKeyOwner = "owner"
)

// DistributedLock is acquired by a single atomic conditional upsert. There is no
// soft delete: a released lock is a deleted row.
type DistributedLock struct {
	Key       string    `gorm:"primaryKey"`
	Owner     string    `gorm:"not null;default:''"`
	ExpiresAt time.Time `gorm:"not null;index:ix_distributed_locks_expires_at"`
}

// BeforeSave keeps the lease in UTC, so a lock taken by one process is honoured by another
// whatever zone either of them runs in.
func (l *DistributedLock) BeforeSave(*gorm.DB) error {
	l.ExpiresAt = utcBound(l.ExpiresAt)

	return nil
}

// AcquireLock attempts to acquire the lock with the given key, taking it over when the
// current holder's lease has expired and extending it when the caller already owns it.
// A lock held by someone else is reported as (false, nil), not an error.
func (i *Indexer) AcquireLock(ctx context.Context, key, owner string, ttl time.Duration) (bool, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(ttl)

	result := i.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "key"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"owner":      owner,
			"expires_at": expiresAt,
		}),
		Where: clause.Where{Exprs: []clause.Expression{
			gorm.Expr("distributed_locks.expires_at < ? OR distributed_locks.owner = ?", now, owner),
		}},
	}).Create(&DistributedLock{
		Key:       key,
		Owner:     owner,
		ExpiresAt: expiresAt,
	})
	if result.Error != nil {
		return false, errors.Wrap(result.Error, "failed to acquire lock")
	}

	if result.RowsAffected == 0 {
		i.log.WithFields(logrus.Fields{
			logKeyLock:  key,
			logKeyOwner: owner,
		}).Debug("Lock is held by another owner")

		return false, nil
	}

	i.log.WithFields(logrus.Fields{
		logKeyLock:  key,
		logKeyOwner: owner,
		"ttl":       ttl,
	}).Debug("Acquired lock")

	return true, nil
}

// ReleaseLock releases a lock with the given key if it's owned by the given owner.
func (i *Indexer) ReleaseLock(ctx context.Context, key, owner string) error {
	result := i.db.WithContext(ctx).Where("key = ? AND owner = ?", key, owner).Delete(&DistributedLock{})
	if result.Error != nil {
		return errors.Wrap(result.Error, "failed to release lock")
	}

	if result.RowsAffected == 0 {
		i.log.WithFields(logrus.Fields{
			logKeyLock:  key,
			logKeyOwner: owner,
		}).Debug("Lock not found or not owned by the given owner")

		return nil
	}

	i.log.WithFields(logrus.Fields{
		logKeyLock:  key,
		logKeyOwner: owner,
	}).Debug("Released lock")

	return nil
}

// cleanupExpiredLocks removes all expired locks from the database. Acquisition does not
// depend on it; it only keeps abandoned rows from accumulating.
func (i *Indexer) cleanupExpiredLocks(ctx context.Context) error {
	result := i.db.WithContext(ctx).Where("expires_at < ?", time.Now().UTC()).Delete(&DistributedLock{})
	if result.Error != nil {
		return errors.Wrap(result.Error, "failed to cleanup expired locks")
	}

	if result.RowsAffected > 0 {
		i.log.WithField("count", result.RowsAffected).Debug("Cleaned up expired locks")
	}

	return nil
}
