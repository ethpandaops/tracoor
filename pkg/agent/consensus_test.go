package agent

import (
	"context"
	goerrors "errors"
	"testing"

	"github.com/stretchr/testify/require"
)

// pinnedTo wraps the fixture's fetcher the way the state path wraps its own for
// the clients that can only be asked for a state by slot.
func pinnedTo(f *dedupFixture, root string, resolved ...string) *stateRootPinnedFetcher {
	reads := 0

	return &stateRootPinnedFetcher{
		payloadFetcher: f.fetcher,
		root:           root,
		resolve: func(context.Context) (string, error) {
			current := resolved[min(reads, len(resolved)-1)]
			reads++

			return current, nil
		},
		discard: f.target.remove,
	}
}

func TestStateRootPinnedFetcherRecordsNothingWhenTheRootMoves(t *testing.T) {
	f := newDedupFixture(t, "root-moved").withPayload(payload(16 * 1024))

	created := 0

	err := f.agent.indexDeduplicated(
		context.Background(),
		f.target,
		pinnedTo(f, "0xroot", "0xsomethingelse"),
		func(context.Context, *dedupOutcome) error {
			created++

			return nil
		},
	)

	// A state fetched by slot whose root moved describes a different state than
	// the row would have claimed, so nothing about it may be kept.
	require.ErrorIs(t, err, errItemNotAvailable)
	require.Zero(t, created, "no row may be written for bytes that belong to another root")
	require.Zero(t, f.indexer.createBlobs, "the canonical payload for the key must not be claimed by another root's bytes")
	require.Empty(t, f.indexer.recorded(), "a reorg is not a divergence")
	require.Empty(t, storedObjects(t, f.base), "the payload that was staged before the root moved is taken with us")

	uploads, _ := f.fetcher.counts()
	require.Equal(t, 1, uploads, "the attempt was spent")
}

func TestStateRootPinnedFetcherRecordsNothingWhenTheRootMovesOnTheHitPath(t *testing.T) {
	data := payload(8192)
	f := newDedupFixture(t, "root-moved-hit").withPayload(data)

	// Somebody already stored this payload, so this node is only read to be
	// compared against it. The comparison is worthless if the bytes came from a
	// different state than the key names.
	f.indexer.setBlob(f.target.dedupKey, hashOf(data), "somebody/elses/object.ssz")

	created := 0

	err := f.agent.indexDeduplicated(
		context.Background(),
		f.target,
		pinnedTo(f, "0xroot", "0xsomethingelse"),
		func(context.Context, *dedupOutcome) error {
			created++

			return nil
		},
	)

	require.ErrorIs(t, err, errItemNotAvailable)
	require.Zero(t, created)
	require.Empty(t, f.indexer.recorded())
	require.Empty(t, storedObjects(t, f.base))
}

func TestStateRootPinnedFetcherIndexesWhenTheRootHeld(t *testing.T) {
	data := payload(16 * 1024)
	f := newDedupFixture(t, "root-held").withPayload(data)

	var outcome *dedupOutcome

	require.NoError(t, f.agent.indexDeduplicated(
		context.Background(),
		f.target,
		pinnedTo(f, "0xroot", "0xroot"),
		func(_ context.Context, o *dedupOutcome) error {
			outcome = o

			return nil
		},
	))

	require.Equal(t, hashOf(data), outcome.ContentHash)
	require.Equal(t, f.target.finalLocation(hashOf(data)), outcome.Location)
	require.Equal(t, []string{outcome.Location}, storedObjects(t, f.base))
}

func TestStateRootPinnedFetcherFailsWhenTheRootCannotBeConfirmed(t *testing.T) {
	f := newDedupFixture(t, "root-unconfirmable").withPayload(payload(4096))

	resolveErr := goerrors.New("node stopped answering")

	fetcher := &stateRootPinnedFetcher{
		payloadFetcher: f.fetcher,
		root:           "0xroot",
		resolve: func(context.Context) (string, error) {
			return "", resolveErr
		},
		discard: f.target.remove,
	}

	created := 0

	err := f.agent.indexDeduplicated(context.Background(), f.target, fetcher, func(context.Context, *dedupOutcome) error {
		created++

		return nil
	})

	// An unconfirmable root is not a confirmed one: the payload is refused
	// either way.
	require.ErrorIs(t, err, resolveErr)
	require.Zero(t, created)
	require.Empty(t, storedObjects(t, f.base))
}
