package indexer

import (
	"context"
	"errors"
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/tracoor/pkg/server/persistence"
	"github.com/ethpandaops/tracoor/pkg/store"
	"github.com/sirupsen/logrus"
)

func (i *Indexer) startRetentionWatchers(ctx context.Context) {
	i.log.WithFields(logrus.Fields{
		"beacon_state":          i.config.Retention.BeaconStates.Duration,
		"beacon_block":          i.config.Retention.BeaconBlocks.Duration,
		"beacon_bad_block":      i.config.Retention.BeaconBadBlocks.Duration,
		"beacon_bad_blob":       i.config.Retention.BeaconBadBlobs.Duration,
		"execution_block_trace": i.config.Retention.ExecutionBlockTraces.Duration,
		"execution_bad_block":   i.config.Retention.ExecutionBadBlocks.Duration,
	}).Info("Starting retention watcher")

	for {
		if err := i.purgeOldBeaconStates(ctx); err != nil {
			i.log.WithError(err).Error("Failed to delete old beacon states")
		}

		if err := i.purgeOldBeaconBlocks(ctx); err != nil {
			i.log.WithError(err).Error("Failed to delete old beacon blocks")
		}

		if err := i.purgeOldBeaconBadBlocks(ctx); err != nil {
			i.log.WithError(err).Error("Failed to delete old beacon bad blocks")
		}

		if err := i.purgeOldBeaconBadBlobs(ctx); err != nil {
			i.log.WithError(err).Error("Failed to delete old beacon bad blobs")
		}

		if err := i.purgeOldExecutionTraces(ctx); err != nil {
			i.log.WithError(err).Error("Failed to delete old execution traces")
		}

		if err := i.purgeOldExecutionBadBlocks(ctx); err != nil {
			i.log.WithError(err).Error("Failed to delete old execution bad blocks")
		}

		select {
		case <-time.After(1 * time.Minute):
		case <-ctx.Done():
			return
		}
	}
}

func (i *Indexer) purgeOldBeaconStates(ctx context.Context) error {
	before := time.Now().Add(-i.config.Retention.BeaconStates.Duration)

	filter := &persistence.BeaconStateFilter{
		Before: &before,
	}

	states, err := i.db.ListBeaconState(ctx, filter, &persistence.PaginationCursor{Limit: 10000, Offset: 0, OrderBy: "fetched_at ASC"})
	if err != nil {
		return err
	}

	i.log.WithField("before", before).Debugf("Purging %d old beacon states", len(states))

	for _, state := range states {
		// Delete from the store first
		if err := i.store.DeleteBeaconState(ctx, state.Location); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				i.log.WithField("state_id", state.ID).Warn("Beacon state not found in store")
			} else {
				i.log.WithError(err).WithField("state_id", state.ID).Error("Failed to delete beacon state from store, will retry next time")

				continue
			}
		}

		err := i.db.DeleteBeaconState(ctx, state.ID)
		if err != nil {
			i.log.WithError(err).WithField("state_id", state.ID).Error("Failed to delete beacon state")

			continue
		}

		i.log.WithFields(
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

func (i *Indexer) purgeOldBeaconBlocks(ctx context.Context) error {
	before := time.Now().Add(-i.config.Retention.BeaconBlocks.Duration)

	filter := &persistence.BeaconBlockFilter{
		Before: &before,
	}

	blocks, err := i.db.ListBeaconBlock(ctx, filter, &persistence.PaginationCursor{Limit: 10000, Offset: 0, OrderBy: "fetched_at ASC"})
	if err != nil {
		return err
	}

	i.log.WithField("before", before).Debugf("Purging %d old beacon blocks", len(blocks))

	for _, block := range blocks {
		// Check if the block needs to be processed by the permanent store
		b := PermanentStoreBlock{
			Location:      block.Location,
			BlockRoot:     block.BlockRoot,
			Network:       block.Network,
			ProcessedChan: make(chan struct{}),
			//nolint:gosec // This is a valid conversion
			Slot: phase0.Slot(block.Slot),
		}

		i.permanentStore.QueueBlock(b)

		// Wait for the block to be processed
		<-b.ProcessedChan

		// Delete from the store first
		if err := i.store.DeleteBeaconBlock(ctx, block.Location); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				i.log.WithField("block_id", block.ID).Warn("Beacon block not found in store")
			} else {
				i.log.WithError(err).WithField("block_id", block.ID).Error("Failed to delete beacon block from store, will retry next time")

				continue
			}
		}

		err := i.db.DeleteBeaconBlock(ctx, block.ID)
		if err != nil {
			i.log.WithError(err).WithField("block_id", block.ID).Error("Failed to delete beacon block")

			continue
		}

		i.log.WithFields(
			logrus.Fields{
				"node":    block.Node,
				"network": block.Network,
				"slot":    block.Slot,
				"id":      block.ID,
			},
		).Debug("Deleted beacon block")
	}

	return nil
}

func (i *Indexer) purgeOldBeaconBadBlocks(ctx context.Context) error {
	before := time.Now().Add(-i.config.Retention.BeaconBadBlocks.Duration)

	filter := &persistence.BeaconBadBlockFilter{
		Before: &before,
	}

	blocks, err := i.db.ListBeaconBadBlock(ctx, filter, &persistence.PaginationCursor{Limit: 10000, Offset: 0, OrderBy: "fetched_at ASC"})
	if err != nil {
		return err
	}

	i.log.WithField("before", before).Debugf("Purging %d old beacon bad blocks", len(blocks))

	for _, block := range blocks {
		// Delete from the store first
		if err := i.store.DeleteBeaconBadBlock(ctx, block.Location); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				i.log.WithField("block_id", block.ID).Warn("Beacon bad block not found in store")
			} else {
				i.log.WithError(err).WithField("block_id", block.ID).Error("Failed to delete beacon bad block from store, will retry next time")

				continue
			}
		}

		err := i.db.DeleteBeaconBadBlock(ctx, block.ID)
		if err != nil {
			i.log.WithError(err).WithField("block_id", block.ID).Error("Failed to delete beacon bad block")

			continue
		}

		i.log.WithFields(
			logrus.Fields{
				"node":    block.Node,
				"network": block.Network,
				"slot":    block.Slot,
				"id":      block.ID,
			},
		).Debug("Deleted beacon bad block")
	}

	return nil
}

func (i *Indexer) purgeOldBeaconBadBlobs(ctx context.Context) error {
	before := time.Now().Add(-i.config.Retention.BeaconBadBlobs.Duration)

	filter := &persistence.BeaconBadBlobFilter{
		Before: &before,
	}

	blobs, err := i.db.ListBeaconBadBlob(ctx, filter, &persistence.PaginationCursor{Limit: 10000, Offset: 0, OrderBy: "fetched_at ASC"})
	if err != nil {
		return err
	}

	i.log.WithField("before", before).Debugf("Purging %d old beacon bad blobs", len(blobs))

	for _, blob := range blobs {
		// Delete from the store first
		if err := i.store.DeleteBeaconBadBlob(ctx, blob.Location); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				i.log.WithField("blob_id", blob.ID).Warn("Beacon bad blob not found in store")
			} else {
				i.log.WithError(err).WithField("blob_id", blob.ID).Error("Failed to delete beacon bad blob from store, will retry next time")

				continue
			}
		}

		err := i.db.DeleteBeaconBadBlob(ctx, blob.ID)
		if err != nil {
			i.log.WithError(err).WithField("blob_id", blob.ID).Error("Failed to delete beacon bad blob")

			continue
		}

		i.log.WithFields(
			logrus.Fields{
				"node":    blob.Node,
				"network": blob.Network,
				"slot":    blob.Slot,
				"index":   blob.Index,
				"id":      blob.ID,
			},
		).Debug("Deleted beacon bad blob")
	}

	return nil
}

func (i *Indexer) purgeOldExecutionTraces(ctx context.Context) error {
	before := time.Now().Add(-i.config.Retention.ExecutionBlockTraces.Duration)

	filter := &persistence.ExecutionBlockTraceFilter{
		Before: &before,
	}

	traces, err := i.db.ListExecutionBlockTrace(ctx, filter, &persistence.PaginationCursor{Limit: 10000, Offset: 0, OrderBy: "fetched_at ASC"})
	if err != nil {
		return err
	}

	i.log.WithField("before", before).Debugf("Purging %d old execution block traces", len(traces))

	for _, trace := range traces {
		// Delete from the store first
		if err := i.store.DeleteExecutionBlockTrace(ctx, trace.Location); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				i.log.WithField("trace_id", trace.ID).Warn("Execution block trace not found in store")
			} else {
				i.log.WithError(err).WithField("trace_id", trace.ID).Error("Failed to delete execution block trace from store, will retry next time")

				continue
			}
		}

		err := i.db.DeleteExecutionBlockTrace(ctx, trace.ID)
		if err != nil {
			i.log.WithError(err).WithField("trace_id", trace.ID).Error("Failed to delete execution block trace")

			continue
		}

		i.log.WithFields(
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

func (i *Indexer) purgeOldExecutionBadBlocks(ctx context.Context) error {
	before := time.Now().Add(-i.config.Retention.ExecutionBadBlocks.Duration)

	filter := &persistence.ExecutionBadBlockFilter{
		Before: &before,
	}

	blocks, err := i.db.ListExecutionBadBlock(ctx, filter, &persistence.PaginationCursor{Limit: 10000, Offset: 0, OrderBy: "fetched_at ASC"})
	if err != nil {
		return err
	}

	i.log.WithField("before", before).Debugf("Purging %d old execution bad blocks", len(blocks))

	for _, block := range blocks {
		// Delete from the store first
		if err := i.store.DeleteExecutionBadBlock(ctx, block.Location); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				i.log.WithField("block_id", block.ID).Warn("Execution bad block not found in store")
			} else {
				i.log.WithError(err).WithField("block_id", block.ID).Error("Failed to delete execution bad block from store, will retry next time")

				continue
			}
		}

		err := i.db.DeleteExecutionBadBlock(ctx, block.ID)
		if err != nil {
			i.log.WithError(err).WithField("block_id", block.ID).Error("Failed to delete execution bad block")

			continue
		}

		i.log.WithFields(
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
