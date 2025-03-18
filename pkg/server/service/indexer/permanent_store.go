package indexer

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/sirupsen/logrus"

	"github.com/ethpandaops/tracoor/pkg/server/persistence"
	"github.com/ethpandaops/tracoor/pkg/store"
)

// PermanentStoreBlock contains the minimal information needed to identify a block
type PermanentStoreBlock struct {
	Location      string
	BlockRoot     string
	Network       string
	Slot          phase0.Slot
	ProcessedChan chan struct{}
}

// PermanentStore ensures that at least one copy of each block per network is retained
// by copying it to a permanent location in the store.
type PermanentStore struct {
	log     logrus.FieldLogger
	store   store.Store
	db      *persistence.Indexer
	queue   chan PermanentStoreBlock
	cache   *lru.Cache[string, bool]
	enabled bool
	stopped bool
	nodeID  string
}

type PermanentStoreConfig struct {
	Blocks BlockConfig `yaml:"blocks"`
}

type BlockConfig struct {
	Enabled bool `yaml:"enabled" default:"false"`
}

// NewPermanentStore creates a new permanent store.
func NewPermanentStore(log logrus.FieldLogger, st store.Store, db *persistence.Indexer, nodeID string, conf *PermanentStoreConfig) (*PermanentStore, error) {
	cache, err := lru.New[string, bool](5000)
	if err != nil {
		return nil, fmt.Errorf("failed to create LRU cache: %w", err)
	}

	return &PermanentStore{
		log:     log.WithField("component", "permanent_store"),
		store:   st,
		db:      db,
		queue:   make(chan PermanentStoreBlock, 5000),
		cache:   cache,
		enabled: conf.Blocks.Enabled,
		nodeID:  nodeID,
		stopped: false,
	}, nil
}

// Start starts the permanent store.
func (p *PermanentStore) Start(ctx context.Context) error {
	p.log.Info("Starting permanent store")

	// Start multiple goroutines to process the queue
	for i := 0; i < 10; i++ {
		go p.processQueue(ctx)
	}

	return nil
}

// Stop stops the permanent store.
func (p *PermanentStore) Stop(ctx context.Context) error {
	p.log.Info("Stopping permanent store")

	// Set the stopped flag to prevent new blocks from being queued
	p.stopped = true

	// Wait until the queue is empty
	attempts := 0

	for len(p.queue) > 0 {
		p.log.WithField("remaining", len(p.queue)).Debug("Waiting for queue to empty")
		select {
		case <-ctx.Done():
			return ctx.Err()
		// Continue waiting
		case <-time.After(250 * time.Millisecond):
			attempts++

			p.log.WithField("attempts", attempts).Info("Waiting for queue to drain...")
		}
	}

	p.log.Debug("Queue is empty, permanent store stopped")

	return nil
}

func (p *PermanentStore) IsEnabled() bool {
	return p.enabled
}

// QueueBlock adds a block to the queue for processing.
func (p *PermanentStore) QueueBlock(block PermanentStoreBlock) {
	// Check if the permanent store is enabled
	if !p.IsEnabled() {
		return
	}

	if p.stopped {
		return
	}

	select {
	case p.queue <- block:
		p.log.WithFields(logrus.Fields{
			"block_root": block.BlockRoot,
			"network":    block.Network,
			"location":   block.Location,
		}).Debug("Queued block for permanent storage")
	default:
		p.log.WithFields(logrus.Fields{
			"block_root": block.BlockRoot,
			"network":    block.Network,
			"location":   block.Location,
		}).Warn("Failed to queue block for permanent storage, queue is full")
	}
}

// processQueue processes blocks from the queue.
func (p *PermanentStore) processQueue(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case block, ok := <-p.queue:
			// Check if the channel was closed
			if !ok {
				return
			}

			// Skip empty blocks
			if block.BlockRoot == "" || block.Network == "" || block.Location == "" {
				continue
			}

			if err := p.processBlock(ctx, block); err != nil {
				p.log.WithError(err).WithFields(logrus.Fields{
					"block_root": block.BlockRoot,
					"network":    block.Network,
					"location":   block.Location,
				}).Error("Failed to process block for permanent storage")
			}
		}
	}
}

// processBlock processes a single block.
func (p *PermanentStore) processBlock(ctx context.Context, block PermanentStoreBlock) error {
	// Create a cache key for this block
	cacheKey := fmt.Sprintf("%s:%s", block.Network, block.BlockRoot)

	// Close the processed channel so that the caller can wait for the block to be processed
	defer func() {
		if block.ProcessedChan != nil {
			close(block.ProcessedChan)
		}
	}()

	// Check if we've already processed this block
	if _, ok := p.cache.Get(cacheKey); ok {
		p.log.WithFields(logrus.Fields{
			"block_root": block.BlockRoot,
			"network":    block.Network,
		}).Debug("Block already processed (cache hit)")

		return nil
	}

	// Create a lock key for this block
	lockKey := fmt.Sprintf("permanent_store:%s", cacheKey)

	// Try to acquire a distributed lock with retries
	var acquired bool

	var err error

	retryInterval := 200 * time.Millisecond
	maxRetryDuration := 35 * time.Second
	startTime := time.Now()

	for time.Since(startTime) < maxRetryDuration {
		if acquired {
			break
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			acquired, err = p.db.AcquireLock(ctx, lockKey, p.nodeID, 30*time.Second)
			if err != nil {
				// If the error indicates someone else has the lock, retry
				if err.Error() != "" && time.Since(startTime) < maxRetryDuration {
					p.log.WithFields(logrus.Fields{
						"block_root": block.BlockRoot,
						"network":    block.Network,
						"lock_key":   lockKey,
						"error":      err.Error(),
						"elapsed":    time.Since(startTime).String(),
					}).Debug("Failed to acquire lock, retrying...")

					time.Sleep(retryInterval)

					continue
				}

				return fmt.Errorf("failed to acquire lock: %w", err)
			}

			if acquired {
				break
			}

			// If we couldn't acquire the lock but there's no error, retry
			if time.Since(startTime) < maxRetryDuration {
				p.log.WithFields(logrus.Fields{
					"block_root": block.BlockRoot,
					"network":    block.Network,
					"lock_key":   lockKey,
					"elapsed":    time.Since(startTime).String(),
				}).Debug("Failed to acquire lock, retrying...")

				time.Sleep(retryInterval)

				continue
			}

			p.log.WithFields(logrus.Fields{
				"block_root": block.BlockRoot,
				"network":    block.Network,
				"lock_key":   lockKey,
			}).Debug("Failed to acquire lock after retries, another instance is processing this block")

			return nil
		}
	}

	if !acquired {
		p.log.WithFields(logrus.Fields{
			"block_root": block.BlockRoot,
			"network":    block.Network,
			"lock_key":   lockKey,
		}).Debug("Failed to acquire lock after maximum retry duration")

		return nil
	}

	defer func() {
		if err := p.db.ReleaseLock(ctx, lockKey, p.nodeID); err != nil {
			p.log.WithError(err).WithFields(logrus.Fields{
				"block_root": block.BlockRoot,
				"network":    block.Network,
				"lock_key":   lockKey,
			}).Error("Failed to release lock")
		}
	}()

	// Check again after acquiring the lock
	if _, ok := p.cache.Get(cacheKey); ok {
		p.log.WithFields(logrus.Fields{
			"block_root": block.BlockRoot,
			"network":    block.Network,
		}).Debug("Block already processed (cache hit after lock)")

		return nil
	}

	// Determine the permanent location for this block
	permanentLocation := p.GetPermanentLocation(block)

	// Check if the block already exists in the permanent location
	exists, err := p.store.Exists(ctx, permanentLocation)
	if err != nil {
		return fmt.Errorf("failed to check if block exists in permanent location: %w", err)
	}

	if exists {
		p.log.WithFields(logrus.Fields{
			"block_root": block.BlockRoot,
			"network":    block.Network,
			"location":   permanentLocation,
		}).Debug("Block already exists in permanent location")

		// Add to cache to avoid future checks
		p.cache.Add(cacheKey, true)

		// Ensure the block is recorded in the database even if it already exists in storage
		if err := p.recordPermanentBlock(ctx, block); err != nil {
			p.log.WithError(err).WithFields(logrus.Fields{
				"block_root": block.BlockRoot,
				"network":    block.Network,
				"slot":       block.Slot,
			}).Error("Failed to record permanent block in database")
		}

		return nil
	}

	// Copy the block to the permanent location
	err = p.store.Copy(ctx, &store.CopyParams{
		Source:      block.Location,
		Destination: permanentLocation,
	})
	if err != nil {
		return fmt.Errorf("failed to copy block to permanent location: %w", err)
	}

	p.log.WithFields(logrus.Fields{
		"block_root": block.BlockRoot,
		"network":    block.Network,
		"from":       block.Location,
		"to":         permanentLocation,
	}).Info("Copied block to permanent location")

	// Record the block in the database
	if err := p.recordPermanentBlock(ctx, block); err != nil {
		p.log.WithError(err).WithFields(logrus.Fields{
			"block_root": block.BlockRoot,
			"network":    block.Network,
			"slot":       block.Slot,
		}).Error("Failed to record permanent block in database")
	}

	// Add to cache to avoid future checks
	p.cache.Add(cacheKey, true)

	return nil
}

// recordPermanentBlock records the block in the PermanentBlock table
func (p *PermanentStore) recordPermanentBlock(ctx context.Context, block PermanentStoreBlock) error {
	// Check if the block is already recorded
	permanentBlock, err := p.db.GetPermanentBlockByBlockRoot(ctx, block.BlockRoot, block.Network)
	if err != nil {
		return fmt.Errorf("failed to check if block is already recorded: %w", err)
	}

	// If the block is already recorded, we're done
	if permanentBlock != nil {
		return nil
	}

	// Record the block
	return p.db.InsertPermanentBlock(ctx, &persistence.PermanentBlock{
		//nolint:gosec // At the mercy of the database
		Slot:      int64(block.Slot),
		BlockRoot: block.BlockRoot,
		Network:   block.Network,
	})
}

// GetPermanentLocation returns the permanent location for a block.
func (p *PermanentStore) GetPermanentLocation(block PermanentStoreBlock) string {
	// Extract the file extension from the source location
	extension := filepath.Ext(block.Location)

	return filepath.Join("permanent", block.Network, block.BlockRoot+extension)
}
