//nolint:gosec // Only used in tests
package persistence

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
)

func generateRandomPermanentBlock() *PermanentBlock {
	return &PermanentBlock{
		Slot:      generateRandomInt64(),
		BlockRoot: generateRandomString(32),
		Network:   generateRandomString(5),
	}
}

func TestInsertPermanentBlock(t *testing.T) {
	indexer, mock, err := NewMockIndexer()
	assert.NoError(t, err)

	ctx := context.Background()
	block := generateRandomPermanentBlock()

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO").WithArgs(
		sqlmock.AnyArg(), sqlmock.AnyArg(), nil, block.Slot,
		block.BlockRoot, block.Network,
	).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = indexer.InsertPermanentBlock(ctx, block)
	assert.NoError(t, err)
}

func TestCountPermanentBlock(t *testing.T) {
	indexer, _, err := NewMockIndexer()
	assert.NoError(t, err)

	ctx := context.Background()

	block := generateRandomPermanentBlock()

	err = indexer.InsertPermanentBlock(ctx, block)
	assert.NoError(t, err)

	filter := &PermanentBlockFilter{}
	filter.AddBlockRoot(block.BlockRoot)

	count, err := indexer.CountPermanentBlock(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestListPermanentBlock(t *testing.T) {
	indexer, _, err := NewMockIndexer()
	assert.NoError(t, err)

	ctx := context.Background()

	// Insert 3 blocks to test listing
	for i := 0; i < 3; i++ {
		block := generateRandomPermanentBlock()
		block.Slot = int64(i + 1) // Ensure different slots
		block.Network = "test-network"

		err = indexer.InsertPermanentBlock(ctx, block)
		assert.NoError(t, err)
	}

	// List all blocks
	filter := &PermanentBlockFilter{}
	filter.AddNetwork("test-network")

	blocks, err := indexer.ListPermanentBlock(ctx, filter, &PaginationCursor{Limit: 10})
	assert.NoError(t, err)
	assert.Len(t, blocks, 3)

	// Test pagination
	blocks, err = indexer.ListPermanentBlock(ctx, filter, &PaginationCursor{Limit: 2})
	assert.NoError(t, err)
	assert.Len(t, blocks, 2)
}

func TestGetPermanentBlockByBlockRoot(t *testing.T) {
	indexer, _, err := NewMockIndexer()
	assert.NoError(t, err)

	ctx := context.Background()
	block := generateRandomPermanentBlock()
	block.BlockRoot = "test-block-root"
	block.Network = "test-network"

	err = indexer.InsertPermanentBlock(ctx, block)
	assert.NoError(t, err)

	// Get the block by block root and network
	result, err := indexer.GetPermanentBlockByBlockRoot(ctx, "test-block-root", "test-network")
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, block.BlockRoot, result.BlockRoot)
	assert.Equal(t, block.Network, result.Network)
	assert.Equal(t, block.Slot, result.Slot)

	// Try to get a non-existent block
	result, err = indexer.GetPermanentBlockByBlockRoot(ctx, "non-existent", "test-network")
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestDistinctPermanentBlockValues(t *testing.T) {
	indexer, _, err := NewMockIndexer()
	assert.NoError(t, err)

	ctx := context.Background()

	// Insert 20 blocks with some duplicate values
	networks := []string{"mainnet", "goerli", "sepolia"}
	slots := []int64{1, 2, 3, 4, 5}

	for i := 0; i < 20; i++ {
		block := &PermanentBlock{
			Slot:      slots[i%len(slots)],
			BlockRoot: fmt.Sprintf("block-root-%d", i),
			Network:   networks[i%len(networks)],
		}

		err = indexer.InsertPermanentBlock(ctx, block)
		assert.NoError(t, err)
	}

	// Test getting distinct values
	distinctValues, err := indexer.DistinctPermanentBlockValues(ctx, []string{"network", "slot"})
	assert.NoError(t, err)
	assert.NotNil(t, distinctValues)

	// Check that we have the right number of distinct networks
	assert.Len(t, distinctValues.Network, len(networks))

	for _, network := range networks {
		found := false

		for _, n := range distinctValues.Network {
			if n == network {
				found = true

				break
			}
		}

		assert.True(t, found, "Expected to find network %s in distinct values", network)
	}

	// Check that we have the right number of distinct slots
	assert.Len(t, distinctValues.Slot, len(slots))

	for _, slot := range slots {
		found := false

		for _, s := range distinctValues.Slot {
			if uint64(slot) == s {
				found = true

				break
			}
		}

		assert.True(t, found, "Expected to find slot %d in distinct values", slot)
	}
}
