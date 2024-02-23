//nolint:gosec // Only used in tests
package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
)

func generateRandomExecutionBadBlock() *ExecutionBadBlock {
	return &ExecutionBadBlock{
		ID:                      generateRandomString(10),
		Node:                    generateRandomString(5),
		FetchedAt:               time.Now(),
		ExecutionImplementation: generateRandomString(15),
		NodeVersion:             generateRandomString(8),
		Location:                generateRandomString(10),
		Network:                 generateRandomString(5),
		BlockHash:               generateRandomString(64),
		BlockNumber:             sql.NullInt64{Int64: generateRandomInt64(), Valid: true},
		BlockExtraData:          sql.NullString{String: generateRandomString(20), Valid: true},
	}
}

func TestInsertExecutionBadBlock(t *testing.T) {
	indexer, mock, err := NewMockIndexer()
	assert.NoError(t, err)

	ctx := context.Background()
	block := generateRandomExecutionBadBlock()

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO").WithArgs(
		sqlmock.AnyArg(), sqlmock.AnyArg(), nil, block.ID, block.Node, block.FetchedAt, block.ExecutionImplementation,
		block.NodeVersion, block.Location, block.Network, block.BlockHash, block.BlockNumber, block.BlockExtraData,
	).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = indexer.InsertExecutionBadBlock(ctx, block)
	assert.NoError(t, err)
}

func TestRemoveExecutionBadBlock(t *testing.T) {
	indexer, mock, err := NewMockIndexer()
	assert.NoError(t, err)

	ctx := context.Background()
	id := "test-id"

	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM").WithArgs(id).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = indexer.DeleteExecutionBadBlock(ctx, id)
	assert.NoError(t, err)
}

func TestCountExecutionBadBlock(t *testing.T) {
	indexer, _, err := NewMockIndexer()
	assert.NoError(t, err)

	ctx := context.Background()

	block := generateRandomExecutionBadBlock()

	err = indexer.InsertExecutionBadBlock(ctx, block)
	assert.NoError(t, err)

	filter := &ExecutionBadBlockFilter{
		ID: &block.ID,
	}

	count, err := indexer.CountExecutionBadBlock(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestListExecutionBadBlock(t *testing.T) {
	indexer, mock, err := NewMockIndexer()
	assert.NoError(t, err)

	ctx := context.Background()
	block := generateRandomExecutionBadBlock()

	err = indexer.InsertExecutionBadBlock(ctx, block)
	assert.NoError(t, err)

	filter := &ExecutionBadBlockFilter{
		ID: &block.ID,
	}
	page := &PaginationCursor{}

	mock.ExpectQuery("SELECT \\* FROM").WithArgs(filter.ID).WillReturnRows(sqlmock.NewRows([]string{"id", "node"}).AddRow("test-id", "test-node"))

	blocks, err := indexer.ListExecutionBadBlock(ctx, filter, page)
	assert.NoError(t, err)
	assert.Len(t, blocks, 1)
	assert.Equal(t, block.ID, blocks[0].ID)
	assert.Equal(t, block.Node, blocks[0].Node)
}

//nolint:gocyclo // Test is long but manageable
func TestExecutionBadBlockFilters(t *testing.T) {
	t.Run("By random combinations", func(t *testing.T) {
		indexer, _, err := NewMockIndexer()
		if err != nil {
			t.Fatal(err)
		}

		// Add 10000 random execution bad blocks
		for i := 0; i < 10000; i++ {
			id := uuid.New().String()
			node := fmt.Sprintf("node-%d", i)
			blockHash := fmt.Sprintf("blockHash-%d", rand.Intn(100))
			nodeVersion := fmt.Sprintf("version%d", rand.Intn(10))
			location := fmt.Sprintf("location%d", rand.Intn(10))
			network := fmt.Sprintf("network%d", rand.Intn(10))
			executionImplementation := fmt.Sprintf("implementation%d", rand.Intn(10))
			blockNumber := sql.NullInt64{Int64: int64(rand.Intn(1000)), Valid: true}
			blockExtraData := sql.NullString{String: fmt.Sprintf("extraData%d", rand.Intn(100)), Valid: true}

			executionBadBlock := &ExecutionBadBlock{
				ID:                      id,
				Node:                    node,
				FetchedAt:               time.Now(),
				ExecutionImplementation: executionImplementation,
				NodeVersion:             nodeVersion,
				Location:                location,
				Network:                 network,
				BlockHash:               blockHash,
				BlockNumber:             blockNumber,
				BlockExtraData:          blockExtraData,
			}

			err = indexer.InsertExecutionBadBlock(context.Background(), executionBadBlock)
			if err != nil {
				t.Fatal(err)
			}
		}

		// List execution bad blocks with random filters
		for i := 0; i < 5000; i++ {
			var filter ExecutionBadBlockFilter

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
				filter.AddBlockHash(fmt.Sprintf("blockHash-%d", rand.Intn(100)))
			}

			if rand.Intn(2) == 1 {
				filter.AddBlockNumber(int64(rand.Intn(1000)))
			}

			if rand.Intn(2) == 1 {
				filter.AddBlockExtraData(fmt.Sprintf("extraData%d", rand.Intn(100)))
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
				filter.AddExecutionImplementation(fmt.Sprintf("implementation%d", rand.Intn(10)))
			}

			blocks, err := indexer.ListExecutionBadBlock(context.Background(), &filter, &PaginationCursor{})
			if err != nil {
				t.Fatal(err)
			}

			for _, block := range blocks {
				if filter.ID != nil && *filter.ID != block.ID {
					t.Fatalf("expected ID %s, got %s", *filter.ID, block.ID)
				}

				if filter.Node != nil && *filter.Node != block.Node {
					t.Fatalf("expected Node %s, got %s", *filter.Node, block.Node)
				}

				if filter.Before != nil && block.FetchedAt.After(*filter.Before) {
					t.Fatalf("expected FetchedAt before %s, got %s", *filter.Before, block.FetchedAt)
				}

				if filter.After != nil && block.FetchedAt.Before(*filter.After) {
					t.Fatalf("expected FetchedAt after %s, got %s", *filter.After, block.FetchedAt)
				}

				if filter.BlockHash != nil && *filter.BlockHash != block.BlockHash {
					t.Fatalf("expected BlockHash %s, got %s", *filter.BlockHash, block.BlockHash)
				}

				if filter.BlockNumber != nil && *filter.BlockNumber != block.BlockNumber.Int64 {
					t.Fatalf("expected BlockNumber %d, got %d", *filter.BlockNumber, block.BlockNumber.Int64)
				}

				if filter.BlockExtraData != nil && *filter.BlockExtraData != block.BlockExtraData.String {
					t.Fatalf("expected BlockExtraData %s, got %s", *filter.BlockExtraData, block.BlockExtraData.String)
				}

				if filter.NodeVersion != nil && *filter.NodeVersion != block.NodeVersion {
					t.Fatalf("expected NodeVersion %s, got %s", *filter.NodeVersion, block.NodeVersion)
				}

				if filter.Location != nil && *filter.Location != block.Location {
					t.Fatalf("expected Location %s, got %s", *filter.Location, block.Location)
				}

				if filter.Network != nil && *filter.Network != block.Network {
					t.Fatalf("expected Network %s, got %s", *filter.Network, block.Network)
				}

				if filter.ExecutionImplementation != nil && *filter.ExecutionImplementation != block.ExecutionImplementation {
					t.Fatalf("expected ExecutionImplementation %s, got %s", *filter.ExecutionImplementation, block.ExecutionImplementation)
				}
			}
		}
	})
}
func TestExecutionBadBlockIndividualFilters(t *testing.T) {
	t.Run("By individual attributes", func(t *testing.T) {
		indexer, _, err := NewMockIndexer()
		if err != nil {
			t.Fatal(err)
		}

		block := generateRandomExecutionBadBlock()

		err = indexer.InsertExecutionBadBlock(context.Background(), block)
		if err != nil {
			t.Fatal(err)
		}

		// Test filters individually
		testCases := []struct {
			name   string
			filter ExecutionBadBlockFilter
		}{
			{"ID", ExecutionBadBlockFilter{ID: &block.ID}},
			{"Node", ExecutionBadBlockFilter{Node: &block.Node}},
			{"BlockHash", ExecutionBadBlockFilter{BlockHash: &block.BlockHash}},
			{"BlockNumber", ExecutionBadBlockFilter{BlockNumber: &block.BlockNumber.Int64}},
			{"BlockExtraData", ExecutionBadBlockFilter{BlockExtraData: &block.BlockExtraData.String}},
			{"NodeVersion", ExecutionBadBlockFilter{NodeVersion: &block.NodeVersion}},
			{"Location", ExecutionBadBlockFilter{Location: &block.Location}},
			{"Network", ExecutionBadBlockFilter{Network: &block.Network}},
			{"ExecutionImplementation", ExecutionBadBlockFilter{ExecutionImplementation: &block.ExecutionImplementation}},
		}

		for _, tc := range testCases {
			ttc := tc

			t.Run(tc.name, func(t *testing.T) {
				blocks, err := indexer.ListExecutionBadBlock(context.Background(), &ttc.filter, &PaginationCursor{})
				if err != nil {
					t.Fatal(err)
				}

				if len(blocks) != 1 {
					t.Fatalf("expected 1 execution bad block, got %d", len(blocks))
				}

				if blocks[0].ID != block.ID {
					t.Fatalf("expected ID %s, got %s", block.ID, blocks[0].ID)
				}
			})
		}
	})
}
