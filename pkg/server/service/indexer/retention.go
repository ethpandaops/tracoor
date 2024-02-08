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
		"beacon_state":          e.config.Retention.BeaconStates.Duration,
		"execution_block_trace": e.config.Retention.ExecutionBlockTraces.Duration,
		"execution_bad_block":   e.config.Retention.ExecutionBadBlocks.Duration,
	}).Info("Starting retention watcher")

	for {
		if err := e.purgeOldBeaconStates(ctx); err != nil {
			e.log.WithError(err).Error("Failed to delete old beacon states")
		}

		if err := e.purgeOldExecutionTraces(ctx); err != nil {
			e.log.WithError(err).Error("Failed to delete old execution traces")
		}

		if err := e.purgeOldExecutionBadBlocks(ctx); err != nil {
			e.log.WithError(err).Error("Failed to delete old execution bad blocks")
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
				e.log.WithField("state_id", state.ID).Warn("Beacon state not found in store")
			} else {
				e.log.WithError(err).WithField("state_id", state.ID).Error("Failed to delete beacon state from store, will retry next time")

				continue
			}
		}

		err := e.db.DeleteBeaconState(ctx, state.ID)
		if err != nil {
			e.log.WithError(err).WithField("state_id", state.ID).Error("Failed to delete beacon state")

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
				e.log.WithField("trace_id", trace.ID).Warn("Execution block trace not found in store")
			} else {
				e.log.WithError(err).WithField("trace_id", trace.ID).Error("Failed to delete execution block trace from store, will retry next time")

				continue
			}
		}

		err := e.db.DeleteExecutionBlockTrace(ctx, trace.ID)
		if err != nil {
			e.log.WithError(err).WithField("trace_id", trace.ID).Error("Failed to delete execution block trace")

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

func (e *Indexer) purgeOldExecutionBadBlocks(ctx context.Context) error {
	before := time.Now().Add(-e.config.Retention.ExecutionBadBlocks.Duration)

	filter := &persistence.ExecutionBadBlockFilter{
		Before: &before,
	}

	blocks, err := e.db.ListExecutionBadBlock(ctx, filter, &persistence.PaginationCursor{Limit: 10000, Offset: 1, OrderBy: "fetched_at ASC"})
	if err != nil {
		return err
	}

	e.log.WithField("before", before).Debugf("Purging %d old execution bad blocks", len(blocks))

	for _, block := range blocks {
		// Delete from the store first
		if err := e.store.DeleteExecutionBadBlock(ctx, block.Location); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				e.log.WithField("block_id", block.ID).Warn("Execution bad block not found in store")
			} else {
				e.log.WithError(err).WithField("block_id", block.ID).Error("Failed to delete execution bad block from store, will retry next time")

				continue
			}
		}

		err := e.db.DeleteExecutionBadBlock(ctx, block.ID)
		if err != nil {
			e.log.WithError(err).WithField("block_id", block.ID).Error("Failed to delete execution bad block")

			continue
		}

		e.log.WithFields(
			logrus.Fields{
				"node":       block.Node,
				"network":    block.Network,
				"block_hash": block.BlockHash,
				"id":         block.ID,
			},
		).Debug("Deleted execution block trace")
	}

	return nil
}
