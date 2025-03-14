package indexer

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

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
	}, nil
}

// Start starts the permanent store.
func (p *PermanentStore) Start(ctx context.Context) error {
	p.log.Info("Starting permanent store")

	go p.processQueue(ctx)

	return nil
}

// Stop stops the permanent store.
func (p *PermanentStore) Stop(ctx context.Context) error {
	p.log.Info("Stopping permanent store")

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
		case block := <-p.queue:
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

	// Try to acquire a distributed lock
	acquired, err := p.db.AcquireLock(ctx, lockKey, p.nodeID, 30*time.Second)
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}

	if !acquired {
		p.log.WithFields(logrus.Fields{
			"block_root": block.BlockRoot,
			"network":    block.Network,
			"lock_key":   lockKey,
		}).Debug("Failed to acquire lock, another instance is processing this block")

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
	permanentLocation := p.getPermanentLocation(block)

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

	// Add to cache to avoid future checks
	p.cache.Add(cacheKey, true)

	return nil
}

// getPermanentLocation returns the permanent location for a block.
func (p *PermanentStore) getPermanentLocation(block PermanentStoreBlock) string {
	// Extract the file extension from the source location
	extension := filepath.Ext(block.Location)

	return filepath.Join("permanent", block.Network, "blocks", block.BlockRoot+extension)
}
