package persistence

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
