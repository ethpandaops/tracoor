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

	// node, slot, node_version and beacon_implementation ride the loose index scan, epoch is
	// a fallback DISTINCT, and network exercises the prefixed scan collapsing to the filtered
	// network itself.
	results, err := indexer.DistinctBeaconBlockValues(
		context.Background(),
		[]string{KeyNode, KeySlot, KeyNodeVersion, KeyBeaconImplementation, KeyEpoch, KeyNetwork},
		"mainnet",
	)
	require.NoError(t, err)

	require.ElementsMatch(t, []string{"node-a", "node-b"}, results.Node)
	require.ElementsMatch(t, []uint64{1, 2, 3}, results.Slot)
	require.ElementsMatch(t, []string{"v1", "v2"}, results.NodeVersion)
	require.ElementsMatch(t, []string{"impl-mainnet"}, results.BeaconImplementation)
	require.ElementsMatch(t, []uint64{0}, results.Epoch)
	require.ElementsMatch(t, []string{"mainnet"}, results.Network)

	// Unrequested fields stay empty.
	require.Empty(t, results.BlockRoot)
	require.Empty(t, results.Location)
}

func TestDistinctBeaconBlockValuesEmptyNetwork(t *testing.T) {
	indexer, _, err := NewMockIndexer()
	require.NoError(t, err)

	insertDistinctBeaconBlockFixture(t, indexer)

	// Without a network the loose-scanned fields union the per-network scans, so shared
	// values (node-b, slot 2) must still appear exactly once.
	results, err := indexer.DistinctBeaconBlockValues(
		context.Background(),
		[]string{KeyNode, KeySlot, KeyNodeVersion, KeyBeaconImplementation, KeyNetwork},
		"",
	)
	require.NoError(t, err)

	require.ElementsMatch(t, []string{"node-a", "node-b", "node-c"}, results.Node)
	require.ElementsMatch(t, []uint64{1, 2, 3, 4}, results.Slot)
	require.ElementsMatch(t, []string{"v1", "v2", "v3"}, results.NodeVersion)
	require.ElementsMatch(t, []string{"impl-mainnet", "impl-holesky"}, results.BeaconImplementation)
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
			ExecutionImplementation: "geth-" + row.network,
			NodeVersion:             "v1-" + row.node,
			Location:                "loc",
			Network:                 row.network,
			BlockHash:               row.blockHash,
			BlockNumber:             row.blockNumber,
		}))
	}

	// block_hash is loose-scanned through the dedupe unique index, node_version and
	// execution_implementation through their (network, <field>) indexes, and block_number
	// falls back to a plain DISTINCT.
	scoped, err := indexer.DistinctExecutionBlockTraceValues(
		ctx,
		[]string{KeyBlockHash, KeyBlockNumber, KeyNode, KeyNodeVersion, KeyExecutionImplementation},
		"mainnet",
	)
	require.NoError(t, err)

	require.ElementsMatch(t, []string{"0xaa", "0xbb"}, scoped.BlockHash)
	require.ElementsMatch(t, []int64{100, 101}, scoped.BlockNumber)
	require.ElementsMatch(t, []string{"node-a", "node-b"}, scoped.Node)
	require.ElementsMatch(t, []string{"v1-node-a", "v1-node-b"}, scoped.NodeVersion)
	require.ElementsMatch(t, []string{"geth-mainnet"}, scoped.ExecutionImplementation)

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

// TestDistinctLooseScanFieldsAreIndexed pins every loose-scanned field to an index that
// leads (network, <field>). The loose scan is only cheap because each recursion step is one
// index seek; on an unindexed column the same CTE degrades to a full scan per distinct value,
// which is worse than the plain DISTINCT it replaced. Network itself only needs an index
// leading network.
func TestDistinctLooseScanFieldsAreIndexed(t *testing.T) {
	indexer, _, err := NewMockIndexer()
	require.NoError(t, err)

	tables := []distinctTable{
		beaconStateDistinct,
		beaconBlockDistinct,
		executionPayloadEnvelopeDistinct,
		beaconBadBlockDistinct,
		beaconBadBlobDistinct,
		executionBlockTraceDistinct,
		executionBadBlockDistinct,
	}

	for _, table := range tables {
		leading := indexLeadingColumns(t, indexer, table.name)

		for field, strategy := range table.fields {
			if strategy != distinctLooseScan {
				continue
			}

			want := []string{KeyNetwork, field}
			if field == KeyNetwork {
				want = []string{KeyNetwork}
			}

			require.True(t, hasLeadingIndex(leading, want), "%s.%s is loose-scanned but no index leads %v", table.name, field, want)
		}
	}
}

// indexLeadingColumns returns each index's columns in key order, unique and plain alike.
func indexLeadingColumns(t *testing.T, indexer *Indexer, table string) [][]string {
	t.Helper()

	var names []string

	require.NoError(t, indexer.db.Raw(
		"SELECT name FROM sqlite_master WHERE type = 'index' AND tbl_name = ?", table,
	).Scan(&names).Error)

	out := make([][]string, 0, len(names))

	for _, name := range names {
		var cols []string

		require.NoError(t, indexer.db.Raw(
			"SELECT name FROM pragma_index_info(?) ORDER BY seqno", name,
		).Scan(&cols).Error)

		out = append(out, cols)
	}

	return out
}

// hasLeadingIndex reports whether any index's key starts with want, in order.
func hasLeadingIndex(leading [][]string, want []string) bool {
	for _, cols := range leading {
		if len(cols) >= len(want) && slicesEqual(cols[:len(want)], want) {
			return true
		}
	}

	return false
}

func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

// TestListFilterAndSortColumnsAreIndexed pins every column the web UI's tables filter or sort
// the high-volume tables by to an index leading (network, <column>). The UI always scopes a
// list to one network, so with that index a filtered count is a range scan and a sorted page
// reads one page of the index; without it, both read every row of the network. The bad-block
// and bad-blob tables are small enough to scan.
func TestListFilterAndSortColumnsAreIndexed(t *testing.T) {
	indexer, _, err := NewMockIndexer()
	require.NoError(t, err)

	beacon := []string{"fetched_at", "node", "slot", "epoch", "node_version", "beacon_implementation"}

	columns := map[string][]string{
		"beacon_states":               append([]string{"state_root"}, beacon...),
		"beacon_blocks":               append([]string{"block_root"}, beacon...),
		"execution_payload_envelopes": append([]string{"block_root"}, beacon...),
		"execution_block_traces": {
			"fetched_at", "node", "block_hash", "block_number", "node_version", "execution_implementation",
		},
	}

	for table, cols := range columns {
		leading := indexLeadingColumns(t, indexer, table)

		for _, col := range cols {
			want := []string{KeyNetwork, col}
			require.True(t, hasLeadingIndex(leading, want), "%s is filtered or sorted by %s but no index leads %v", table, col, want)
		}
	}
}
