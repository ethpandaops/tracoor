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

		if err := e.purgeOldExecutionTraces(ctx); err != nil {
			e.log.WithError(err).Error("Failed to delete old execution traces")
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

func (e *Indexer) purgeOldExecutionTraces(ctx context.Context) error {
	before := time.Now().Add(-e.config.Retention.ExecutionBlockTraces.Duration)

	filter := &persistence.ExecutionBlockTraceFilter{
		Before: &before,
	}

	traces, err := e.db.ListExecutionBlockTrace(ctx, filter, &persistence.PaginationCursor{Limit: 10000, Offset: 1, OrderBy: "fetched_at ASC"})
	if err != nil {
		return err
	}

	e.log.WithField("before", before).Debugf("Purging %d old execution block traces", len(traces))

	for _, trace := range traces {
		// Delete from the store first
		if err := e.store.DeleteExecutionBlockTrace(ctx, trace.Location); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				e.log.WithField("trace", trace).Warn("Execution block trace not found in store")
			} else {
				e.log.WithError(err).WithField("trace", trace).Error("Failed to delete execution block trace from store, will retry next time")

				continue
			}
		}

		err := e.db.DeleteExecutionBlockTrace(ctx, trace.ID)
		if err != nil {
			e.log.WithError(err).WithField("trace", trace).Error("Failed to delete execution block trace")

			continue
		}

		e.log.WithFields(
			logrus.Fields{
				"node":         trace.Node,
				"network":      trace.Network,
				"block_number": trace.BlockNumber,
				"id":           trace.ID,
			},
		).Debug("Deleted execution block trace")
	}

	return nil
}
