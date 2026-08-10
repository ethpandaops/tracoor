package indexer

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/ethpandaops/beacon/pkg/human"
	pindexer "github.com/ethpandaops/tracoor/pkg/proto/tracoor/indexer"
	"github.com/ethpandaops/tracoor/pkg/server/persistence"
	"github.com/ethpandaops/tracoor/pkg/store"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// expireImmediately purges everything on sight and collects payloads with no grace, which is
// what makes a single pass observable in a test.
func expireImmediately() *Config {
	return &Config{
		Retention: RetentionConfig{
			BeaconStates:              human.Duration{},
			BeaconBlocks:              human.Duration{},
			ExecutionPayloadEnvelopes: human.Duration{},
			BeaconBadBlocks:           human.Duration{},
			BeaconBadBlobs:            human.Duration{},
			ExecutionBlockTraces:      human.Duration{},
			ExecutionBadBlocks:        human.Duration{},
		},
		BlobGCGracePeriod: human.Duration{},
	}
}

func saveObject(ctx context.Context, t *testing.T, index *Indexer, location string) {
	t.Helper()

	_, err := index.Store().SaveBeaconState(ctx, &store.SaveParams{
		Data:     bytes.NewReader([]byte("payload")),
		Location: location,
	})
	require.NoError(t, err)
}

func stateRequest(network, node, root, location, hash, dedupKey string, slot uint64) *pindexer.CreateBeaconStateRequest {
	return &pindexer.CreateBeaconStateRequest{
		Node:                 wrapperspb.String(node),
		Slot:                 wrapperspb.UInt64(slot),
		Epoch:                wrapperspb.UInt64(slot / 32),
		StateRoot:            wrapperspb.String(root),
		FetchedAt:            timestamppb.New(time.Now().Add(-time.Hour)),
		BeaconImplementation: wrapperspb.String("impl"),
		NodeVersion:          wrapperspb.String("v1"),
		Location:             wrapperspb.String(location),
		Network:              wrapperspb.String(network),
		ContentHash:          wrapperspb.String(hash),
		DedupKey:             wrapperspb.String(dedupKey),
	}
}

func TestCreatePathStatuses(t *testing.T) {
	ctx := context.Background()

	index, cleanup, err := NewMockIndexer(ctx, &Config{})
	require.NoError(t, err)

	defer func() { require.NoError(t, cleanup()) }()

	network := generateRandomString(6)
	hash := generateRandomContentHash()
	location := "states/" + generateRandomString(8) + ".ssz"
	dedupKey := "100/root-a"

	t.Run("a payload with no blob is refused so the agent re-uploads", func(t *testing.T) {
		saveObject(ctx, t, index, location)

		_, cerr := index.CreateBeaconState(ctx, stateRequest(network, "node-a", "root-a", location, hash, dedupKey, 100))
		require.Error(t, cerr)
		require.Equal(t, codes.FailedPrecondition, status.Code(cerr))
		require.Contains(t, status.Convert(cerr).Message(), "blob not linkable")
	})

	_, err = index.CreateBlob(ctx, &pindexer.CreateBlobRequest{
		Kind:        wrapperspb.String(persistence.KindBeaconState),
		Network:     wrapperspb.String(network),
		DedupKey:    wrapperspb.String(dedupKey),
		ContentHash: wrapperspb.String(hash),
		Location:    wrapperspb.String(location),
	})
	require.NoError(t, err)

	t.Run("a linked create takes exactly one reference", func(t *testing.T) {
		_, err := index.CreateBeaconState(ctx, stateRequest(network, "node-a", "root-a", location, hash, dedupKey, 100))
		require.NoError(t, err)

		blob, err := index.db.GetBlob(ctx, persistence.KindBeaconState, network, dedupKey)
		require.NoError(t, err)
		require.Equal(t, int64(1), blob.RefCount)
	})

	t.Run("a duplicate is rejected and does not take a second reference", func(t *testing.T) {
		_, err := index.CreateBeaconState(ctx, stateRequest(network, "node-a", "root-a", location, hash, dedupKey, 100))
		require.Error(t, err)
		require.Equal(t, codes.AlreadyExists, status.Code(err))

		blob, err := index.db.GetBlob(ctx, persistence.KindBeaconState, network, dedupKey)
		require.NoError(t, err)
		require.Equal(t, int64(1), blob.RefCount)
	})

	t.Run("an unlinked write still has to prove its object exists", func(t *testing.T) {
		_, err := index.CreateBeaconBadBlock(ctx, &pindexer.CreateBeaconBadBlockRequest{
			Node:                 wrapperspb.String("node-a"),
			Slot:                 wrapperspb.UInt64(100),
			Epoch:                wrapperspb.UInt64(3),
			BlockRoot:            wrapperspb.String("root-bad"),
			FetchedAt:            timestamppb.Now(),
			BeaconImplementation: wrapperspb.String("impl"),
			NodeVersion:          wrapperspb.String("v1"),
			Location:             wrapperspb.String("bad_blocks/nothing-here.ssz"),
			Network:              wrapperspb.String(network),
		})
		require.Error(t, err)
		require.Equal(t, codes.FailedPrecondition, status.Code(err))
		require.Contains(t, status.Convert(err).Message(), "object not present in store")
	})
}

func TestRetentionReleasesReferencesAndCollectsPayloads(t *testing.T) {
	ctx := context.Background()

	index, cleanup, err := NewMockIndexer(ctx, expireImmediately())
	require.NoError(t, err)

	defer func() { require.NoError(t, cleanup()) }()

	network := generateRandomString(6)
	hash := generateRandomContentHash()
	location := "states/" + generateRandomString(8) + ".ssz"
	dedupKey := "200/root-b"

	saveObject(ctx, t, index, location)

	_, err = index.CreateBlob(ctx, &pindexer.CreateBlobRequest{
		Kind:        wrapperspb.String(persistence.KindBeaconState),
		Network:     wrapperspb.String(network),
		DedupKey:    wrapperspb.String(dedupKey),
		ContentHash: wrapperspb.String(hash),
		Location:    wrapperspb.String(location),
	})
	require.NoError(t, err)

	for _, node := range []string{"node-a", "node-b"} {
		_, cerr := index.CreateBeaconState(ctx, stateRequest(network, node, "root-b", location, hash, dedupKey, 200))
		require.NoError(t, cerr)
	}

	blob, err := index.db.GetBlob(ctx, persistence.KindBeaconState, network, dedupKey)
	require.NoError(t, err)
	require.Equal(t, int64(2), blob.RefCount)

	require.NoError(t, index.purgeArtifacts(ctx, specForKind(t, index, persistence.KindBeaconState)))

	rows, err := index.ListBeaconState(ctx, &pindexer.ListBeaconStateRequest{Network: network})
	require.NoError(t, err)
	require.Empty(t, rows.BeaconStates)

	blob, err = index.db.GetBlob(ctx, persistence.KindBeaconState, network, dedupKey)
	require.NoError(t, err)
	require.Equal(t, int64(0), blob.RefCount)

	exists, err := index.Store().Exists(ctx, location)
	require.NoError(t, err)
	require.True(t, exists, "a shared payload outlives the rows that referenced it")

	require.NoError(t, index.purgeOrphanedBlobs(ctx))

	_, err = index.db.GetBlob(ctx, persistence.KindBeaconState, network, dedupKey)
	require.ErrorIs(t, err, persistence.ErrBlobNotFound)

	exists, err = index.Store().Exists(ctx, location)
	require.NoError(t, err)
	require.False(t, exists, "the collector takes the object with the last reference")
}

func TestRetentionDeletesObjectsOwnedByUnlinkedRows(t *testing.T) {
	ctx := context.Background()

	index, cleanup, err := NewMockIndexer(ctx, expireImmediately())
	require.NoError(t, err)

	defer func() { require.NoError(t, cleanup()) }()

	network := generateRandomString(6)
	location := "bad_blocks/" + generateRandomString(8) + ".ssz"

	saveObject(ctx, t, index, location)

	_, err = index.CreateBeaconBadBlock(ctx, &pindexer.CreateBeaconBadBlockRequest{
		Node:                 wrapperspb.String("node-a"),
		Slot:                 wrapperspb.UInt64(300),
		Epoch:                wrapperspb.UInt64(9),
		BlockRoot:            wrapperspb.String("root-c"),
		FetchedAt:            timestamppb.New(time.Now().Add(-time.Hour)),
		BeaconImplementation: wrapperspb.String("impl"),
		NodeVersion:          wrapperspb.String("v1"),
		Location:             wrapperspb.String(location),
		Network:              wrapperspb.String(network),
	})
	require.NoError(t, err)

	require.NoError(t, index.purgeArtifacts(ctx, specForKind(t, index, persistence.KindBeaconBadBlock)))

	exists, err := index.Store().Exists(ctx, location)
	require.NoError(t, err)
	require.False(t, exists, "a row that owns its object takes the object with it")
}

func TestBlobCollectionWaitsOutTheGracePeriod(t *testing.T) {
	ctx := context.Background()

	config := expireImmediately()
	config.BlobGCGracePeriod = human.Duration{Duration: time.Hour}

	index, cleanup, err := NewMockIndexer(ctx, config)
	require.NoError(t, err)

	defer func() { require.NoError(t, cleanup()) }()

	network := generateRandomString(6)
	hash := generateRandomContentHash()
	location := "states/" + generateRandomString(8) + ".ssz"
	dedupKey := "400/root-d"

	saveObject(ctx, t, index, location)

	_, err = index.CreateBlob(ctx, &pindexer.CreateBlobRequest{
		Kind:        wrapperspb.String(persistence.KindBeaconState),
		Network:     wrapperspb.String(network),
		DedupKey:    wrapperspb.String(dedupKey),
		ContentHash: wrapperspb.String(hash),
		Location:    wrapperspb.String(location),
	})
	require.NoError(t, err)

	require.NoError(t, index.purgeOrphanedBlobs(ctx))

	blob, err := index.db.GetBlob(ctx, persistence.KindBeaconState, network, dedupKey)
	require.NoError(t, err)
	require.Equal(t, persistence.BlobStateReady, blob.State)

	exists, err := index.Store().Exists(ctx, location)
	require.NoError(t, err)
	require.True(t, exists)
}

func TestRootDisagreementDetector(t *testing.T) {
	ctx := context.Background()

	index, cleanup, err := NewMockIndexer(ctx, &Config{})
	require.NoError(t, err)

	defer func() { require.NoError(t, cleanup()) }()

	logger, hook := logrustest.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)
	index.log = logger

	network := generateRandomString(6)
	hash := generateRandomContentHash()

	for _, root := range []string{"root-one", "root-two"} {
		location := "states/" + generateRandomString(8) + ".ssz"
		dedupKey := "500/" + root

		saveObject(ctx, t, index, location)

		_, cerr := index.CreateBlob(ctx, &pindexer.CreateBlobRequest{
			Kind:        wrapperspb.String(persistence.KindBeaconState),
			Network:     wrapperspb.String(network),
			DedupKey:    wrapperspb.String(dedupKey),
			ContentHash: wrapperspb.String(hash),
			Location:    wrapperspb.String(location),
		})
		require.NoError(t, cerr)

		_, cerr = index.CreateBeaconState(ctx, stateRequest(network, "node-"+root, root, location, hash, dedupKey, 500))
		require.NoError(t, cerr)
	}

	before := testutil.ToFloat64(index.metrics.rootDivergence.WithLabelValues(network, persistence.KindBeaconState))

	index.detectRootDisagreements(ctx)

	after := testutil.ToFloat64(index.metrics.rootDivergence.WithLabelValues(network, persistence.KindBeaconState))
	require.Equal(t, before+1, after)

	warnings := 0

	for _, entry := range hook.AllEntries() {
		if entry.Level == logrus.WarnLevel && entry.Data[KeyNetwork] == network {
			warnings++
		}
	}

	require.Equal(t, 1, warnings)

	// A disagreement that persists must not be re-reported every cycle.
	hook.Reset()
	index.detectRootDisagreements(ctx)

	for _, entry := range hook.AllEntries() {
		require.NotEqual(t, logrus.WarnLevel, entry.Level, "the same slot must only be warned about once")
	}
}

// A location the store keeps refusing is retried a bounded number of times and then reported,
// because rows are deleted first and nothing else will ever drive the retry.
func TestObjectReaperQuarantinesAPoisonedLocation(t *testing.T) {
	reaper := newObjectReaper()

	require.Empty(t, reaper.pending())

	require.Empty(t, reaper.settle([]string{"a", "b"}, []string{"a"}))
	require.Equal(t, []string{"a"}, reaper.pending())

	require.Empty(t, reaper.settle([]string{"a"}, []string{"a"}))
	require.Equal(t, []string{"a"}, reaper.pending())

	require.Equal(t, []string{"a"}, reaper.settle([]string{"a"}, []string{"a"}))
	require.Empty(t, reaper.pending(), "a quarantined location is not retried again")
}

func TestObjectReaperForgetsWhatSucceeded(t *testing.T) {
	reaper := newObjectReaper()

	require.Empty(t, reaper.settle([]string{"a"}, []string{"a"}))
	require.Equal(t, []string{"a"}, reaper.pending())

	require.Empty(t, reaper.settle([]string{"a"}, nil))
	require.Empty(t, reaper.pending())
}
