package persistence

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// The fixture spreads values over two networks so both the network-scoped and the
// empty-network (union) paths have something to disagree about: node-b and v2 exist on both
// networks, everything else is network-specific.
func insertDistinctBeaconBlockFixture(t *testing.T, indexer *Indexer) {
	t.Helper()

	ctx := context.Background()

	rows := []struct {
		network     string
		node        string
		nodeVersion string
		slot        int64
	}{
		{"mainnet", "node-a", "v1", 1},
		{"mainnet", "node-a", "v1", 2},
		{"mainnet", "node-b", "v2", 2},
		{"mainnet", "node-b", "v2", 3},
		{"holesky", "node-b", "v2", 2},
		{"holesky", "node-c", "v3", 4},
	}

	for idx, row := range rows {
		require.NoError(t, indexer.InsertBeaconBlock(ctx, &BeaconBlock{
			ID:                   uuid.New().String(),
			Node:                 row.node,
			Slot:                 row.slot,
			Epoch:                row.slot / 32,
			BlockRoot:            fmt.Sprintf("root-%d", idx),
			FetchedAt:            time.Now(),
			BeaconImplementation: "impl-" + row.network,
			NodeVersion:          row.nodeVersion,
			Location:             fmt.Sprintf("loc-%d", idx),
			Network:              row.network,
		}))
	}
}

func TestDistinctBeaconBlockValuesNetworkScoped(t *testing.T) {
	indexer, _, err := NewMockIndexer()
	require.NoError(t, err)

	insertDistinctBeaconBlockFixture(t, indexer)

	// node and slot ride the loose index scan, node_version is a fallback DISTINCT, and
	// network exercises the prefixed scan collapsing to the filtered network itself.
	results, err := indexer.DistinctBeaconBlockValues(
		context.Background(),
		[]string{KeyNode, KeySlot, KeyNodeVersion, KeyNetwork},
		"mainnet",
	)
	require.NoError(t, err)

	require.ElementsMatch(t, []string{"node-a", "node-b"}, results.Node)
	require.ElementsMatch(t, []uint64{1, 2, 3}, results.Slot)
	require.ElementsMatch(t, []string{"v1", "v2"}, results.NodeVersion)
	require.ElementsMatch(t, []string{"mainnet"}, results.Network)

	// Unrequested fields stay empty.
	require.Empty(t, results.Epoch)
	require.Empty(t, results.BlockRoot)
	require.Empty(t, results.Location)
	require.Empty(t, results.BeaconImplementation)
}

func TestDistinctBeaconBlockValuesEmptyNetwork(t *testing.T) {
	indexer, _, err := NewMockIndexer()
	require.NoError(t, err)

	insertDistinctBeaconBlockFixture(t, indexer)

	// Without a network the loose-scanned fields union the per-network scans, so shared
	// values (node-b, slot 2) must still appear exactly once.
	results, err := indexer.DistinctBeaconBlockValues(
		context.Background(),
		[]string{KeyNode, KeySlot, KeyNodeVersion, KeyNetwork},
		"",
	)
	require.NoError(t, err)

	require.ElementsMatch(t, []string{"node-a", "node-b", "node-c"}, results.Node)
	require.ElementsMatch(t, []uint64{1, 2, 3, 4}, results.Slot)
	require.ElementsMatch(t, []string{"v1", "v2", "v3"}, results.NodeVersion)
	require.ElementsMatch(t, []string{"mainnet", "holesky"}, results.Network)
}

func TestDistinctBeaconBlockValuesUnknownNetwork(t *testing.T) {
	indexer, _, err := NewMockIndexer()
	require.NoError(t, err)

	insertDistinctBeaconBlockFixture(t, indexer)

	results, err := indexer.DistinctBeaconBlockValues(
		context.Background(),
		[]string{KeyNode, KeyNodeVersion, KeyNetwork},
		"nonexistent",
	)
	require.NoError(t, err)

	require.Empty(t, results.Node)
	require.Empty(t, results.NodeVersion)
	require.Empty(t, results.Network)
}

func TestDistinctBeaconBlockValuesUnknownField(t *testing.T) {
	indexer, _, err := NewMockIndexer()
	require.NoError(t, err)

	_, err = indexer.DistinctBeaconBlockValues(
		context.Background(),
		[]string{"fetched_at; DROP TABLE beacon_blocks"},
		"mainnet",
	)
	require.Error(t, err)
}

func TestDistinctExecutionBlockTraceValues(t *testing.T) {
	indexer, _, err := NewMockIndexer()
	require.NoError(t, err)

	ctx := context.Background()

	rows := []struct {
		network     string
		node        string
		blockHash   string
		blockNumber int64
	}{
		{"mainnet", "node-a", "0xaa", 100},
		{"mainnet", "node-b", "0xaa", 100},
		{"mainnet", "node-b", "0xbb", 101},
		{"holesky", "node-c", "0xcc", 7},
	}

	for _, row := range rows {
		require.NoError(t, indexer.InsertExecutionBlockTrace(ctx, &ExecutionBlockTrace{
			ID:                      uuid.New().String(),
			Node:                    row.node,
			FetchedAt:               time.Now(),
			ExecutionImplementation: "geth",
			NodeVersion:             "v1",
			Location:                "loc",
			Network:                 row.network,
			BlockHash:               row.blockHash,
			BlockNumber:             row.blockNumber,
		}))
	}

	// block_hash is loose-scanned through the dedupe unique index, block_number falls back
	// to a plain DISTINCT.
	scoped, err := indexer.DistinctExecutionBlockTraceValues(
		ctx,
		[]string{KeyBlockHash, KeyBlockNumber, KeyNode},
		"mainnet",
	)
	require.NoError(t, err)

	require.ElementsMatch(t, []string{"0xaa", "0xbb"}, scoped.BlockHash)
	require.ElementsMatch(t, []int64{100, 101}, scoped.BlockNumber)
	require.ElementsMatch(t, []string{"node-a", "node-b"}, scoped.Node)

	all, err := indexer.DistinctExecutionBlockTraceValues(
		ctx,
		[]string{KeyBlockHash, KeyBlockNumber, KeyNetwork},
		"",
	)
	require.NoError(t, err)

	require.ElementsMatch(t, []string{"0xaa", "0xbb", "0xcc"}, all.BlockHash)
	require.ElementsMatch(t, []int64{7, 100, 101}, all.BlockNumber)
	require.ElementsMatch(t, []string{"mainnet", "holesky"}, all.Network)
}
