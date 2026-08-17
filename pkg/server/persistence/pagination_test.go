package persistence

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestParseOrderByAcceptsSortableColumns(t *testing.T) {
	tests := []struct {
		name    string
		orderBy string
		column  string
		desc    bool
	}{
		{name: "column only", orderBy: "slot", column: "slot", desc: false},
		{name: "ascending", orderBy: "fetched_at ASC", column: "fetched_at", desc: false},
		{name: "descending", orderBy: "fetched_at DESC", column: "fetched_at", desc: true},
		{name: "lowercase direction", orderBy: "node desc", column: "node", desc: true},
		{name: "mixed case direction", orderBy: "epoch AsC", column: "epoch", desc: false},
		{name: "uppercase column", orderBy: "BLOCK_ROOT DESC", column: "block_root", desc: true},
		{name: "reserved word column", orderBy: "index ASC", column: "index", desc: false},
		{name: "state root", orderBy: "state_root ASC", column: "state_root", desc: false},
		{name: "block hash", orderBy: "block_hash DESC", column: "block_hash", desc: true},
		{name: "block number", orderBy: "block_number ASC", column: "block_number", desc: false},
		{name: "node version", orderBy: "node_version ASC", column: "node_version", desc: false},
		{name: "beacon implementation", orderBy: "beacon_implementation DESC", column: "beacon_implementation", desc: true},
		{name: "execution implementation", orderBy: "execution_implementation DESC", column: "execution_implementation", desc: true},
		{name: "block extra data", orderBy: "block_extra_data ASC", column: "block_extra_data", desc: false},
		{name: "network", orderBy: "network ASC", column: "network", desc: false},
		{name: "id", orderBy: "id DESC", column: "id", desc: true},
		{name: "extra whitespace", orderBy: "  slot   DESC  ", column: "slot", desc: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			order, err := parseOrderBy(test.orderBy)
			require.NoError(t, err)
			assert.Equal(t, test.column, order.Column.Name)
			assert.Equal(t, test.desc, order.Desc)
			assert.False(t, order.Column.Raw)
		})
	}
}

func TestParseOrderByRejectsEverythingElse(t *testing.T) {
	tests := []struct {
		name    string
		orderBy string
	}{
		{name: "subquery", orderBy: "(SELECT owner FROM distributed_locks LIMIT 1)"},
		{name: "statement injection", orderBy: "1; DROP TABLE"},
		{name: "stacked statement", orderBy: "slot; DROP TABLE beacon_states"},
		{name: "blind boolean", orderBy: "CASE WHEN (SELECT 1) THEN 1 ELSE 2 END"},
		{name: "unknown column", orderBy: "deleted_at DESC"},
		{name: "unknown direction", orderBy: "slot SIDEWAYS"},
		{name: "comma separated columns", orderBy: "slot ASC, node DESC"},
		{name: "trailing comment", orderBy: "slot--"},
		{name: "function call", orderBy: "random()"},
		{name: "qualified column", orderBy: "beacon_states.slot"},
		{name: "column index", orderBy: "1"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := parseOrderBy(test.orderBy)
			assert.Error(t, err)
		})
	}
}

func TestPaginationCursorValidate(t *testing.T) {
	tests := []struct {
		name    string
		cursor  PaginationCursor
		wantErr bool
	}{
		{name: "empty", cursor: PaginationCursor{}, wantErr: false},
		{name: "retention cursor", cursor: PaginationCursor{Limit: 10000, OrderBy: "fetched_at ASC"}, wantErr: false},
		{name: "negative limit", cursor: PaginationCursor{Limit: -1}, wantErr: true},
		{name: "negative offset", cursor: PaginationCursor{Offset: -1}, wantErr: true},
		{name: "injected order by", cursor: PaginationCursor{OrderBy: "(SELECT owner FROM distributed_locks LIMIT 1)"}, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.cursor.Validate()
			if test.wantErr {
				assert.Error(t, err)

				return
			}

			assert.NoError(t, err)
		})
	}
}

func TestApplyOrderBy(t *testing.T) {
	indexer, _, err := NewMockIndexer()
	require.NoError(t, err)

	t.Run("defaults to fetched_at desc", func(t *testing.T) {
		page := &PaginationCursor{}

		query, err := page.ApplyOrderBy(indexer.db.Model(&BeaconState{}))
		require.NoError(t, err)

		sql := query.ToSQL(func(tx *gorm.DB) *gorm.DB { return tx.Find(&[]*BeaconState{}) })
		assert.Contains(t, sql, "ORDER BY `fetched_at` DESC")
	})

	t.Run("quotes the column", func(t *testing.T) {
		page := &PaginationCursor{OrderBy: "index ASC"}

		query, err := page.ApplyOrderBy(indexer.db.Model(&BeaconBadBlob{}))
		require.NoError(t, err)

		sql := query.ToSQL(func(tx *gorm.DB) *gorm.DB { return tx.Find(&[]*BeaconBadBlob{}) })
		assert.Contains(t, sql, "ORDER BY `index`")
	})

	t.Run("rejects injection", func(t *testing.T) {
		page := &PaginationCursor{OrderBy: "(SELECT owner FROM distributed_locks LIMIT 1)"}

		_, err := page.ApplyOrderBy(indexer.db.Model(&BeaconState{}))
		assert.Error(t, err)
	})
}

func TestApplyOffsetLimit(t *testing.T) {
	indexer, _, err := NewMockIndexer()
	require.NoError(t, err)

	tests := []struct {
		name   string
		cursor PaginationCursor
		limit  int
		offset int
	}{
		{name: "zero limit takes the default", cursor: PaginationCursor{}, limit: DefaultPageLimit, offset: 0},
		{name: "negative limit takes the default", cursor: PaginationCursor{Limit: -5}, limit: DefaultPageLimit, offset: 0},
		{name: "limit is honoured", cursor: PaginationCursor{Limit: 25, Offset: 50}, limit: 25, offset: 50},
		{name: "retention limit is not clamped", cursor: PaginationCursor{Limit: MaxPageLimit}, limit: MaxPageLimit, offset: 0},
		{name: "limit is clamped to the max", cursor: PaginationCursor{Limit: MaxPageLimit + 1}, limit: MaxPageLimit, offset: 0},
		{name: "negative offset is clamped", cursor: PaginationCursor{Limit: 10, Offset: -100}, limit: 10, offset: 0},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			query := test.cursor.ApplyOffsetLimit(indexer.db.Model(&BeaconState{}))

			sql := query.ToSQL(func(tx *gorm.DB) *gorm.DB { return tx.Find(&[]*BeaconState{}) })
			assert.Contains(t, sql, "LIMIT "+strconv.Itoa(test.limit))

			if test.offset > 0 {
				assert.Contains(t, sql, "OFFSET "+strconv.Itoa(test.offset))
			} else {
				assert.NotContains(t, sql, "OFFSET")
			}
		})
	}
}
