//nolint:gosec // Only used in tests
package persistence

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
)

func generateRandomBeaconBlock() *BeaconBlock {
	return &BeaconBlock{
		ID:                   generateRandomString(10),
		Node:                 generateRandomString(5),
		Slot:                 generateRandomInt64(),
		Epoch:                generateRandomInt64(),
		BlockRoot:            generateRandomString(32),
		FetchedAt:            time.Now(),
		BeaconImplementation: generateRandomString(15),
		NodeVersion:          generateRandomString(8),
		Location:             generateRandomString(10),
		Network:              generateRandomString(5),
	}
}
func TestInsertBeaconBlock(t *testing.T) {
	indexer, mock, err := NewMockIndexer()
	assert.NoError(t, err)

	ctx := context.Background()
	block := generateRandomBeaconBlock()

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO").WithArgs(
		sqlmock.AnyArg(), sqlmock.AnyArg(), nil, block.ID, block.Node, block.Slot, block.Epoch,
		block.BlockRoot, block.FetchedAt, block.BeaconImplementation, block.NodeVersion,
		block.Location, block.Network,
	).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = indexer.InsertBeaconBlock(ctx, block)
	assert.NoError(t, err)
}

func TestRemoveBeaconBlock(t *testing.T) {
	indexer, mock, err := NewMockIndexer()
	assert.NoError(t, err)

	ctx := context.Background()
	id := "test-id"

	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM").WithArgs(id).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = indexer.RemoveBeaconBlock(ctx, id)
	assert.NoError(t, err)
}

func TestCountBeaconBlock(t *testing.T) {
	indexer, _, err := NewMockIndexer()
	assert.NoError(t, err)

	ctx := context.Background()

	block := generateRandomBeaconBlock()

	err = indexer.InsertBeaconBlock(ctx, block)
	assert.NoError(t, err)

	filter := &BeaconBlockFilter{
		ID: &block.ID,
	}

	count, err := indexer.CountBeaconBlock(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestListBeaconBlock(t *testing.T) {
	indexer, mock, err := NewMockIndexer()
	assert.NoError(t, err)

	ctx := context.Background()
	block := generateRandomBeaconBlock()

	err = indexer.InsertBeaconBlock(ctx, block)
	assert.NoError(t, err)

	filter := &BeaconBlockFilter{
		ID: &block.ID,
	}
	page := &PaginationCursor{}

	mock.ExpectQuery("SELECT \\* FROM").WithArgs(filter.ID).WillReturnRows(sqlmock.NewRows([]string{"id", "node"}).AddRow("test-id", "test-node"))

	blocks, err := indexer.ListBeaconBlock(ctx, filter, page)
	assert.NoError(t, err)
	assert.Len(t, blocks, 1)
	assert.Equal(t, block.ID, blocks[0].ID)
	assert.Equal(t, block.Node, blocks[0].Node)
}

func TestUpdateBeaconBlock(t *testing.T) {
	indexer, mock, err := NewMockIndexer()
	assert.NoError(t, err)

	ctx := context.Background()
	block := generateRandomBeaconBlock()

	err = indexer.InsertBeaconBlock(ctx, block)
	assert.NoError(t, err)

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE").WithArgs(
		block.ID, block.Node, block.Slot, block.Epoch, block.BlockRoot, block.FetchedAt,
		block.BeaconImplementation, block.NodeVersion, block.Location, block.Network,
	).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = indexer.UpdateBeaconBlock(ctx, block)
	assert.NoError(t, err)
}

//nolint:gocyclo // Test is long but manageable
func TestBeaconBlockFilters(t *testing.T) {
	t.Run("By random combinations", func(t *testing.T) {
		indexer, _, err := NewMockIndexer()
		if err != nil {
			t.Fatal(err)
		}

		// Add 10000 random beacon blocks
		for i := 0; i < 10000; i++ {
			id := uuid.New().String()
			node := fmt.Sprintf("node-%d", i)
			slot := int64(rand.Intn(1000))
			epoch := int64(rand.Intn(100))
			blockRoot := fmt.Sprintf("blockRoot-%d", rand.Intn(100))
			nodeVersion := fmt.Sprintf("version%d", rand.Intn(10))
			location := fmt.Sprintf("location%d", rand.Intn(10))
			network := fmt.Sprintf("network%d", rand.Intn(10))
			beaconImplementation := fmt.Sprintf("implementation%d", rand.Intn(10))

			beaconBlock := &BeaconBlock{
				ID:                   id,
				Node:                 node,
				Slot:                 slot,
				Epoch:                epoch,
				BlockRoot:            blockRoot,
				FetchedAt:            time.Now(),
				NodeVersion:          nodeVersion,
				Location:             location,
				Network:              network,
				BeaconImplementation: beaconImplementation,
			}

			err = indexer.InsertBeaconBlock(context.Background(), beaconBlock)
			if err != nil {
				t.Fatal(err)
			}
		}

		// List beacon blocks with random filters
		for i := 0; i < 5000; i++ {
			var filter BeaconBlockFilter

			if rand.Intn(2) == 1 {
				filter.AddID(uuid.New().String())
			}

			if rand.Intn(2) == 1 {
				filter.AddNode(fmt.Sprintf("node-%d", rand.Intn(100)))
			}

			if rand.Intn(2) == 1 {
				filter.AddBefore(time.Now())
			}

			if rand.Intn(2) == 1 {
				filter.AddAfter(time.Now().Add(-24 * time.Hour))
			}

			if rand.Intn(2) == 1 {
				filter.AddSlot(uint64(rand.Intn(1000)))
			}

			if rand.Intn(2) == 1 {
				filter.AddEpoch(uint64(rand.Intn(100)))
			}

			if rand.Intn(2) == 1 {
				filter.AddBlockRoot(fmt.Sprintf("blockRoot-%d", rand.Intn(100)))
			}

			if rand.Intn(2) == 1 {
				filter.AddNodeVersion(fmt.Sprintf("version%d", rand.Intn(10)))
			}

			if rand.Intn(2) == 1 {
				filter.AddLocation(fmt.Sprintf("location%d", rand.Intn(10)))
			}

			if rand.Intn(2) == 1 {
				filter.AddNetwork(fmt.Sprintf("network%d", rand.Intn(10)))
			}

			if rand.Intn(2) == 1 {
				filter.AddBeaconImplementation(fmt.Sprintf("implementation%d", rand.Intn(10)))
			}

			beaconBlocks, err := indexer.ListBeaconBlock(context.Background(), &filter, &PaginationCursor{})
			if err != nil {
				t.Fatal(err)
			}

			for _, beaconBlock := range beaconBlocks {
				if filter.ID != nil && *filter.ID != beaconBlock.ID {
					t.Fatalf("expected ID %s, got %s", *filter.ID, beaconBlock.ID)
				}

				if filter.Node != nil && *filter.Node != beaconBlock.Node {
					t.Fatalf("expected Node %s, got %s", *filter.Node, beaconBlock.Node)
				}

				if filter.Before != nil && beaconBlock.FetchedAt.After(*filter.Before) {
					t.Fatalf("expected FetchedAt before %s, got %s", *filter.Before, beaconBlock.FetchedAt)
				}

				if filter.After != nil && beaconBlock.FetchedAt.Before(*filter.After) {
					t.Fatalf("expected FetchedAt after %s, got %s", *filter.After, beaconBlock.FetchedAt)
				}

				if filter.Slot != nil && *filter.Slot != uint64(beaconBlock.Slot) {
					t.Fatalf("expected Slot %d, got %d", *filter.Slot, beaconBlock.Slot)
				}

				if filter.Epoch != nil && *filter.Epoch != uint64(beaconBlock.Epoch) {
					t.Fatalf("expected Epoch %d, got %d", *filter.Epoch, beaconBlock.Epoch)
				}

				if filter.BlockRoot != nil && *filter.BlockRoot != beaconBlock.BlockRoot {
					t.Fatalf("expected BlockRoot %s, got %s", *filter.BlockRoot, beaconBlock.BlockRoot)
				}

				if filter.NodeVersion != nil && *filter.NodeVersion != beaconBlock.NodeVersion {
					t.Fatalf("expected NodeVersion %s, got %s", *filter.NodeVersion, beaconBlock.NodeVersion)
				}

				if filter.Location != nil && *filter.Location != beaconBlock.Location {
					t.Fatalf("expected Location %s, got %s", *filter.Location, beaconBlock.Location)
				}

				if filter.Network != nil && *filter.Network != beaconBlock.Network {
					t.Fatalf("expected Network %s, got %s", *filter.Network, beaconBlock.Network)
				}

				if filter.BeaconImplementation != nil && *filter.BeaconImplementation != beaconBlock.BeaconImplementation {
					t.Fatalf("expected BeaconImplementation %s, got %s", *filter.BeaconImplementation, beaconBlock.BeaconImplementation)
				}
			}
		}
	})
}
func TestBeaconBlockIndividualFilters(t *testing.T) {
	t.Run("By individual attributes", func(t *testing.T) {
		indexer, _, err := NewMockIndexer()
		if err != nil {
			t.Fatal(err)
		}

		beaconBlock := generateRandomBeaconBlock()

		err = indexer.InsertBeaconBlock(context.Background(), beaconBlock)
		if err != nil {
			t.Fatal(err)
		}

		slot := uint64(beaconBlock.Slot)
		epoch := uint64(beaconBlock.Epoch)

		// Test filters individually
		testCases := []struct {
			name   string
			filter BeaconBlockFilter
		}{
			{"ID", BeaconBlockFilter{ID: &beaconBlock.ID}},
			{"Node", BeaconBlockFilter{Node: &beaconBlock.Node}},
			{"Slot", BeaconBlockFilter{Slot: &slot}},
			{"Epoch", BeaconBlockFilter{Epoch: &epoch}},
			{"BlockRoot", BeaconBlockFilter{BlockRoot: &beaconBlock.BlockRoot}},
			{"NodeVersion", BeaconBlockFilter{NodeVersion: &beaconBlock.NodeVersion}},
			{"Location", BeaconBlockFilter{Location: &beaconBlock.Location}},
			{"Network", BeaconBlockFilter{Network: &beaconBlock.Network}},
			{"BeaconImplementation", BeaconBlockFilter{BeaconImplementation: &beaconBlock.BeaconImplementation}},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				beaconBlocks, err := indexer.ListBeaconBlock(context.Background(), &tc.filter, &PaginationCursor{})
				if err != nil {
					t.Fatal(err)
				}

				if len(beaconBlocks) != 1 {
					t.Fatalf("expected 1 beacon block, got %d", len(beaconBlocks))
				}

				if beaconBlocks[0].ID != beaconBlock.ID {
					t.Fatalf("expected ID %s, got %s", beaconBlock.ID, beaconBlocks[0].ID)
				}
			})
		}
	})
}
