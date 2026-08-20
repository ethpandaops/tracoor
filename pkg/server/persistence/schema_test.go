package persistence

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// expectedIndexes is the full index set each artifact table is allowed to carry: the dedupe
// unique index, the three fetched_at orderings the list, filter and retention queries use, and
// on beacon_blocks the (network, epoch) pair the promotion service walks.
var expectedIndexes = map[string][]string{
	"beacon_states": {
		"ux_beacon_states_dedupe",
		"ix_beacon_states_network_node_fetched_at",
		"ix_beacon_states_network_fetched_at",
		"ix_beacon_states_fetched_at",
	},
	"beacon_blocks": {
		"ux_beacon_blocks_dedupe",
		"ix_beacon_blocks_network_node_fetched_at",
		"ix_beacon_blocks_network_fetched_at",
		"ix_beacon_blocks_fetched_at",
		"ix_beacon_blocks_network_epoch",
	},
	"execution_payload_envelopes": {
		"ux_execution_payload_envelopes_dedupe",
		"ix_execution_payload_envelopes_network_node_fetched_at",
		"ix_execution_payload_envelopes_network_fetched_at",
		"ix_execution_payload_envelopes_fetched_at",
	},
	"beacon_bad_blocks": {
		"ux_beacon_bad_blocks_dedupe",
		"ix_beacon_bad_blocks_network_node_fetched_at",
		"ix_beacon_bad_blocks_network_fetched_at",
		"ix_beacon_bad_blocks_fetched_at",
	},
	"beacon_bad_blobs": {
		"ux_beacon_bad_blobs_dedupe",
		"ix_beacon_bad_blobs_network_node_fetched_at",
		"ix_beacon_bad_blobs_network_fetched_at",
		"ix_beacon_bad_blobs_fetched_at",
	},
	"execution_block_traces": {
		"ux_execution_block_traces_dedupe",
		"ix_execution_block_traces_network_node_fetched_at",
		"ix_execution_block_traces_network_fetched_at",
		"ix_execution_block_traces_fetched_at",
	},
	"execution_bad_blocks": {
		"ux_execution_bad_blocks_dedupe",
		"ix_execution_bad_blocks_network_node_fetched_at",
		"ix_execution_bad_blocks_network_fetched_at",
		"ix_execution_bad_blocks_fetched_at",
	},
	"blobs": {
		"ix_blobs_kind_network_content_hash",
		"ix_blobs_state_created_at",
	},
	"payload_divergences": {
		"ix_payload_divergences_observed_at",
		"ix_payload_divergences_network_kind_observed_at",
		"ix_payload_divergences_network_kind_dedup_key",
	},
	"distributed_locks": {
		"ix_distributed_locks_expires_at",
	},
	"permanent_blocks": {
		"ux_permanent_blocks_block_root_network",
	},
}

func namedIndexes(t *testing.T, indexer *Indexer, table string) []string {
	t.Helper()

	var names []string

	// Auto-generated names (sqlite_autoindex_*) back the primary keys and are not part of
	// the declared index set.
	err := indexer.db.Raw(
		"SELECT name FROM sqlite_master WHERE type = 'index' AND tbl_name = ? AND name NOT LIKE 'sqlite_autoindex%'",
		table,
	).Scan(&names).Error
	require.NoError(t, err)

	return names
}

func tableColumns(t *testing.T, indexer *Indexer, table string) []string {
	t.Helper()

	var names []string

	err := indexer.db.Raw("SELECT name FROM pragma_table_info(?)", table).Scan(&names).Error
	require.NoError(t, err)

	return names
}

func TestAutoMigrateSchema(t *testing.T) {
	indexer, _, err := NewMockIndexer()
	require.NoError(t, err)

	for table, expected := range expectedIndexes {
		got := namedIndexes(t, indexer, table)

		assert.ElementsMatch(t, expected, got, "unexpected index set on %s", table)
	}
}

func TestAutoMigrateHasNoGormModelColumns(t *testing.T) {
	indexer, _, err := NewMockIndexer()
	require.NoError(t, err)

	for table := range expectedIndexes {
		columns := tableColumns(t, indexer, table)

		require.NotEmpty(t, columns, "table %s was not created", table)

		for _, column := range []string{"created_at", "updated_at", "deleted_at"} {
			if table == "blobs" && column == "created_at" {
				continue
			}

			assert.NotContains(t, columns, column, "table %s still carries %s", table, column)
		}
	}
}

func TestAutoMigrateArtifactContentColumns(t *testing.T) {
	indexer, _, err := NewMockIndexer()
	require.NoError(t, err)

	artifacts := []string{
		"beacon_states",
		"beacon_blocks",
		"execution_payload_envelopes",
		"beacon_bad_blocks",
		"beacon_bad_blobs",
		"execution_block_traces",
		"execution_bad_blocks",
	}

	for _, table := range artifacts {
		columns := tableColumns(t, indexer, table)

		assert.Contains(t, columns, "content_hash", "table %s", table)
		assert.Contains(t, columns, "verified_at", "table %s", table)
		assert.Contains(t, columns, "content_matched_at", "table %s", table)
	}
}
