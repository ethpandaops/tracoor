package indexer

import (
	"context"
	"testing"
	"time"

	"github.com/ethpandaops/tracoor/pkg/server/persistence"
	"github.com/ethpandaops/tracoor/pkg/store"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupMockIndexer creates a mock indexer
func setupMockIndexer(t *testing.T) *persistence.Indexer {
	t.Helper()

	indexer, _, err := persistence.NewMockIndexer()
	require.NoError(t, err)

	// Ensure the distributed lock table is migrated
	err = indexer.Start(context.Background())
	require.NoError(t, err)

	return indexer
}

// setupPermanentStore creates a new permanent store with mock dependencies
func setupPermanentStore(t *testing.T) (*PermanentStore, store.Store, func()) {
	t.Helper()

	bucket := "mybucket"
	ctx := context.Background()

	mockStore, cleanup, err := store.NewMockS3Store(ctx, bucket)
	require.NoError(t, err)

	clean := func() {
		if err := cleanup(); err != nil {
			t.Fatalf("Failed to clean up: %v", err)
		}
	}

	indexer := setupMockIndexer(t)

	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	nodeID := uuid.New().String()

	permanentStore, err := NewPermanentStore(log, mockStore, indexer, nodeID, &PermanentStoreConfig{
		Blocks: BlockConfig{
			Enabled: true,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, permanentStore)

	// Start the permanent store
	err = permanentStore.Start(context.Background())
	require.NoError(t, err)

	// Ensure the indexer is fully started and migrations are complete
	time.Sleep(100 * time.Millisecond)

	return permanentStore, mockStore, func() {
		clean()
	}
}

func TestPermanentStoreQueueAndProcess(t *testing.T) {
	ctx := context.Background()
	permanentStore, mockStore, cleanup := setupPermanentStore(t)

	defer cleanup()

	// Create a test block
	blockData := []byte("test block data")
	blockLocation := "test/location/block1.ssz"

	// Save the block to the mock store
	_, err := mockStore.SaveBeaconBlock(ctx, &store.SaveParams{
		Location: blockLocation,
		Data:     &blockData,
	})
	require.NoError(t, err)

	t.Run("queue and process block", func(t *testing.T) {
		// Queue a block for processing with a channel
		processChan := make(chan struct{})
		blockInfo := PermanentStoreBlock{
			Location:      blockLocation,
			BlockRoot:     "0x1234",
			Network:       "mainnet",
			Slot:          123,
			ProcessedChan: processChan,
		}

		permanentStore.QueueBlock(blockInfo)

		// Wait for the block to be processed
		select {
		case <-processChan:
			// Block was processed
		case <-time.After(500 * time.Millisecond):
			t.Fatal("Block processing timed out")
		}

		// Check if the block was copied to the permanent location
		permanentLocation := permanentStore.GetPermanentLocation(blockInfo)
		exists, err := mockStore.Exists(ctx, permanentLocation)
		require.NoError(t, err)
		assert.True(t, exists, "Block should be copied to permanent location")

		// Verify the block data
		data, err := mockStore.GetBeaconBlock(ctx, permanentLocation)
		require.NoError(t, err)
		assert.Equal(t, blockData, *data, "Block data should match")

		// Check that the permanent block was recorded in the database
		filter := &persistence.PermanentBlockFilter{}
		filter.AddBlockRoot(blockInfo.BlockRoot)
		filter.AddNetwork(blockInfo.Network)

		permanentBlocks, err := permanentStore.db.ListPermanentBlock(ctx, filter, &persistence.PaginationCursor{Limit: 1})
		require.NoError(t, err)
		assert.Len(t, permanentBlocks, 1, "Permanent block should be recorded in the database")
		assert.Equal(t, int64(blockInfo.Slot), permanentBlocks[0].Slot, "Slot should match")
		assert.Equal(t, blockInfo.BlockRoot, permanentBlocks[0].BlockRoot, "Block root should match")
		assert.Equal(t, blockInfo.Network, permanentBlocks[0].Network, "Network should match")
	})
}

func TestPermanentStoreProcessSameBlockTwice(t *testing.T) {
	ctx := context.Background()
	permanentStore, mockStore, cleanup := setupPermanentStore(t)

	defer cleanup()

	// Create a test block
	blockData := []byte("test block data")
	blockLocation := "test/location/block1.ssz"

	// Save the block to the mock store
	_, err := mockStore.SaveBeaconBlock(ctx, &store.SaveParams{
		Location: blockLocation,
		Data:     &blockData,
	})
	require.NoError(t, err)

	// Queue a block for processing with a channel
	processChan1 := make(chan struct{})
	blockInfo := PermanentStoreBlock{
		Location:      blockLocation,
		BlockRoot:     "0x1234",
		Network:       "mainnet",
		Slot:          123,
		ProcessedChan: processChan1,
	}

	// Process the block first time
	permanentStore.QueueBlock(blockInfo)

	// Wait for the block to be processed
	select {
	case <-processChan1:
		// Block was processed
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Block processing timed out")
	}

	// Get the permanent location
	permanentLocation := permanentStore.GetPermanentLocation(blockInfo)

	// Verify it exists
	exists, err := mockStore.Exists(ctx, permanentLocation)
	require.NoError(t, err)
	assert.True(t, exists, "Block should be copied to permanent location")

	// Check that the permanent block was recorded in the database
	filter := &persistence.PermanentBlockFilter{}
	filter.AddBlockRoot(blockInfo.BlockRoot)
	filter.AddNetwork(blockInfo.Network)

	permanentBlocks, err := permanentStore.db.ListPermanentBlock(ctx, filter, &persistence.PaginationCursor{Limit: 10})
	require.NoError(t, err)
	assert.Len(t, permanentBlocks, 1, "Permanent block should be recorded in the database")

	// Process the same block again with a new channel
	processChan2 := make(chan struct{})
	blockInfo2 := PermanentStoreBlock{
		Location:      blockLocation,
		BlockRoot:     "0x1234",
		Network:       "mainnet",
		Slot:          123,
		ProcessedChan: processChan2,
	}
	permanentStore.QueueBlock(blockInfo2)

	// Wait for the block to be processed again
	select {
	case <-processChan2:
		// Block was processed again
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Second block processing timed out")
	}

	// The block should still exist in the permanent location
	exists, err = mockStore.Exists(ctx, permanentLocation)
	require.NoError(t, err)
	assert.True(t, exists, "Block should still exist in permanent location")

	// Check that there is still only one permanent block record
	permanentBlocks, err = permanentStore.db.ListPermanentBlock(ctx, filter, &persistence.PaginationCursor{Limit: 10})
	require.NoError(t, err)
	assert.Len(t, permanentBlocks, 1, "There should still be only one permanent block record")
}

func TestPermanentStoreDifferentNetworks(t *testing.T) {
	ctx := context.Background()
	permanentStore, mockStore, cleanup := setupPermanentStore(t)

	defer cleanup()

	// Create test blocks
	blockData1 := []byte("test block data 1")
	blockLocation1 := "test/location/block1.ssz"

	// Save the first block to the mock store
	_, err := mockStore.SaveBeaconBlock(ctx, &store.SaveParams{
		Location: blockLocation1,
		Data:     &blockData1,
	})
	require.NoError(t, err)

	blockData2 := []byte("test block data 2")
	blockLocation2 := "test/location/block2.ssz"

	// Save the second block to the mock store
	_, err = mockStore.SaveBeaconBlock(ctx, &store.SaveParams{
		Location: blockLocation2,
		Data:     &blockData2,
	})
	require.NoError(t, err)

	// Process blocks sequentially to avoid race conditions with the distributed_locks table
	// First block
	blockInfo1 := PermanentStoreBlock{
		Location:      blockLocation1,
		BlockRoot:     "0x1234",
		Network:       "mainnet",
		Slot:          123,
		ProcessedChan: make(chan struct{}),
	}

	// Second block
	blockInfo2 := PermanentStoreBlock{
		Location:      blockLocation2,
		BlockRoot:     "0x1234", // Same root
		Network:       "goerli", // Different network
		Slot:          123,
		ProcessedChan: make(chan struct{}),
	}

	permanentStore.QueueBlock(blockInfo1)

	// Wait for the first block to be processed
	select {
	case <-blockInfo1.ProcessedChan:
		// Block 1 processed
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Block 1 processing timed out")
	}

	permanentStore.QueueBlock(blockInfo2)

	// Wait for the second block to be processed
	select {
	case <-blockInfo2.ProcessedChan:
		// Block 2 processed
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Block 2 processing timed out")
	}

	// Check if both blocks were copied to their respective permanent locations
	permanentLocation1 := permanentStore.GetPermanentLocation(blockInfo1)
	permanentLocation2 := permanentStore.GetPermanentLocation(blockInfo2)

	// Verify both exist
	exists1, err := mockStore.Exists(ctx, permanentLocation1)
	require.NoError(t, err)
	assert.True(t, exists1, "Block for mainnet should be copied to permanent location")

	exists2, err := mockStore.Exists(ctx, permanentLocation2)
	require.NoError(t, err)
	assert.True(t, exists2, "Block for goerli should be copied to permanent location")

	// Verify the block data
	data1, err := mockStore.GetBeaconBlock(ctx, permanentLocation1)
	require.NoError(t, err)
	assert.Equal(t, blockData1, *data1, "Block data for mainnet should match")

	data2, err := mockStore.GetBeaconBlock(ctx, permanentLocation2)
	require.NoError(t, err)
	assert.Equal(t, blockData2, *data2, "Block data for goerli should match")

	// Check that both permanent blocks were recorded in the database
	filter1 := &persistence.PermanentBlockFilter{}
	filter1.AddBlockRoot(blockInfo1.BlockRoot)
	filter1.AddNetwork(blockInfo1.Network)

	permanentBlocks1, err := permanentStore.db.ListPermanentBlock(ctx, filter1, &persistence.PaginationCursor{Limit: 1})
	require.NoError(t, err)
	assert.Len(t, permanentBlocks1, 1, "Permanent block should be recorded for mainnet")

	filter2 := &persistence.PermanentBlockFilter{}
	filter2.AddBlockRoot(blockInfo2.BlockRoot)
	filter2.AddNetwork(blockInfo2.Network)

	permanentBlocks2, err := permanentStore.db.ListPermanentBlock(ctx, filter2, &persistence.PaginationCursor{Limit: 1})
	require.NoError(t, err)
	assert.Len(t, permanentBlocks2, 1, "Permanent block should be recorded for goerli")
}

func TestPermanentStoreDistributedLock(t *testing.T) {
	ctx := context.Background()

	// Create a mock store
	bucket := "mybucket"

	mockStore, cleanup, err := store.NewMockS3Store(ctx, bucket)
	require.NoError(t, err)

	defer func() {
		if err := cleanup(); err != nil {
			t.Fatalf("Failed to clean up: %v", err)
		}
	}()

	// Use a shared indexer for both permanent stores
	indexer := setupMockIndexer(t)

	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	nodeID1 := "node-1"
	nodeID2 := "node-2"

	// Create and start the first permanent store
	permanentStore1, err := NewPermanentStore(log, mockStore, indexer, nodeID1, &PermanentStoreConfig{
		Blocks: BlockConfig{
			Enabled: true,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, permanentStore1)

	err = permanentStore1.Start(ctx)
	require.NoError(t, err)

	// Ensure the first permanent store is fully started
	time.Sleep(200 * time.Millisecond)

	// Create a test block
	blockData := []byte("test block data")
	blockLocation := "test/location/block1.ssz"

	// Save the block to the mock store
	_, err = mockStore.SaveBeaconBlock(ctx, &store.SaveParams{
		Location: blockLocation,
		Data:     &blockData,
	})
	require.NoError(t, err)

	// Process the block with the first permanent store
	blockInfo1 := PermanentStoreBlock{
		Location:      blockLocation,
		BlockRoot:     "0xabcd",
		Network:       "mainnet",
		ProcessedChan: make(chan struct{}),
	}

	permanentStore1.QueueBlock(blockInfo1)

	// Wait for the block to be processed
	select {
	case <-blockInfo1.ProcessedChan:
		// Block processed by store 1
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Block processing timed out")
	}

	// Check if the block was copied to the permanent location
	permanentLocation := permanentStore1.GetPermanentLocation(blockInfo1)
	exists, err := mockStore.Exists(ctx, permanentLocation)
	require.NoError(t, err)
	assert.True(t, exists, "Block should be copied to permanent location")

	// Verify the block data
	data, err := mockStore.GetBeaconBlock(ctx, permanentLocation)
	require.NoError(t, err)
	assert.Equal(t, blockData, *data, "Block data should match")

	// Now create and start the second permanent store
	permanentStore2, err := NewPermanentStore(log, mockStore, indexer, nodeID2, &PermanentStoreConfig{
		Blocks: BlockConfig{
			Enabled: true,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, permanentStore2)

	err = permanentStore2.Start(ctx)
	require.NoError(t, err)

	// Try to process the same block with the second permanent store
	blockInfo2 := PermanentStoreBlock{
		Location:      blockLocation,
		BlockRoot:     "0xabcd",
		Network:       "mainnet",
		ProcessedChan: make(chan struct{}),
	}

	permanentStore2.QueueBlock(blockInfo2)

	// Wait for the block to be processed
	select {
	case <-blockInfo2.ProcessedChan:
		// Block processed by store 2
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Block processing timed out")
	}

	// The block should still exist in the permanent location
	exists, err = mockStore.Exists(ctx, permanentLocation)
	require.NoError(t, err)
	assert.True(t, exists, "Block should still exist in permanent location")
}

func TestPermanentStoreStop(t *testing.T) {
	ctx := context.Background()
	permanentStore, mockStore, cleanup := setupPermanentStore(t)

	defer cleanup()

	// Create a test block
	blockData := []byte("test block data")
	blockLocation := "test/location/block1.ssz"

	// Save the block to the mock store
	_, err := mockStore.SaveBeaconBlock(ctx, &store.SaveParams{
		Location: blockLocation,
		Data:     &blockData,
	})
	require.NoError(t, err)

	// Create a channel to track when processing is complete
	processChan := make(chan struct{})

	// Queue a block for processing with the channel
	blockInfo := PermanentStoreBlock{
		Location:      blockLocation,
		BlockRoot:     "0xstop",
		Network:       "mainnet",
		ProcessedChan: processChan,
		Slot:          1,
	}

	// Queue the block
	permanentStore.QueueBlock(blockInfo)

	// Wait for the block to be processed
	select {
	case <-processChan:
		// Block was processed
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Block processing timed out")
	}

	// Verify the block was copied to the permanent location
	permanentLocation := permanentStore.GetPermanentLocation(blockInfo)
	exists, err := mockStore.Exists(ctx, permanentLocation)
	require.NoError(t, err)
	assert.True(t, exists, "Block should be copied to permanent location")

	// Now test the stop procedure with a block in the queue
	// Queue another block before stopping
	processChan2 := make(chan struct{})
	queuedBlock := PermanentStoreBlock{
		Location:      blockLocation,
		BlockRoot:     "0xqueued",
		Network:       "mainnet",
		ProcessedChan: processChan2,
		Slot:          2,
	}
	permanentStore.QueueBlock(queuedBlock)

	// Start a goroutine to stop the permanent store
	stopDone := make(chan struct{})
	go func() {
		err := permanentStore.Stop(ctx)
		require.NoError(t, err)
		close(stopDone)
	}()

	// Wait for the queued block to be processed
	select {
	case <-processChan2:
		// Block was processed
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Queued block processing timed out")
	}

	// Verify the queued block was copied to the permanent location
	queuedLocation := permanentStore.GetPermanentLocation(queuedBlock)
	exists, err = mockStore.Exists(ctx, queuedLocation)
	require.NoError(t, err)
	assert.True(t, exists, "Queued block should be copied to permanent location before Stop completes")

	// Wait for Stop to complete
	select {
	case <-stopDone:
		// Stop completed
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Stop did not complete in time")
	}

	// Try to queue a block after stopping
	unprocessedBlock := PermanentStoreBlock{
		Location:      blockLocation,
		BlockRoot:     "0xunprocessed",
		Network:       "mainnet",
		ProcessedChan: make(chan struct{}),
		Slot:          3,
	}
	permanentStore.QueueBlock(unprocessedBlock)

	// Wait a bit to ensure the block isn't processed
	time.Sleep(100 * time.Millisecond)

	// Verify the unprocessed block was not copied to the permanent location
	unprocessedLocation := permanentStore.GetPermanentLocation(unprocessedBlock)
	exists, err = mockStore.Exists(ctx, unprocessedLocation)
	require.NoError(t, err)
	assert.False(t, exists, "Block should not be copied after permanent store is stopped")

	// Verify that the stopped flag is set
	assert.True(t, permanentStore.stopped, "Stopped flag should be set")
}

func TestPermanentStoreLocation(t *testing.T) {
	ctx := context.Background()
	permanentStore, mockStore, cleanup := setupPermanentStore(t)

	defer cleanup()

	// Create a block with known values
	blockInfo := PermanentStoreBlock{
		Location:  "test/location/block.ssz",
		BlockRoot: "0xabcd1234",
		Network:   "mainnet",
		Slot:      123456,
	}

	// Get the permanent location
	permanentLocation := permanentStore.GetPermanentLocation(blockInfo)

	// Verify the location format
	expectedLocation := "permanent/mainnet/0xabcd1234.ssz"
	assert.Equal(t, expectedLocation, permanentLocation, "Permanent location should not include slot in the path")

	// Create a test file at the source location
	data := []byte("test data")
	params := &store.SaveParams{
		Location: blockInfo.Location,
		Data:     &data,
	}
	_, err := mockStore.SaveBeaconBlock(ctx, params)
	require.NoError(t, err)

	// Verify we can find blocks by querying the permanent block table
	processChan := make(chan struct{})
	blockInfo.ProcessedChan = processChan

	// Queue the block for processing
	permanentStore.QueueBlock(blockInfo)

	// Wait for processing to complete
	select {
	case <-processChan:
		// Block processed
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Block processing timed out")
	}

	// Search for blocks by slot
	filter := &persistence.PermanentBlockFilter{}
	filter.AddSlot(int64(blockInfo.Slot))

	blocks, err := permanentStore.db.ListPermanentBlock(ctx, filter, &persistence.PaginationCursor{Limit: 10})
	require.NoError(t, err)
	assert.Len(t, blocks, 1, "Should find one permanent block with the specified slot")
	if len(blocks) > 0 {
		assert.Equal(t, blockInfo.BlockRoot, blocks[0].BlockRoot)
		assert.Equal(t, blockInfo.Network, blocks[0].Network)
		assert.Equal(t, int64(blockInfo.Slot), blocks[0].Slot)
	}
}
