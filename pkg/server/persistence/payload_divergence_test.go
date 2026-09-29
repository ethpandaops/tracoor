package persistence

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func generateRandomPayloadDivergence(network, kind, dedupKey string, observedAt time.Time) *PayloadDivergence {
	return &PayloadDivergence{
		ID:           uuid.New().String(),
		ObservedAt:   observedAt,
		Network:      network,
		Kind:         kind,
		Node:         generateRandomString(5),
		DedupKey:     dedupKey,
		ExpectedHash: generateRandomString(64),
		ActualHash:   generateRandomString(64),
		Slot:         100,
		Identifier:   generateRandomString(32),
		Attempt:      1,
		Severity:     "alarm",
	}
}

func TestPayloadDivergenceInsertAndList(t *testing.T) {
	indexer, _, err := NewMockIndexer()
	require.NoError(t, err)

	ctx := context.Background()
	now := time.Now()

	require.NoError(t, indexer.InsertPayloadDivergence(ctx,
		generateRandomPayloadDivergence("mainnet", "beacon_state", "100/0xaaaa", now.Add(-2*time.Hour))))
	require.NoError(t, indexer.InsertPayloadDivergence(ctx,
		generateRandomPayloadDivergence("mainnet", "beacon_state", "101/0xbbbb", now.Add(-1*time.Hour))))
	require.NoError(t, indexer.InsertPayloadDivergence(ctx,
		generateRandomPayloadDivergence("sepolia", "beacon_block", "102/0xcccc", now)))

	t.Run("lists newest first", func(t *testing.T) {
		got, err := indexer.ListPayloadDivergence(ctx, &PayloadDivergenceFilter{}, &PaginationCursor{})
		require.NoError(t, err)
		require.Len(t, got, 3)
		assert.Equal(t, "102/0xcccc", got[0].DedupKey)
	})

	t.Run("filters by network and kind", func(t *testing.T) {
		filter := &PayloadDivergenceFilter{}
		filter.AddNetwork("mainnet")
		filter.AddKind("beacon_state")

		got, err := indexer.ListPayloadDivergence(ctx, filter, &PaginationCursor{})
		require.NoError(t, err)
		assert.Len(t, got, 2)
	})

	t.Run("filters by dedup key", func(t *testing.T) {
		filter := &PayloadDivergenceFilter{}
		filter.AddDedupKey("101/0xbbbb")

		got, err := indexer.ListPayloadDivergence(ctx, filter, &PaginationCursor{})
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, "mainnet", got[0].Network)
	})

	t.Run("filters by time range", func(t *testing.T) {
		filter := &PayloadDivergenceFilter{}
		filter.AddAfter(now.Add(-90 * time.Minute))

		got, err := indexer.ListPayloadDivergence(ctx, filter, &PaginationCursor{})
		require.NoError(t, err)
		assert.Len(t, got, 2)
	})
}
