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

func generateRandomBeaconState() *BeaconState {
	return &BeaconState{
		ID:                   generateRandomString(10),
		Node:                 generateRandomString(5),
		Slot:                 generateRandomInt64(),
		Epoch:                generateRandomInt64(),
		StateRoot:            generateRandomString(32),
		FetchedAt:            time.Now(),
		BeaconImplementation: generateRandomString(15),
		NodeVersion:          generateRandomString(8),
		Location:             generateRandomString(10),
		Network:              generateRandomString(5),
	}
}

func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

func generateRandomInt64() int64 {
	return rand.Int63()
}

func TestInsertBeaconState(t *testing.T) {
	indexer, mock, err := newMockIndexer()
	assert.NoError(t, err)

	ctx := context.Background()
	state := generateRandomBeaconState()

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO").WithArgs(
		sqlmock.AnyArg(), sqlmock.AnyArg(), nil, state.ID, state.Node, state.Slot, state.Epoch,
		state.StateRoot, state.FetchedAt, state.BeaconImplementation, state.NodeVersion,
		state.Location, state.Network,
	).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = indexer.InsertBeaconState(ctx, state)
	assert.NoError(t, err)
}

func TestRemoveBeaconState(t *testing.T) {
	indexer, mock, err := newMockIndexer()
	assert.NoError(t, err)

	ctx := context.Background()
	id := "test-id"

	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM").WithArgs(id).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = indexer.RemoveBeaconState(ctx, id)
	assert.NoError(t, err)
}

func TestCountBeaconState(t *testing.T) {
	indexer, _, err := newMockIndexer()
	assert.NoError(t, err)

	ctx := context.Background()

	state := generateRandomBeaconState()

	err = indexer.InsertBeaconState(ctx, state)
	assert.NoError(t, err)

	filter := &BeaconStateFilter{
		ID: &state.ID,
	}

	count, err := indexer.CountBeaconState(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestListBeaconState(t *testing.T) {
	indexer, mock, err := newMockIndexer()
	assert.NoError(t, err)

	ctx := context.Background()
	state := generateRandomBeaconState()

	err = indexer.InsertBeaconState(ctx, state)
	assert.NoError(t, err)

	filter := &BeaconStateFilter{
		ID: &state.ID,
	}
	page := &PaginationCursor{}

	mock.ExpectQuery("SELECT \\* FROM").WithArgs(filter.ID).WillReturnRows(sqlmock.NewRows([]string{"id", "node"}).AddRow("test-id", "test-node"))

	states, err := indexer.ListBeaconState(ctx, filter, page)
	assert.NoError(t, err)
	assert.Len(t, states, 1)
	assert.Equal(t, state.ID, states[0].ID)
	assert.Equal(t, state.Node, states[0].Node)
}

func TestUpdateBeaconState(t *testing.T) {
	indexer, mock, err := newMockIndexer()
	assert.NoError(t, err)

	ctx := context.Background()
	state := generateRandomBeaconState()

	err = indexer.InsertBeaconState(ctx, state)
	assert.NoError(t, err)

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE").WithArgs(
		state.ID, state.Node, state.Slot, state.Epoch, state.StateRoot, state.FetchedAt,
		state.BeaconImplementation, state.NodeVersion, state.Location, state.Network,
	).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = indexer.UpdateBeaconState(ctx, state)
	assert.NoError(t, err)
}

func TestBeaconStateFilters(t *testing.T) {
	t.Run("By random combinations", func(t *testing.T) {
		indexer, _, err := newMockIndexer()
		if err != nil {
			t.Fatal(err)
		}

		// Add 10000 random beacon states
		for i := 0; i < 10000; i++ {
			id := uuid.New().String()
			node := fmt.Sprintf("node-%d", i)
			slot := int64(rand.Intn(1000))
			epoch := int64(rand.Intn(100))
			stateRoot := fmt.Sprintf("stateRoot-%d", rand.Intn(100))
			nodeVersion := fmt.Sprintf("version%d", rand.Intn(10))
			location := fmt.Sprintf("location%d", rand.Intn(10))
			network := fmt.Sprintf("network%d", rand.Intn(10))
			beaconImplementation := fmt.Sprintf("implementation%d", rand.Intn(10))

			beaconState := &BeaconState{
				ID:                   id,
				Node:                 node,
				Slot:                 slot,
				Epoch:                epoch,
				StateRoot:            stateRoot,
				FetchedAt:            time.Now(),
				NodeVersion:          nodeVersion,
				Location:             location,
				Network:              network,
				BeaconImplementation: beaconImplementation,
			}

			err = indexer.InsertBeaconState(context.Background(), beaconState)
			if err != nil {
				t.Fatal(err)
			}
		}

		// List beacon states with random filters
		for i := 0; i < 5000; i++ {
			var filter BeaconStateFilter

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
				filter.AddStateRoot(fmt.Sprintf("stateRoot-%d", rand.Intn(100)))
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

			beaconStates, err := indexer.ListBeaconState(context.Background(), &filter, &PaginationCursor{})
			if err != nil {
				t.Fatal(err)
			}

			for _, beaconState := range beaconStates {
				if filter.ID != nil && *filter.ID != beaconState.ID {
					t.Fatalf("expected ID %s, got %s", *filter.ID, beaconState.ID)
				}

				if filter.Node != nil && *filter.Node != beaconState.Node {
					t.Fatalf("expected Node %s, got %s", *filter.Node, beaconState.Node)
				}

				if filter.Before != nil && beaconState.FetchedAt.After(*filter.Before) {
					t.Fatalf("expected FetchedAt before %s, got %s", *filter.Before, beaconState.FetchedAt)
				}

				if filter.After != nil && beaconState.FetchedAt.Before(*filter.After) {
					t.Fatalf("expected FetchedAt after %s, got %s", *filter.After, beaconState.FetchedAt)
				}

				if filter.Slot != nil && *filter.Slot != uint64(beaconState.Slot) {
					t.Fatalf("expected Slot %d, got %d", *filter.Slot, beaconState.Slot)
				}

				if filter.Epoch != nil && *filter.Epoch != uint64(beaconState.Epoch) {
					t.Fatalf("expected Epoch %d, got %d", *filter.Epoch, beaconState.Epoch)
				}

				if filter.StateRoot != nil && *filter.StateRoot != beaconState.StateRoot {
					t.Fatalf("expected StateRoot %s, got %s", *filter.StateRoot, beaconState.StateRoot)
				}

				if filter.NodeVersion != nil && *filter.NodeVersion != beaconState.NodeVersion {
					t.Fatalf("expected NodeVersion %s, got %s", *filter.NodeVersion, beaconState.NodeVersion)
				}

				if filter.Location != nil && *filter.Location != beaconState.Location {
					t.Fatalf("expected Location %s, got %s", *filter.Location, beaconState.Location)
				}

				if filter.Network != nil && *filter.Network != beaconState.Network {
					t.Fatalf("expected Network %s, got %s", *filter.Network, beaconState.Network)
				}

				if filter.BeaconImplementation != nil && *filter.BeaconImplementation != beaconState.BeaconImplementation {
					t.Fatalf("expected BeaconImplementation %s, got %s", *filter.BeaconImplementation, beaconState.BeaconImplementation)
				}
			}
		}
	})
}
func TestBeaconStateIndividualFilters(t *testing.T) {
	t.Run("By individual attributes", func(t *testing.T) {
		indexer, _, err := newMockIndexer()
		if err != nil {
			t.Fatal(err)
		}

		beaconState := generateRandomBeaconState()

		err = indexer.InsertBeaconState(context.Background(), beaconState)
		if err != nil {
			t.Fatal(err)
		}

		slot := uint64(beaconState.Slot)
		epoch := uint64(beaconState.Epoch)

		// Test filters individually
		testCases := []struct {
			name   string
			filter BeaconStateFilter
		}{
			{"ID", BeaconStateFilter{ID: &beaconState.ID}},
			{"Node", BeaconStateFilter{Node: &beaconState.Node}},
			{"Slot", BeaconStateFilter{Slot: &slot}},
			{"Epoch", BeaconStateFilter{Epoch: &epoch}},
			{"StateRoot", BeaconStateFilter{StateRoot: &beaconState.StateRoot}},
			{"NodeVersion", BeaconStateFilter{NodeVersion: &beaconState.NodeVersion}},
			{"Location", BeaconStateFilter{Location: &beaconState.Location}},
			{"Network", BeaconStateFilter{Network: &beaconState.Network}},
			{"BeaconImplementation", BeaconStateFilter{BeaconImplementation: &beaconState.BeaconImplementation}},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				beaconStates, err := indexer.ListBeaconState(context.Background(), &tc.filter, &PaginationCursor{})
				if err != nil {
					t.Fatal(err)
				}

				if len(beaconStates) != 1 {
					t.Fatalf("expected 1 beacon state, got %d", len(beaconStates))
				}

				if beaconStates[0].ID != beaconState.ID {
					t.Fatalf("expected ID %s, got %s", beaconState.ID, beaconStates[0].ID)
				}
			})
		}
	})
}
