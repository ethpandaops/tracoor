package indexer

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/ethpandaops/tracoor/pkg/server/persistence"
	"github.com/ethpandaops/tracoor/pkg/store"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupMockIndexer creates a mock indexer with an in-memory SQLite database
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

	return permanentStore, mockStore, clean
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
		// Queue a block for processing
		blockInfo := PermanentStoreBlock{
			Location:  blockLocation,
			BlockRoot: "0x1234",
			Network:   "mainnet",
		}

		permanentStore.QueueBlock(blockInfo)

		// Wait for the block to be processed
		time.Sleep(100 * time.Millisecond)

		// Check if the block was copied to the permanent location
		permanentLocation := filepath.Join("permanent", blockInfo.Network, "blocks", blockInfo.BlockRoot+filepath.Ext(blockLocation))
		exists, err := mockStore.Exists(ctx, permanentLocation)
		require.NoError(t, err)
		assert.True(t, exists, "Block should be copied to permanent location")

		// Verify the block data
		data, err := mockStore.GetBeaconBlock(ctx, permanentLocation)
		require.NoError(t, err)
		assert.Equal(t, blockData, *data, "Block data should match")
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

	// Queue a block for processing
	blockInfo := PermanentStoreBlock{
		Location:  blockLocation,
		BlockRoot: "0x1234",
		Network:   "mainnet",
	}

	// Process the block first time
	permanentStore.QueueBlock(blockInfo)
	time.Sleep(100 * time.Millisecond)

	// Get the permanent location
	permanentLocation := filepath.Join("permanent", blockInfo.Network, "blocks", blockInfo.BlockRoot+filepath.Ext(blockLocation))

	// Verify it exists
	exists, err := mockStore.Exists(ctx, permanentLocation)
	require.NoError(t, err)
	assert.True(t, exists, "Block should be copied to permanent location")

	// Process the same block again
	permanentStore.QueueBlock(blockInfo)
	time.Sleep(100 * time.Millisecond)

	// The block should still exist in the permanent location
	exists, err = mockStore.Exists(ctx, permanentLocation)
	require.NoError(t, err)
	assert.True(t, exists, "Block should still exist in permanent location")
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

	// Queue blocks with same root but different networks
	blockInfo1 := PermanentStoreBlock{
		Location:  blockLocation1,
		BlockRoot: "0x1234",
		Network:   "mainnet",
	}

	blockInfo2 := PermanentStoreBlock{
		Location:  blockLocation2,
		BlockRoot: "0x1234", // Same root
		Network:   "goerli", // Different network
	}

	// Process both blocks
	permanentStore.QueueBlock(blockInfo1)
	permanentStore.QueueBlock(blockInfo2)
	time.Sleep(200 * time.Millisecond)

	// Check if both blocks were copied to their respective permanent locations
	permanentLocation1 := filepath.Join("permanent", blockInfo1.Network, "blocks", blockInfo1.BlockRoot+filepath.Ext(blockLocation1))
	permanentLocation2 := filepath.Join("permanent", blockInfo2.Network, "blocks", blockInfo2.BlockRoot+filepath.Ext(blockLocation2))

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

	indexer := setupMockIndexer(t)

	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	nodeID1 := "node-1"
	nodeID2 := "node-2"

	permanentStore1, err := NewPermanentStore(log, mockStore, indexer, nodeID1, &PermanentStoreConfig{
		Blocks: BlockConfig{
			Enabled: true,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, permanentStore1)

	permanentStore2, err := NewPermanentStore(log, mockStore, indexer, nodeID2, &PermanentStoreConfig{
		Blocks: BlockConfig{
			Enabled: true,
		},
	})
	require.NoError(t, err)

	require.NotNil(t, permanentStore2)

	// Start both permanent stores
	err = permanentStore1.Start(ctx)
	require.NoError(t, err)

	err = permanentStore2.Start(ctx)
	require.NoError(t, err)

	// Create a test block
	blockData := []byte("test block data")
	blockLocation := "test/location/block1.ssz"

	// Save the block to the mock store
	_, err = mockStore.SaveBeaconBlock(ctx, &store.SaveParams{
		Location: blockLocation,
		Data:     &blockData,
	})
	require.NoError(t, err)

	// Create a block info
	blockInfo := PermanentStoreBlock{
		Location:  blockLocation,
		BlockRoot: "0xabcd",
		Network:   "mainnet",
	}

	// Queue the block to both permanent stores
	permanentStore1.QueueBlock(blockInfo)
	permanentStore2.QueueBlock(blockInfo)

	// Wait for the blocks to be processed
	time.Sleep(200 * time.Millisecond)

	// Check if the block was copied to the permanent location
	permanentLocation := filepath.Join("permanent", blockInfo.Network, "blocks", blockInfo.BlockRoot+filepath.Ext(blockLocation))
	exists, err := mockStore.Exists(ctx, permanentLocation)
	require.NoError(t, err)
	assert.True(t, exists, "Block should be copied to permanent location")

	// Verify the block data
	data, err := mockStore.GetBeaconBlock(ctx, permanentLocation)
	require.NoError(t, err)
	assert.Equal(t, blockData, *data, "Block data should match")
}
