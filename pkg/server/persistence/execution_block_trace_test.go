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

func generateRandomExecutionBlockTrace() *ExecutionBlockTrace {
	return &ExecutionBlockTrace{
		ID:                      generateRandomString(10),
		Node:                    generateRandomString(5),
		FetchedAt:               time.Now(),
		ExecutionImplementation: generateRandomString(15),
		NodeVersion:             generateRandomString(8),
		Location:                generateRandomString(10),
		Network:                 generateRandomString(5),
		BlockHash:               generateRandomString(64),
		BlockNumber:             generateRandomInt64(),
	}
}

func TestInsertExecutionBlockTrace(t *testing.T) {
	indexer, mock, err := newMockIndexer()
	assert.NoError(t, err)

	ctx := context.Background()
	trace := generateRandomExecutionBlockTrace()

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO").WithArgs(
		sqlmock.AnyArg(), sqlmock.AnyArg(), nil, trace.ID, trace.Node, trace.FetchedAt, trace.ExecutionImplementation,
		trace.NodeVersion, trace.Location, trace.Network, trace.BlockHash, trace.BlockNumber,
	).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = indexer.InsertExecutionBlockTrace(ctx, trace)
	assert.NoError(t, err)
}

func TestRemoveExecutionBlockTrace(t *testing.T) {
	indexer, mock, err := newMockIndexer()
	assert.NoError(t, err)

	ctx := context.Background()
	id := "test-id"

	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM").WithArgs(id).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = indexer.DeleteExecutionBlockTrace(ctx, id)
	assert.NoError(t, err)
}

func TestCountExecutionBlockTrace(t *testing.T) {
	indexer, _, err := newMockIndexer()
	assert.NoError(t, err)

	ctx := context.Background()

	trace := generateRandomExecutionBlockTrace()

	err = indexer.InsertExecutionBlockTrace(ctx, trace)
	assert.NoError(t, err)

	filter := &ExecutionBlockTraceFilter{
		ID: &trace.ID,
	}

	count, err := indexer.CountExecutionBlockTrace(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestListExecutionBlockTrace(t *testing.T) {
	indexer, mock, err := newMockIndexer()
	assert.NoError(t, err)

	ctx := context.Background()
	trace := generateRandomExecutionBlockTrace()

	err = indexer.InsertExecutionBlockTrace(ctx, trace)
	assert.NoError(t, err)

	filter := &ExecutionBlockTraceFilter{
		ID: &trace.ID,
	}
	page := &PaginationCursor{}

	mock.ExpectQuery("SELECT \\* FROM").WithArgs(filter.ID).WillReturnRows(sqlmock.NewRows([]string{"id", "node"}).AddRow("test-id", "test-node"))

	traces, err := indexer.ListExecutionBlockTrace(ctx, filter, page)
	assert.NoError(t, err)
	assert.Len(t, traces, 1)
	assert.Equal(t, trace.ID, traces[0].ID)
	assert.Equal(t, trace.Node, traces[0].Node)
}

func TestExecutionBlockTraceFilters(t *testing.T) {
	t.Run("By individual attributes", func(t *testing.T) {
		indexer, _, err := newMockIndexer()
		if err != nil {
			t.Fatal(err)
		}

		trace := generateRandomExecutionBlockTrace()

		err = indexer.InsertExecutionBlockTrace(context.Background(), trace)
		if err != nil {
			t.Fatal(err)
		}

		// Test filters individually
		testCases := []struct {
			name   string
			filter ExecutionBlockTraceFilter
		}{
			{"ID", ExecutionBlockTraceFilter{ID: &trace.ID}},
			{"Node", ExecutionBlockTraceFilter{Node: &trace.Node}},
			{"BlockHash", ExecutionBlockTraceFilter{BlockHash: &trace.BlockHash}},
			{"BlockNumber", ExecutionBlockTraceFilter{BlockNumber: &trace.BlockNumber}},
			{"NodeVersion", ExecutionBlockTraceFilter{NodeVersion: &trace.NodeVersion}},
			{"Location", ExecutionBlockTraceFilter{Location: &trace.Location}},
			{"Network", ExecutionBlockTraceFilter{Network: &trace.Network}},
			{"ExecutionImplementation", ExecutionBlockTraceFilter{ExecutionImplementation: &trace.ExecutionImplementation}},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				traces, err := indexer.ListExecutionBlockTrace(context.Background(), &tc.filter, &PaginationCursor{})
				if err != nil {
					t.Fatal(err)
				}

				if len(traces) != 1 {
					t.Fatalf("expected 1 execution block trace, got %d", len(traces))
				}

				if traces[0].ID != trace.ID {
					t.Fatalf("expected ID %s, got %s", trace.ID, traces[0].ID)
				}
			})
		}
	})
	t.Run("By random combinations", func(t *testing.T) {
		indexer, _, err := newMockIndexer()
		if err != nil {
			t.Fatal(err)
		}

		// Add 10000 random execution block traces
		for i := 0; i < 10000; i++ {
			id := uuid.New().String()
			node := fmt.Sprintf("node-%d", i)
			blockHash := fmt.Sprintf("blockHash-%d", rand.Intn(100))
			blockNumber := int64(rand.Intn(1000))
			nodeVersion := fmt.Sprintf("version%d", rand.Intn(10))
			location := fmt.Sprintf("location%d", rand.Intn(10))
			network := fmt.Sprintf("network%d", rand.Intn(10))
			executionImplementation := fmt.Sprintf("implementation%d", rand.Intn(10))

			executionBlockTrace := &ExecutionBlockTrace{
				ID:                      id,
				Node:                    node,
				BlockHash:               blockHash,
				BlockNumber:             blockNumber,
				NodeVersion:             nodeVersion,
				Location:                location,
				Network:                 network,
				ExecutionImplementation: executionImplementation,
				FetchedAt:               time.Now(),
			}

			err = indexer.InsertExecutionBlockTrace(context.Background(), executionBlockTrace)
			if err != nil {
				t.Fatal(err)
			}
		}

		// List execution block traces with random filters
		for i := 0; i < 5000; i++ {
			var filter ExecutionBlockTraceFilter

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

			executionBlockTraces, err := indexer.ListExecutionBlockTrace(context.Background(), &filter, &PaginationCursor{})
			if err != nil {
				t.Fatal(err)
			}

			for _, executionBlockTrace := range executionBlockTraces {
				if filter.ID != nil && *filter.ID != executionBlockTrace.ID {
					t.Fatalf("expected ID %s, got %s", *filter.ID, executionBlockTrace.ID)
				}

				if filter.Node != nil && *filter.Node != executionBlockTrace.Node {
					t.Fatalf("expected Node %s, got %s", *filter.Node, executionBlockTrace.Node)
				}

				if filter.Before != nil && executionBlockTrace.FetchedAt.After(*filter.Before) {
					t.Fatalf("expected FetchedAt before %s, got %s", *filter.Before, executionBlockTrace.FetchedAt)
				}

				if filter.After != nil && executionBlockTrace.FetchedAt.Before(*filter.After) {
					t.Fatalf("expected FetchedAt after %s, got %s", *filter.After, executionBlockTrace.FetchedAt)
				}

				if filter.BlockHash != nil && *filter.BlockHash != executionBlockTrace.BlockHash {
					t.Fatalf("expected BlockHash %s, got %s", *filter.BlockHash, executionBlockTrace.BlockHash)
				}

				if filter.BlockNumber != nil && *filter.BlockNumber != executionBlockTrace.BlockNumber {
					t.Fatalf("expected BlockNumber %d, got %d", *filter.BlockNumber, executionBlockTrace.BlockNumber)
				}

				if filter.NodeVersion != nil && *filter.NodeVersion != executionBlockTrace.NodeVersion {
					t.Fatalf("expected NodeVersion %s, got %s", *filter.NodeVersion, executionBlockTrace.NodeVersion)
				}

				if filter.Location != nil && *filter.Location != executionBlockTrace.Location {
					t.Fatalf("expected Location %s, got %s", *filter.Location, executionBlockTrace.Location)
				}

				if filter.Network != nil && *filter.Network != executionBlockTrace.Network {
					t.Fatalf("expected Network %s, got %s", *filter.Network, executionBlockTrace.Network)
				}

				if filter.ExecutionImplementation != nil && *filter.ExecutionImplementation != executionBlockTrace.ExecutionImplementation {
					t.Fatalf("expected ExecutionImplementation %s, got %s", *filter.ExecutionImplementation, executionBlockTrace.ExecutionImplementation)
				}
			}
		}
	})

}
