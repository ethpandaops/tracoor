package indexer

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/sirupsen/logrus"

	"github.com/ethpandaops/tracoor/pkg/server/persistence"
	"github.com/ethpandaops/tracoor/pkg/store"
)

// PermanentStoreResult is what became of a queued block. Archived is true only when the
// block is known to be present in the permanent store; a full queue, a lost lock or a
// worker error all report false, so a waiter can hold its row back rather than purge an
// object the archive never received.
type PermanentStoreResult struct {
	Archived bool
}

// PermanentStoreBlock contains the minimal information needed to identify an archivable
// artifact. Kind distinguishes the two artifacts that share a block root from gloas on -
// the beacon block and its execution payload envelope - and an empty Kind means
// persistence.KindBeaconBlock, which is what every caller meant before envelopes existed.
type PermanentStoreBlock struct {
	Kind      string
	Location  string
	BlockRoot string
	Network   string
	Slot      phase0.Slot
	// ProcessedChan, when non-nil, receives exactly one result and is then closed. It must
	// be buffered: the send never blocks, so on an unbuffered channel a waiter that has
	// already given up costs the result its value and the close reads as not archived.
	ProcessedChan chan PermanentStoreResult
}

// PermanentStore ensures that at least one copy of each block per network is retained
// by copying it to a permanent location in the store.
type PermanentStore struct {
	log     logrus.FieldLogger
	store   store.Store
	db      *persistence.Indexer
	queue   chan PermanentStoreBlock
	cache   *lru.Cache[string, bool]
	enabled map[string]bool
	stopped bool
	nodeID  string
}

type PermanentStoreConfig struct {
	Blocks BlockConfig `yaml:"blocks"`
	// ExecutionPayloadEnvelopes is its own switch, but archiving gloas blocks without it
	// keeps only half of each slot: the payload lives in the envelope from that fork on.
	ExecutionPayloadEnvelopes BlockConfig `yaml:"executionPayloadEnvelopes"`
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
		log:   log.WithField("component", "permanent_store"),
		store: st,
		db:    db,
		queue: make(chan PermanentStoreBlock, 5000),
		cache: cache,
		enabled: map[string]bool{
			persistence.KindBeaconBlock:              conf.Blocks.Enabled,
			persistence.KindExecutionPayloadEnvelope: conf.ExecutionPayloadEnvelopes.Enabled,
		},
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

// IsEnabledFor reports whether this kind of artifact is archived.
func (p *PermanentStore) IsEnabledFor(kind string) bool {
	return p.enabled[kind]
}

// IsEnabled reports whether the permanent store archives anything at all.
func (p *PermanentStore) IsEnabled() bool {
	for _, on := range p.enabled {
		if on {
			return true
		}
	}

	return false
}

// QueueBlock adds a block to the queue for processing. Every path that does not hand the block
// to a worker signals ProcessedChan itself — with a not-archived result — so a caller waiting
// on it is never stranded and never mistakes a dropped block for an archived one.
func (p *PermanentStore) QueueBlock(block PermanentStoreBlock) {
	// Check if the permanent store is enabled for this kind of artifact
	if !p.IsEnabledFor(block.kind()) {
		signalProcessed(block, false)

		return
	}

	if p.stopped {
		signalProcessed(block, false)

		return
	}

	select {
	case p.queue <- block:
		p.log.WithFields(logrus.Fields{
			KeyBlockRoot: block.BlockRoot,
			KeyNetwork:   block.Network,
			KeyLocation:  block.Location,
		}).Debug("Queued block for permanent storage")
	default:
		signalProcessed(block, false)

		p.log.WithFields(logrus.Fields{
			KeyBlockRoot: block.BlockRoot,
			KeyNetwork:   block.Network,
			KeyLocation:  block.Location,
		}).Warn("Failed to queue block for permanent storage, queue is full")
	}
}

// signalProcessed delivers a block's result exactly once. The send never blocks — a waiter
// that has already timed out must not strand a worker — and the close still wakes any
// receiver the send could not reach, reading as a zero (not archived) result.
func signalProcessed(block PermanentStoreBlock, archived bool) {
	if block.ProcessedChan == nil {
		return
	}

	select {
	case block.ProcessedChan <- PermanentStoreResult{Archived: archived}:
	default:
	}

	close(block.ProcessedChan)
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
				signalProcessed(block, false)

				continue
			}

			if err := p.processBlock(ctx, block); err != nil {
				p.log.WithError(err).WithFields(logrus.Fields{
					KeyBlockRoot: block.BlockRoot,
					KeyNetwork:   block.Network,
					KeyLocation:  block.Location,
				}).Error("Failed to process block for permanent storage")
			}
		}
	}
}

// processBlock processes a single block.
func (p *PermanentStore) processBlock(ctx context.Context, block PermanentStoreBlock) error {
	// Create a cache key for this artifact. The kind is in the key because a gloas block
	// and its envelope share a block root: without it, archiving one would mark the other
	// as already done and the object would be purged unarchived.
	cacheKey := fmt.Sprintf("%s:%s:%s", block.kind(), block.Network, block.BlockRoot)

	// The result is delivered on the way out. archived flips to true only once the block is
	// known to be in the permanent store, so a waiter never releases a row on a lost lock or
	// a failed copy.
	archived := false

	defer func() { signalProcessed(block, archived) }()

	// Check if we've already processed this block
	if _, ok := p.cache.Get(cacheKey); ok {
		p.log.WithFields(logrus.Fields{
			KeyBlockRoot: block.BlockRoot,
			KeyNetwork:   block.Network,
		}).Debug("Block already processed (cache hit)")

		archived = true

		return nil
	}

	// Create a lock key for this block
	lockKey := fmt.Sprintf("permanent_store:%s", cacheKey)

	// Try to acquire a distributed lock with retries. An unacquired lock is not an error:
	// another instance holds it and is processing this block.
	var acquired bool

	retryInterval := 200 * time.Millisecond
	maxRetryDuration := 35 * time.Second
	startTime := time.Now()

	for time.Since(startTime) < maxRetryDuration {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			a, err := p.db.AcquireLock(ctx, lockKey, p.nodeID, 30*time.Second)
			if err != nil {
				return fmt.Errorf("failed to acquire lock: %w", err)
			}

			if a {
				acquired = true

				break
			}

			p.log.WithFields(logrus.Fields{
				KeyBlockRoot: block.BlockRoot,
				KeyNetwork:   block.Network,
				KeyLockKey:   lockKey,
				"elapsed":    time.Since(startTime).String(),
			}).Debug("Failed to acquire lock, retrying...")

			time.Sleep(retryInterval)
		}

		if acquired {
			break
		}
	}

	if !acquired {
		p.log.WithFields(logrus.Fields{
			KeyBlockRoot: block.BlockRoot,
			KeyNetwork:   block.Network,
			KeyLockKey:   lockKey,
		}).Debug("Failed to acquire lock after maximum retry duration")

		return nil
	}

	defer func() {
		if lerr := p.db.ReleaseLock(ctx, lockKey, p.nodeID); lerr != nil {
			p.log.WithError(lerr).WithFields(logrus.Fields{
				KeyBlockRoot: block.BlockRoot,
				KeyNetwork:   block.Network,
				KeyLockKey:   lockKey,
			}).Error("Failed to release lock")
		}
	}()

	// Check again after acquiring the lock
	if _, ok := p.cache.Get(cacheKey); ok {
		p.log.WithFields(logrus.Fields{
			KeyBlockRoot: block.BlockRoot,
			KeyNetwork:   block.Network,
		}).Debug("Block already processed (cache hit after lock)")

		archived = true

		return nil
	}

	// Check if block is already recorded in database before checking the store
	permanentBlock, err := p.db.GetPermanentBlockByBlockRoot(ctx, block.kind(), block.BlockRoot, block.Network)
	if err != nil && !errors.Is(err, persistence.ErrPermanentBlockNotFound) {
		p.log.WithError(err).WithFields(logrus.Fields{
			KeyBlockRoot: block.BlockRoot,
			KeyNetwork:   block.Network,
			KeyKind:      block.kind(),
		}).Error("Failed to check if block is already recorded in database")
	} else if permanentBlock != nil {
		p.log.WithFields(logrus.Fields{
			KeyBlockRoot: block.BlockRoot,
			KeyNetwork:   block.Network,
		}).Debug("Block already recorded in database")

		// Add to cache to avoid future checks
		p.cache.Add(cacheKey, true)

		archived = true

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
			KeyBlockRoot: block.BlockRoot,
			KeyNetwork:   block.Network,
			KeyLocation:  permanentLocation,
		}).Debug("Block already exists in permanent location")

		// Add to cache to avoid future checks
		p.cache.Add(cacheKey, true)

		archived = true

		// Ensure the block is recorded in the database even if it already exists in storage
		if perr := p.recordPermanentBlock(ctx, block); perr != nil {
			p.log.WithError(perr).WithFields(logrus.Fields{
				KeyBlockRoot: block.BlockRoot,
				KeyNetwork:   block.Network,
				KeySlot:      block.Slot,
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
		KeyBlockRoot: block.BlockRoot,
		KeyNetwork:   block.Network,
		"from":       block.Location,
		"to":         permanentLocation,
	}).Info("Copied block to permanent location")

	// The object is in the permanent location; a failure to record it below is recoverable
	// and must not hold the source row back.
	archived = true

	// Record the block in the database
	if perr := p.recordPermanentBlock(ctx, block); perr != nil {
		p.log.WithError(perr).WithFields(logrus.Fields{
			KeyBlockRoot: block.BlockRoot,
			KeyNetwork:   block.Network,
			KeySlot:      block.Slot,
		}).Error("Failed to record permanent block in database")
	}

	// Add to cache to avoid future checks
	p.cache.Add(cacheKey, true)

	return nil
}

// recordPermanentBlock records the block in the PermanentBlock table.
func (p *PermanentStore) recordPermanentBlock(ctx context.Context, block PermanentStoreBlock) error {
	// Record the block directly since we already checked earlier if it exists
	return p.db.InsertPermanentBlock(ctx, &persistence.PermanentBlock{
		//nolint:gosec // At the mercy of the database
		Slot:      int64(block.Slot),
		Kind:      block.kind(),
		BlockRoot: block.BlockRoot,
		Network:   block.Network,
	})
}

// GetPermanentLocation returns the permanent location for an artifact. Beacon blocks keep
// the layout they have always had, so archives written before envelopes existed still
// resolve; every other kind gets its own subdirectory, which is also what keeps a block and
// its envelope - same root, same extension - from being the same object.
func (p *PermanentStore) GetPermanentLocation(block PermanentStoreBlock) string {
	// Extract the file extension from the source location
	extension := filepath.Ext(block.Location)

	if block.kind() == persistence.KindBeaconBlock {
		return filepath.Join("permanent", block.Network, block.BlockRoot+extension)
	}

	return filepath.Join("permanent", block.Network, block.kind(), block.BlockRoot+extension)
}

// kind is the artifact's kind, defaulting to a beacon block: every caller that predates
// envelopes queues blocks and sets no kind.
func (b PermanentStoreBlock) kind() string {
	if b.Kind == "" {
		return persistence.KindBeaconBlock
	}

	return b.Kind
}
