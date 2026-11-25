package persistence

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestBlobInsertAndGet(t *testing.T) {
	indexer, _, err := NewMockIndexer()
	require.NoError(t, err)

	ctx := context.Background()

	blob := &Blob{
		Kind:            "beacon_state",
		Network:         "mainnet",
		DedupKey:        "100/0xdeadbeef",
		ContentHash:     generateRandomString(64),
		Location:        "states/100-0xdeadbe.ssz.zst",
		ContentEncoding: "zstd",
		RawSize:         100,
		CompressedSize:  10,
	}

	winner, err := indexer.InsertBlob(ctx, blob)
	require.NoError(t, err)
	assert.Equal(t, blob.ContentHash, winner.ContentHash)
	assert.Equal(t, BlobStateReady, winner.State)

	got, err := indexer.GetBlob(ctx, blob.Kind, blob.Network, blob.DedupKey)
	require.NoError(t, err)
	assert.Equal(t, blob.ContentHash, got.ContentHash)
	assert.Equal(t, blob.Location, got.Location)
	assert.Equal(t, int64(100), got.RawSize)
}

func TestBlobInsertReturnsTheWinner(t *testing.T) {
	indexer, _, err := NewMockIndexer()
	require.NoError(t, err)

	ctx := context.Background()

	first := &Blob{
		Kind:        "beacon_block",
		Network:     "mainnet",
		DedupKey:    "200/0xaaaa",
		ContentHash: generateRandomString(64),
		Location:    "blocks/first",
	}

	_, err = indexer.InsertBlob(ctx, first)
	require.NoError(t, err)

	loser := &Blob{
		Kind:        first.Kind,
		Network:     first.Network,
		DedupKey:    first.DedupKey,
		ContentHash: generateRandomString(64),
		Location:    "blocks/second",
	}

	winner, err := indexer.InsertBlob(ctx, loser)
	require.NoError(t, err)

	// The loser must see the row that is actually in the table so it can compare hashes.
	assert.Equal(t, first.ContentHash, winner.ContentHash)
	assert.Equal(t, first.Location, winner.Location)
}

func TestBlobGetSkipsTombstonedBlobs(t *testing.T) {
	indexer, _, err := NewMockIndexer()
	require.NoError(t, err)

	ctx := context.Background()

	blob := &Blob{
		Kind:        "execution_block_trace",
		Network:     "mainnet",
		DedupKey:    "300/0xbbbb",
		ContentHash: generateRandomString(64),
		Location:    "traces/one",
		State:       BlobStateDeleting,
	}

	_, err = indexer.InsertBlob(ctx, blob)
	require.NoError(t, err)

	_, err = indexer.GetBlob(ctx, blob.Kind, blob.Network, blob.DedupKey)
	require.ErrorIs(t, err, ErrBlobNotFound)
}

func TestBlobGetMissing(t *testing.T) {
	indexer, _, err := NewMockIndexer()
	require.NoError(t, err)

	_, err = indexer.GetBlob(context.Background(), "beacon_state", "mainnet", "missing")
	require.ErrorIs(t, err, ErrBlobNotFound)
}

// A tombstoned blob stays a candidate: its object delete may have failed, and nothing else
// ever drives a retry for a row in the deleting state.
func TestListCollectableBlobsOffersTombstonedBlobs(t *testing.T) {
	indexer, _, err := NewMockIndexer()
	require.NoError(t, err)

	ctx := context.Background()

	blob := &Blob{
		Kind:        "beacon_state",
		Network:     "mainnet",
		DedupKey:    "500/0xcccc",
		ContentHash: generateRandomString(64),
		Location:    "states/tombstoned",
		CreatedAt:   time.Now().UTC().Add(-time.Hour),
	}

	_, err = indexer.InsertBlob(ctx, blob)
	require.NoError(t, err)

	cutoff := time.Now().UTC()

	candidates, err := indexer.ListCollectableBlobs(ctx, cutoff, 10, 0)
	require.NoError(t, err)
	require.Len(t, candidates, 1)
	assert.Equal(t, BlobStateReady, candidates[0].State)

	outcome, err := indexer.TombstoneBlob(ctx, candidates[0])
	require.NoError(t, err)
	require.Equal(t, BlobTombstoneMarked, outcome)

	candidates, err = indexer.ListCollectableBlobs(ctx, cutoff, 10, 0)
	require.NoError(t, err)
	require.Len(t, candidates, 1, "a tombstoned blob is offered again so its object delete can be retried")
	assert.Equal(t, BlobStateDeleting, candidates[0].State)
	assert.Equal(t, blob.Generation, candidates[0].Generation, "the generation guard carries through the retry")
}

// The candidate scan runs on every retention pass and nearly every blob is referenced, so it
// must read the partial index of unreferenced blobs rather than walk the table. A predicate
// that drifts from the index's own makes the planner unable to use it, which only shows in
// the plan.
func TestListCollectableBlobsReadsTheUnreferencedIndex(t *testing.T) {
	indexer, _, err := NewMockIndexer()
	require.NoError(t, err)

	var ddl string

	require.NoError(t, indexer.db.Raw(
		"SELECT sql FROM sqlite_master WHERE type = 'index' AND name = ?", blobUnreferencedIndex,
	).Scan(&ddl).Error)
	assert.Contains(t, ddl, "WHERE ref_count <= 0")

	// The shape of a live table: every blob referenced but a couple, with statistics, which is
	// what the planner decides on.
	ctx := context.Background()

	for idx := range 2000 {
		refs := int64(1)
		if idx%1000 == 999 {
			refs = 0
		}

		_, err = indexer.InsertBlob(ctx, &Blob{
			Kind:        "beacon_state",
			Network:     "mainnet",
			DedupKey:    fmt.Sprintf("%d/0xaaaa", idx),
			ContentHash: generateRandomString(64),
			Location:    fmt.Sprintf("states/%d", idx),
			RefCount:    refs,
			CreatedAt:   time.Now().UTC().Add(-time.Duration(2000-idx) * time.Minute),
		})
		require.NoError(t, err)
	}

	require.NoError(t, indexer.db.Exec("ANALYZE").Error)

	candidates, err := indexer.ListCollectableBlobs(ctx, time.Now().UTC(), 100, 0)
	require.NoError(t, err)
	require.Len(t, candidates, 2)

	for _, offset := range []int{0, 100} {
		stmt := collectableBlobsQuery(indexer.db.Session(&gorm.Session{DryRun: true}), time.Now(), 100, offset).
			Scan(&[]*BlobCandidate{}).Statement

		var plan []struct {
			ID      int
			Parent  int
			Notused int
			Detail  string
		}

		require.NoError(t, indexer.db.Raw("EXPLAIN QUERY PLAN "+stmt.SQL.String(), stmt.Vars...).Scan(&plan).Error)

		details := make([]string, 0, len(plan))
		for _, step := range plan {
			details = append(details, step.Detail)
		}

		joined := strings.Join(details, "; ")
		assert.Contains(t, joined, blobUnreferencedIndex, "offset %d: %s", offset, joined)
		assert.NotContains(t, joined, "TEMP B-TREE", "offset %d sorts instead of reading the index in order: %s", offset, joined)
	}
}
