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

func generateRandomBeaconBadBlob() *BeaconBadBlob {
	return &BeaconBadBlob{
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
		Index:                generateRandomInt64(),
	}
}

func TestInsertBeaconBadBlob(t *testing.T) {
	indexer, mock, err := NewMockIndexer()
	assert.NoError(t, err)

	ctx := context.Background()
	blob := generateRandomBeaconBadBlob()

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO").WithArgs(
		sqlmock.AnyArg(), sqlmock.AnyArg(), nil, blob.ID, blob.Node, blob.Slot, blob.Epoch,
		blob.BlockRoot, blob.FetchedAt, blob.BeaconImplementation, blob.NodeVersion,
		blob.Location, blob.Network, blob.Index,
	).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = indexer.InsertBeaconBadBlob(ctx, blob)
	assert.NoError(t, err)
}

func TestRemoveBeaconBadBlob(t *testing.T) {
	indexer, mock, err := NewMockIndexer()
	assert.NoError(t, err)

	ctx := context.Background()
	id := "test-id"

	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM").WithArgs(id).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = indexer.RemoveBeaconBadBlob(ctx, id)
	assert.NoError(t, err)
}

func TestCountBeaconBadBlob(t *testing.T) {
	indexer, _, err := NewMockIndexer()
	assert.NoError(t, err)

	ctx := context.Background()

	blob := generateRandomBeaconBadBlob()

	err = indexer.InsertBeaconBadBlob(ctx, blob)
	assert.NoError(t, err)

	filter := &BeaconBadBlobFilter{
		ID: &blob.ID,
	}

	count, err := indexer.CountBeaconBadBlob(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestListBeaconBadBlob(t *testing.T) {
	indexer, mock, err := NewMockIndexer()
	assert.NoError(t, err)

	ctx := context.Background()
	blob := generateRandomBeaconBadBlob()

	err = indexer.InsertBeaconBadBlob(ctx, blob)
	assert.NoError(t, err)

	filter := &BeaconBadBlobFilter{
		ID: &blob.ID,
	}
	page := &PaginationCursor{}

	mock.ExpectQuery("SELECT \\* FROM").WithArgs(filter.ID).WillReturnRows(sqlmock.NewRows([]string{"id", "node"}).AddRow("test-id", "test-node"))

	blobs, err := indexer.ListBeaconBadBlob(ctx, filter, page)
	assert.NoError(t, err)
	assert.Len(t, blobs, 1)
	assert.Equal(t, blob.ID, blobs[0].ID)
	assert.Equal(t, blob.Node, blobs[0].Node)
}

func TestUpdateBeaconBadBlob(t *testing.T) {
	indexer, mock, err := NewMockIndexer()
	assert.NoError(t, err)

	ctx := context.Background()
	blob := generateRandomBeaconBadBlob()

	err = indexer.InsertBeaconBadBlob(ctx, blob)
	assert.NoError(t, err)

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE").WithArgs(
		blob.ID, blob.Node, blob.Slot, blob.Epoch, blob.BlockRoot, blob.FetchedAt,
		blob.BeaconImplementation, blob.NodeVersion, blob.Location, blob.Network, blob.Index,
	).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = indexer.UpdateBeaconBadBlob(ctx, blob)
	assert.NoError(t, err)
}

//nolint:gocyclo // Test is long but manageable
func TestBeaconBadBlobFilters(t *testing.T) {
	t.Run("By random combinations", func(t *testing.T) {
		indexer, _, err := NewMockIndexer()
		if err != nil {
			t.Fatal(err)
		}

		// Add 10000 random beacon blobs
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
			index := int64(rand.Intn(100))

			beaconBlob := &BeaconBadBlob{
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
				Index:                index,
			}

			err = indexer.InsertBeaconBadBlob(context.Background(), beaconBlob)
			if err != nil {
				t.Fatal(err)
			}
		}

		// List beacon blobs with random filters
		for i := 0; i < 5000; i++ {
			var filter BeaconBadBlobFilter

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

			if rand.Intn(2) == 1 {
				filter.AddIndex(uint64(rand.Intn(100)))
			}

			beaconBlobs, err := indexer.ListBeaconBadBlob(context.Background(), &filter, &PaginationCursor{})
			if err != nil {
				t.Fatal(err)
			}

			for _, beaconBlob := range beaconBlobs {
				if filter.ID != nil && *filter.ID != beaconBlob.ID {
					t.Fatalf("expected ID %s, got %s", *filter.ID, beaconBlob.ID)
				}

				if filter.Node != nil && *filter.Node != beaconBlob.Node {
					t.Fatalf("expected Node %s, got %s", *filter.Node, beaconBlob.Node)
				}

				if filter.Before != nil && beaconBlob.FetchedAt.After(*filter.Before) {
					t.Fatalf("expected FetchedAt before %s, got %s", *filter.Before, beaconBlob.FetchedAt)
				}

				if filter.After != nil && beaconBlob.FetchedAt.Before(*filter.After) {
					t.Fatalf("expected FetchedAt after %s, got %s", *filter.After, beaconBlob.FetchedAt)
				}

				if filter.Slot != nil && *filter.Slot != uint64(beaconBlob.Slot) {
					t.Fatalf("expected Slot %d, got %d", *filter.Slot, beaconBlob.Slot)
				}

				if filter.Epoch != nil && *filter.Epoch != uint64(beaconBlob.Epoch) {
					t.Fatalf("expected Epoch %d, got %d", *filter.Epoch, beaconBlob.Epoch)
				}

				if filter.BlockRoot != nil && *filter.BlockRoot != beaconBlob.BlockRoot {
					t.Fatalf("expected BlockRoot %s, got %s", *filter.BlockRoot, beaconBlob.BlockRoot)
				}

				if filter.NodeVersion != nil && *filter.NodeVersion != beaconBlob.NodeVersion {
					t.Fatalf("expected NodeVersion %s, got %s", *filter.NodeVersion, beaconBlob.NodeVersion)
				}

				if filter.Location != nil && *filter.Location != beaconBlob.Location {
					t.Fatalf("expected Location %s, got %s", *filter.Location, beaconBlob.Location)
				}

				if filter.Network != nil && *filter.Network != beaconBlob.Network {
					t.Fatalf("expected Network %s, got %s", *filter.Network, beaconBlob.Network)
				}

				if filter.BeaconImplementation != nil && *filter.BeaconImplementation != beaconBlob.BeaconImplementation {
					t.Fatalf("expected BeaconImplementation %s, got %s", *filter.BeaconImplementation, beaconBlob.BeaconImplementation)
				}

				if filter.Index != nil && *filter.Index != uint64(beaconBlob.Index) {
					t.Fatalf("expected Index %d, got %d", *filter.Index, beaconBlob.Index)
				}
			}
		}
	})
}
func TestBeaconBadBlobIndividualFilters(t *testing.T) {
	t.Run("By individual attributes", func(t *testing.T) {
		indexer, _, err := NewMockIndexer()
		if err != nil {
			t.Fatal(err)
		}

		beaconBlob := generateRandomBeaconBadBlob()

		err = indexer.InsertBeaconBadBlob(context.Background(), beaconBlob)
		if err != nil {
			t.Fatal(err)
		}

		slot := uint64(beaconBlob.Slot)
		epoch := uint64(beaconBlob.Epoch)
		index := uint64(beaconBlob.Index)

		// Test filters individually
		testCases := []struct {
			name   string
			filter BeaconBadBlobFilter
		}{
			{"ID", BeaconBadBlobFilter{ID: &beaconBlob.ID}},
			{"Node", BeaconBadBlobFilter{Node: &beaconBlob.Node}},
			{"Slot", BeaconBadBlobFilter{Slot: &slot}},
			{"Epoch", BeaconBadBlobFilter{Epoch: &epoch}},
			{"BlockRoot", BeaconBadBlobFilter{BlockRoot: &beaconBlob.BlockRoot}},
			{"NodeVersion", BeaconBadBlobFilter{NodeVersion: &beaconBlob.NodeVersion}},
			{"Location", BeaconBadBlobFilter{Location: &beaconBlob.Location}},
			{"Network", BeaconBadBlobFilter{Network: &beaconBlob.Network}},
			{"BeaconImplementation", BeaconBadBlobFilter{BeaconImplementation: &beaconBlob.BeaconImplementation}},
			{"Index", BeaconBadBlobFilter{Index: &index}},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				beaconBlobs, err := indexer.ListBeaconBadBlob(context.Background(), &tc.filter, &PaginationCursor{})
				if err != nil {
					t.Fatal(err)
				}

				if len(beaconBlobs) != 1 {
					t.Fatalf("expected 1 beacon blob, got %d", len(beaconBlobs))
				}

				if beaconBlobs[0].ID != beaconBlob.ID {
					t.Fatalf("expected ID %s, got %s", beaconBlob.ID, beaconBlobs[0].ID)
				}
			})
		}
	})
}
