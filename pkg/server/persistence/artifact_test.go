package persistence

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func newTestState(network string, slot int64, root, node, location, contentHash string, fetchedAt time.Time) *BeaconState {
	return &BeaconState{
		ID:          generateRandomString(10),
		Node:        node,
		Slot:        slot,
		StateRoot:   root,
		Network:     network,
		Location:    location,
		ContentHash: contentHash,
		FetchedAt:   fetchedAt,
	}
}

func newTestBlob(network, dedupKey, contentHash, location string) *Blob {
	return &Blob{
		Kind:        KindBeaconState,
		Network:     network,
		DedupKey:    dedupKey,
		ContentHash: contentHash,
		Location:    location,
		State:       BlobStateReady,
		CreatedAt:   time.Now(),
	}
}

// The seven artifact filters bind a native timestamp; a formatted string compared
// lexicographically against whatever the driver stored is only accidentally right.
func TestBeaconStateBeforeFilterRespectsSubDayBoundaries(t *testing.T) {
	indexer, _, err := NewMockIndexer()
	require.NoError(t, err)

	ctx := context.Background()
	network := generateRandomString(6)
	now := time.Now().UTC()

	old := newTestState(network, 1, "root-old", "node-a", "loc-old", "", now.Add(-2*time.Hour))
	fresh := newTestState(network, 2, "root-new", "node-a", "loc-new", "", now.Add(-1*time.Minute))

	require.NoError(t, indexer.InsertBeaconState(ctx, old))
	require.NoError(t, indexer.InsertBeaconState(ctx, fresh))

	filter := &BeaconStateFilter{}
	filter.AddNetwork(network)
	filter.AddBefore(now.Add(-time.Hour))

	rows, err := indexer.ListBeaconState(ctx, filter, &PaginationCursor{Limit: 10})
	require.NoError(t, err)
	require.Len(t, rows, 1, "only the row fetched more than an hour ago is before the cutoff")
	require.Equal(t, old.ID, rows[0].ID)

	// And the same boundary within a single second.
	tight := &BeaconStateFilter{}
	tight.AddNetwork(network)
	tight.AddBefore(fresh.FetchedAt.Add(-500 * time.Millisecond))

	rows, err = indexer.ListBeaconState(ctx, tight, &PaginationCursor{Limit: 10})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, old.ID, rows[0].ID)
}

func TestInsertArtifactLinksExactlyOnce(t *testing.T) {
	indexer, _, err := NewMockIndexer()
	require.NoError(t, err)

	ctx := context.Background()
	network := generateRandomString(6)
	dedupKey := "10/root-a"
	hash := generateRandomString(64)
	location := "states/root-a.ssz"

	_, err = indexer.InsertBlob(ctx, newTestBlob(network, dedupKey, hash, location))
	require.NoError(t, err)

	params := func(node string) *InsertArtifactParams {
		return &InsertArtifactParams{
			Row:             newTestState(network, 10, "root-a", node, location, hash, time.Now()),
			ConflictColumns: []string{KeyNetwork, KeySlot, KeyStateRoot, KeyNode},
			Kind:            KindBeaconState,
			Network:         network,
			DedupKey:        dedupKey,
			ContentHash:     hash,
			Location:        location,
		}
	}

	outcome, err := indexer.InsertArtifact(ctx, params("node-a"))
	require.NoError(t, err)
	require.Equal(t, ArtifactInsertLinked, outcome)

	blob, err := indexer.GetBlob(ctx, KindBeaconState, network, dedupKey)
	require.NoError(t, err)
	require.Equal(t, int64(1), blob.RefCount)

	t.Run("a duplicate observation does not take a second reference", func(t *testing.T) {
		outcome, err := indexer.InsertArtifact(ctx, params("node-a"))
		require.NoError(t, err)
		require.Equal(t, ArtifactInsertDuplicate, outcome)

		blob, err := indexer.GetBlob(ctx, KindBeaconState, network, dedupKey)
		require.NoError(t, err)
		require.Equal(t, int64(1), blob.RefCount)
	})

	t.Run("a second node observing the same payload takes its own reference", func(t *testing.T) {
		outcome, err := indexer.InsertArtifact(ctx, params("node-b"))
		require.NoError(t, err)
		require.Equal(t, ArtifactInsertLinked, outcome)

		blob, err := indexer.GetBlob(ctx, KindBeaconState, network, dedupKey)
		require.NoError(t, err)
		require.Equal(t, int64(2), blob.RefCount)
	})
}

func TestInsertArtifactRejectsAnUnlinkablePayload(t *testing.T) {
	indexer, _, err := NewMockIndexer()
	require.NoError(t, err)

	ctx := context.Background()
	network := generateRandomString(6)

	t.Run("no blob at all", func(t *testing.T) {
		row := newTestState(network, 20, "root-missing", "node-a", "states/missing.ssz", generateRandomString(64), time.Now())

		outcome, err := indexer.InsertArtifact(ctx, &InsertArtifactParams{
			Row:             row,
			ConflictColumns: []string{KeyNetwork, KeySlot, KeyStateRoot, KeyNode},
			Kind:            KindBeaconState,
			Network:         network,
			DedupKey:        "20/root-missing",
			ContentHash:     row.ContentHash,
			Location:        row.Location,
		})
		require.NoError(t, err)
		require.Equal(t, ArtifactInsertUnlinkable, outcome)

		filter := &BeaconStateFilter{}
		filter.AddID(row.ID)

		rows, err := indexer.ListBeaconState(ctx, filter, &PaginationCursor{Limit: 1})
		require.NoError(t, err)
		require.Empty(t, rows, "a rejected write must leave nothing behind")
	})

	t.Run("tombstoned blob", func(t *testing.T) {
		dedupKey := "30/root-dying"
		hash := generateRandomString(64)
		location := "states/root-dying.ssz"

		blob := newTestBlob(network, dedupKey, hash, location)
		blob.State = BlobStateDeleting

		_, err := indexer.InsertBlob(ctx, blob)
		require.NoError(t, err)

		row := newTestState(network, 30, "root-dying", "node-a", location, hash, time.Now())

		outcome, err := indexer.InsertArtifact(ctx, &InsertArtifactParams{
			Row:             row,
			ConflictColumns: []string{KeyNetwork, KeySlot, KeyStateRoot, KeyNode},
			Kind:            KindBeaconState,
			Network:         network,
			DedupKey:        dedupKey,
			ContentHash:     hash,
			Location:        location,
		})
		require.NoError(t, err)
		require.Equal(t, ArtifactInsertUnlinkable, outcome)
	})
}

// A copy stored beside the canonical payload because its bytes disagreed is a legitimate row
// that owns its object, not a failed link.
func TestInsertArtifactAcceptsADivergentCopy(t *testing.T) {
	indexer, _, err := NewMockIndexer()
	require.NoError(t, err)

	ctx := context.Background()
	network := generateRandomString(6)
	dedupKey := "40/root-d"

	_, err = indexer.InsertBlob(ctx, newTestBlob(network, dedupKey, generateRandomString(64), "states/canonical.ssz"))
	require.NoError(t, err)

	row := newTestState(network, 40, "root-d", "node-liar", "states/divergent.ssz", generateRandomString(64), time.Now())

	outcome, err := indexer.InsertArtifact(ctx, &InsertArtifactParams{
		Row:             row,
		ConflictColumns: []string{KeyNetwork, KeySlot, KeyStateRoot, KeyNode},
		Kind:            KindBeaconState,
		Network:         network,
		DedupKey:        dedupKey,
		ContentHash:     row.ContentHash,
		Location:        row.Location,
	})
	require.NoError(t, err)
	require.Equal(t, ArtifactInsertUnlinked, outcome)

	blob, err := indexer.GetBlob(ctx, KindBeaconState, network, dedupKey)
	require.NoError(t, err)
	require.Equal(t, int64(0), blob.RefCount, "a divergent copy references nothing")
}

func TestDeleteArtifactsReleasesReferences(t *testing.T) {
	indexer, _, err := NewMockIndexer()
	require.NoError(t, err)

	ctx := context.Background()
	network := generateRandomString(6)
	dedupKey := "50/root-e"
	hash := generateRandomString(64)
	location := "states/root-e.ssz"

	_, err = indexer.InsertBlob(ctx, newTestBlob(network, dedupKey, hash, location))
	require.NoError(t, err)

	ids := make([]string, 0, 3)

	for _, node := range []string{"node-a", "node-b", "node-c"} {
		row := newTestState(network, 50, "root-e", node, location, hash, time.Now())
		ids = append(ids, row.ID)

		outcome, ierr := indexer.InsertArtifact(ctx, &InsertArtifactParams{
			Row:             row,
			ConflictColumns: []string{KeyNetwork, KeySlot, KeyStateRoot, KeyNode},
			Kind:            KindBeaconState,
			Network:         network,
			DedupKey:        dedupKey,
			ContentHash:     hash,
			Location:        location,
		})
		require.NoError(t, ierr)
		require.Equal(t, ArtifactInsertLinked, outcome)
	}

	deleted, err := indexer.DeleteArtifacts(ctx, KindBeaconState, ids[:2], []BlobRefDecrement{{
		Kind:     KindBeaconState,
		Network:  network,
		DedupKey: dedupKey,
		Count:    2,
	}})
	require.NoError(t, err)
	require.Equal(t, int64(2), deleted)

	blob, err := indexer.GetBlob(ctx, KindBeaconState, network, dedupKey)
	require.NoError(t, err)
	require.Equal(t, int64(1), blob.RefCount)

	// Over-releasing must floor at zero rather than go negative.
	_, err = indexer.DeleteArtifacts(ctx, KindBeaconState, ids[2:], []BlobRefDecrement{{
		Kind:     KindBeaconState,
		Network:  network,
		DedupKey: dedupKey,
		Count:    99,
	}})
	require.NoError(t, err)

	blob, err = indexer.GetBlob(ctx, KindBeaconState, network, dedupKey)
	require.NoError(t, err)
	require.Equal(t, int64(0), blob.RefCount)
}

func TestTombstoneBlobRepairsADriftedCounter(t *testing.T) {
	indexer, _, err := NewMockIndexer()
	require.NoError(t, err)

	ctx := context.Background()
	network := generateRandomString(6)
	dedupKey := "60/root-f"
	hash := generateRandomString(64)
	location := "states/root-f.ssz"

	_, err = indexer.InsertBlob(ctx, newTestBlob(network, dedupKey, hash, location))
	require.NoError(t, err)

	// A row that references the payload while the counter claims nobody does.
	require.NoError(t, indexer.InsertBeaconState(ctx, newTestState(network, 60, "root-f", "node-a", location, hash, time.Now())))

	candidates, err := indexer.ListCollectableBlobs(ctx, time.Now().Add(time.Minute), 10)
	require.NoError(t, err)
	require.Len(t, candidates, 1)

	outcome, err := indexer.TombstoneBlob(ctx, candidates[0])
	require.NoError(t, err)
	require.Equal(t, BlobTombstoneReferenced, outcome)

	blob, err := indexer.GetBlob(ctx, KindBeaconState, network, dedupKey)
	require.NoError(t, err)
	require.Equal(t, int64(1), blob.RefCount, "the count is repaired from the rows themselves")
	require.Equal(t, BlobStateReady, blob.State)
}

func TestBlobCollectionRespectsTheGracePeriod(t *testing.T) {
	indexer, _, err := NewMockIndexer()
	require.NoError(t, err)

	ctx := context.Background()
	network := generateRandomString(6)

	blob := newTestBlob(network, "70/root-g", generateRandomString(64), "states/root-g.ssz")
	blob.CreatedAt = time.Now()

	_, err = indexer.InsertBlob(ctx, blob)
	require.NoError(t, err)

	candidates, err := indexer.ListCollectableBlobs(ctx, time.Now().Add(-10*time.Minute), 10)
	require.NoError(t, err)
	require.Empty(t, candidates, "a payload younger than the grace period is not a candidate")

	candidates, err = indexer.ListCollectableBlobs(ctx, time.Now().Add(time.Minute), 10)
	require.NoError(t, err)
	require.Len(t, candidates, 1)

	outcome, err := indexer.TombstoneBlob(ctx, candidates[0])
	require.NoError(t, err)
	require.Equal(t, BlobTombstoneMarked, outcome)

	removed, err := indexer.DeleteTombstonedBlob(ctx, candidates[0])
	require.NoError(t, err)
	require.True(t, removed)

	_, err = indexer.GetBlob(ctx, KindBeaconState, network, "70/root-g")
	require.ErrorIs(t, err, ErrBlobNotFound)
}

// A payload that comes back while its object is being deleted must survive the collector's
// final row delete, and must not be handed out at the path that is about to be emptied.
func TestResurrectedBlobOutlivesItsCollector(t *testing.T) {
	indexer, _, err := NewMockIndexer()
	require.NoError(t, err)

	ctx := context.Background()
	network := generateRandomString(6)
	dedupKey := "80/root-h"
	hash := generateRandomString(64)
	location := "states/root-h.ssz"

	_, err = indexer.InsertBlob(ctx, newTestBlob(network, dedupKey, hash, location))
	require.NoError(t, err)

	candidates, err := indexer.ListCollectableBlobs(ctx, time.Now().Add(time.Minute), 10)
	require.NoError(t, err)
	require.Len(t, candidates, 1)

	outcome, err := indexer.TombstoneBlob(ctx, candidates[0])
	require.NoError(t, err)
	require.Equal(t, BlobTombstoneMarked, outcome)

	// The agent re-uploads the same bytes, which land at the same path.
	revived, err := indexer.InsertBlob(ctx, newTestBlob(network, dedupKey, hash, location))
	require.NoError(t, err)
	require.Equal(t, BlobStateReady, revived.State)
	require.Equal(t, int64(1), revived.Generation)
	require.Equal(t, location+"-g1", revived.Location)

	removed, err := indexer.DeleteTombstonedBlob(ctx, candidates[0])
	require.NoError(t, err)
	require.False(t, removed, "the collector must not remove a payload that has come back")

	blob, err := indexer.GetBlob(ctx, KindBeaconState, network, dedupKey)
	require.NoError(t, err)
	require.Equal(t, location+"-g1", blob.Location)
}

func TestFindRootDisagreements(t *testing.T) {
	indexer, _, err := NewMockIndexer()
	require.NoError(t, err)

	ctx := context.Background()
	network := generateRandomString(6)
	now := time.Now()

	require.NoError(t, indexer.InsertBeaconState(ctx, newTestState(network, 100, "root-x", "node-a", "a", "", now)))
	require.NoError(t, indexer.InsertBeaconState(ctx, newTestState(network, 100, "root-y", "node-b", "b", "", now)))
	require.NoError(t, indexer.InsertBeaconState(ctx, newTestState(network, 101, "root-z", "node-a", "c", "", now)))
	require.NoError(t, indexer.InsertBeaconState(ctx, newTestState(network, 101, "root-z", "node-b", "d", "", now)))

	disagreements, err := indexer.FindRootDisagreements(ctx, KindBeaconState, 64)
	require.NoError(t, err)

	found := make(map[int64]int64)

	for _, d := range disagreements {
		if d.Network == network {
			found[d.Slot] = d.Roots
		}
	}

	require.Equal(t, map[int64]int64{100: 2}, found)
}
