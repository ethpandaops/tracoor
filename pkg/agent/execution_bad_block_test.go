package agent

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"

	"github.com/ethpandaops/tracoor/pkg/proto/tracoor/indexer"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

// badBlockIndexer answers ListExecutionBadBlock from a fixed set of hashes
// and counts how often it is asked.
type badBlockIndexer struct {
	indexerClient

	mu      sync.Mutex
	indexed map[string]bool
	asked   []string
	err     error
}

func (f *badBlockIndexer) ListExecutionBadBlock(_ context.Context, req *indexer.ListExecutionBadBlockRequest) (*indexer.ListExecutionBadBlockResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.asked = append(f.asked, req.GetBlockHash())

	if f.err != nil {
		return nil, f.err
	}

	rsp := &indexer.ListExecutionBadBlockResponse{}
	if f.indexed[req.GetBlockHash()] {
		rsp.ExecutionBadBlocks = []*indexer.ExecutionBadBlock{{}}
	}

	return rsp, nil
}

func newBadBlockTestAgent(fake *badBlockIndexer) *agent {
	log := logrus.New()
	log.SetOutput(io.Discard)

	return &agent{
		Config:  &Config{Name: "node-1"},
		log:     log,
		indexer: fake,
	}
}

func TestExecutionBadBlockWantedAsksTheIndexerOnceForAnIndexedBlock(t *testing.T) {
	fake := &badBlockIndexer{indexed: map[string]bool{"0x01": true}}
	s := newBadBlockTestAgent(fake)

	require.False(t, s.executionBadBlockWanted(context.Background(), "testnet", "0x01"))
	require.False(t, s.executionBadBlockWanted(context.Background(), "testnet", "0x01"))

	require.Equal(t, []string{"0x01"}, fake.asked, "the second poll must be answered from memory")
}

func TestExecutionBadBlockWantedKeepsAskingAboutAnUnindexedBlock(t *testing.T) {
	fake := &badBlockIndexer{}
	s := newBadBlockTestAgent(fake)

	require.True(t, s.executionBadBlockWanted(context.Background(), "testnet", "0x02"))
	require.True(t, s.executionBadBlockWanted(context.Background(), "testnet", "0x02"))

	require.Equal(t, []string{"0x02", "0x02"}, fake.asked, "nothing is remembered until the indexer has the block")
}

func TestExecutionBadBlockWantedLeavesABlockAloneWhenTheIndexerIsUnreachable(t *testing.T) {
	fake := &badBlockIndexer{err: errors.New("indexer down")}
	s := newBadBlockTestAgent(fake)

	require.False(t, s.executionBadBlockWanted(context.Background(), "testnet", "0x03"))
	require.False(t, s.indexedExecutionBadBlocks.has("0x03"), "an unanswered question must not be cached as an answer")
}

func TestHashSetZeroValueIsUsable(t *testing.T) {
	var set hashSet

	require.False(t, set.has("0x01"))

	set.add("0x01")

	require.True(t, set.has("0x01"))
	require.False(t, set.has("0x02"))
}
