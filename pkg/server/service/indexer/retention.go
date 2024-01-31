package indexer

import (
	"context"
	"errors"
	"time"

	"github.com/ethpandaops/tracoor/pkg/server/persistence"
	"github.com/ethpandaops/tracoor/pkg/store"
	"github.com/sirupsen/logrus"
)

func (e *Indexer) startRetentionWatchers(ctx context.Context) {
	e.log.WithFields(logrus.Fields{
		"beacon_state": e.config.Retention.BeaconStates.Duration,
	}).Info("Starting retention watcher")

	for {
		if err := e.purgeOldBeaconStates(ctx); err != nil {
			e.log.WithError(err).Error("Failed to delete old beacon states")
		}

		select {
		case <-time.After(1 * time.Minute):
		case <-ctx.Done():
			return
		}
	}
}

func (e *Indexer) purgeOldBeaconStates(ctx context.Context) error {
	before := time.Now().Add(-e.config.Retention.BeaconStates.Duration)

	filter := &persistence.BeaconStateFilter{
		Before: &before,
	}

	states, err := e.db.ListBeaconState(ctx, filter, &persistence.PaginationCursor{Limit: 10000, Offset: 1, OrderBy: "fetched_at ASC"})
	if err != nil {
		return err
	}

	e.log.WithField("before", before).Debugf("Purging %d old beacon states", len(states))

	for _, state := range states {
		// Delete from the store first
		if err := e.store.DeleteBeaconState(ctx, state.Location); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				e.log.WithField("state", state).Warn("Beacon state not found in store")
			} else {
				e.log.WithError(err).WithField("state", state).Error("Failed to delete beacon state from store, will retry next time")

				continue
			}
		}

		err := e.db.DeleteBeaconState(ctx, state.ID)
		if err != nil {
			e.log.WithError(err).WithField("state", state).Error("Failed to delete beacon state")

			continue
		}

		e.log.WithFields(
			logrus.Fields{
				"node":    state.Node,
				"network": state.Network,
				"slot":    state.Slot,
				"id":      state.ID,
			},
		).Debug("Deleted beacon state")
	}

	return nil
}
