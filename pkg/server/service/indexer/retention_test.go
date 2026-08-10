package indexer

import (
	"bytes"
	"context"
	"strconv"
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

	reorgs := func() float64 {
		return testutil.ToFloat64(index.metrics.rootDivergence.WithLabelValues(network, persistence.KindBeaconState, causeReorg))
	}

	before := reorgs()

	index.detectRootDisagreements(ctx)

	require.Equal(t, before+1, reorgs(), "an unexplained disagreement is a reorg until proven otherwise")

	warnings := 0

	for _, entry := range hook.AllEntries() {
		if entry.Level == logrus.WarnLevel && entry.Data[KeyNetwork] == network {
			warnings++
		}
	}

	require.Equal(t, 1, warnings)

	// A disagreement that persists is one event, however many cycles observe it. The detector
	// runs every minute over a window many minutes wide, so counting each pass would make the
	// metric a measure of the cadence.
	hook.Reset()

	for cycle := 0; cycle < 3; cycle++ {
		index.detectRootDisagreements(ctx)
	}

	require.Equal(t, before+1, reorgs(), "the same disagreement must only be counted once")

	for _, entry := range hook.AllEntries() {
		require.NotEqual(t, logrus.WarnLevel, entry.Level, "the same slot must only be warned about once")
	}
}

// A slot two nodes disagree about is a reorg unless some node was also caught serving bytes
// that did not match the canonical payload. Only the latter is worth waking someone for.
func TestRootDisagreementCauseSeparatesReorgsFromDivergence(t *testing.T) {
	ctx := context.Background()

	index, cleanup, err := NewMockIndexer(ctx, &Config{})
	require.NoError(t, err)

	defer func() { require.NoError(t, cleanup()) }()

	network := generateRandomString(6)
	hash := generateRandomContentHash()

	for _, root := range []string{"root-one", "root-two"} {
		location := "states/" + generateRandomString(8) + ".ssz"
		dedupKey := "600/" + root

		saveObject(ctx, t, index, location)

		_, cerr := index.CreateBlob(ctx, &pindexer.CreateBlobRequest{
			Kind:        wrapperspb.String(persistence.KindBeaconState),
			Network:     wrapperspb.String(network),
			DedupKey:    wrapperspb.String(dedupKey),
			ContentHash: wrapperspb.String(hash),
			Location:    wrapperspb.String(location),
		})
		require.NoError(t, cerr)

		_, cerr = index.CreateBeaconState(ctx, stateRequest(network, "node-"+root, root, location, hash, dedupKey, 600))
		require.NoError(t, cerr)
	}

	_, err = index.CreatePayloadDivergence(ctx, &pindexer.CreatePayloadDivergenceRequest{
		Network:      wrapperspb.String(network),
		Kind:         wrapperspb.String(persistence.KindBeaconState),
		Node:         wrapperspb.String("node-root-one"),
		DedupKey:     wrapperspb.String("600/root-one"),
		ExpectedHash: wrapperspb.String(hash),
		ActualHash:   wrapperspb.String(generateRandomContentHash()),
		Slot:         wrapperspb.UInt64(600),
		Identifier:   wrapperspb.String("root-one"),
		Severity:     wrapperspb.String("alarm"),
	})
	require.NoError(t, err)

	index.detectRootDisagreements(ctx)

	require.Equal(t, float64(1),
		testutil.ToFloat64(index.metrics.rootDivergence.WithLabelValues(network, persistence.KindBeaconState, causeDivergence)))
	require.Equal(t, float64(0),
		testutil.ToFloat64(index.metrics.rootDivergence.WithLabelValues(network, persistence.KindBeaconState, causeReorg)))
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

// stateRequestAt is stateRequest with the fetch time spelled out, for the tests that care
// exactly how old a row is.
func stateRequestAt(network, node, root, location, hash, dedupKey string, slot uint64, fetchedAt time.Time) *pindexer.CreateBeaconStateRequest {
	req := stateRequest(network, node, root, location, hash, dedupKey, slot)
	req.FetchedAt = timestamppb.New(fetchedAt)

	return req
}

// Retention builds its cutoff from the process clock, which carries the host's zone, while
// rows are stored with a UTC fetch time. Comparing those by anything other than the instant
// they name empties the index on a host that is not set to UTC.
func TestRetentionCutoffIsIndependentOfTheProcessZone(t *testing.T) {
	ctx := context.Background()

	local := time.Local
	time.Local = time.FixedZone("test", 10*60*60)

	t.Cleanup(func() { time.Local = local })

	config := expireImmediately()
	config.Retention.BeaconStates = human.Duration{Duration: 30 * time.Minute}

	index, cleanup, err := NewMockIndexer(ctx, config)
	require.NoError(t, err)

	defer func() { require.NoError(t, cleanup()) }()

	network := generateRandomString(6)
	now := time.Now().UTC()

	rows := []struct {
		root      string
		slot      uint64
		fetchedAt time.Time
	}{
		{"root-fresh", 700, now.Add(-5 * time.Minute)},
		{"root-stale", 701, now.Add(-2 * time.Hour)},
	}

	for _, row := range rows {
		hash := generateRandomContentHash()
		location := "states/" + generateRandomString(8) + ".ssz"
		dedupKey := strconv.FormatUint(row.slot, 10) + "/" + row.root

		saveObject(ctx, t, index, location)

		_, cerr := index.CreateBlob(ctx, &pindexer.CreateBlobRequest{
			Kind:        wrapperspb.String(persistence.KindBeaconState),
			Network:     wrapperspb.String(network),
			DedupKey:    wrapperspb.String(dedupKey),
			ContentHash: wrapperspb.String(hash),
			Location:    wrapperspb.String(location),
		})
		require.NoError(t, cerr)

		_, cerr = index.CreateBeaconState(ctx,
			stateRequestAt(network, "node-a", row.root, location, hash, dedupKey, row.slot, row.fetchedAt))
		require.NoError(t, cerr)
	}

	require.NoError(t, index.purgeArtifacts(ctx, specForKind(t, index, persistence.KindBeaconState)))

	listed, err := index.ListBeaconState(ctx, &pindexer.ListBeaconStateRequest{Network: network})
	require.NoError(t, err)
	require.Len(t, listed.BeaconStates, 1, "a row fetched five minutes ago is inside a thirty minute window")
	require.Equal(t, "root-fresh", listed.BeaconStates[0].GetStateRoot().GetValue())
}

// A payload the collector cannot evaluate must not stop it reaching the payloads behind it.
// Candidates come back oldest first, so without a way past them the head of the queue is the
// whole queue for ever.
func TestBlobCollectionStepsPastPayloadsItCannotEvaluate(t *testing.T) {
	ctx := context.Background()

	index, cleanup, err := NewMockIndexer(ctx, expireImmediately())
	require.NoError(t, err)

	defer func() { require.NoError(t, cleanup()) }()

	network := generateRandomString(6)
	old := time.Now().UTC().Add(-time.Hour)

	// More poison than fits in a page, so the collector cannot reach anything behind it by
	// accident.
	for i := 0; i < purgePageSize+1; i++ {
		_, ierr := index.db.InsertBlob(ctx, &persistence.Blob{
			Kind:      persistence.KindBeaconState,
			Network:   network,
			DedupKey:  "no-slot-here-" + strconv.Itoa(i),
			Location:  "states/poison-" + strconv.Itoa(i) + ".ssz",
			State:     persistence.BlobStateReady,
			CreatedAt: old,
		})
		require.NoError(t, ierr)
	}

	location := "states/" + generateRandomString(8) + ".ssz"
	dedupKey := "800/root-collectable"

	saveObject(ctx, t, index, location)

	_, err = index.db.InsertBlob(ctx, &persistence.Blob{
		Kind:      persistence.KindBeaconState,
		Network:   network,
		DedupKey:  dedupKey,
		Location:  location,
		State:     persistence.BlobStateReady,
		CreatedAt: old.Add(time.Minute),
	})
	require.NoError(t, err)

	require.NoError(t, index.purgeOrphanedBlobs(ctx))

	_, err = index.db.GetBlob(ctx, persistence.KindBeaconState, network, dedupKey)
	require.ErrorIs(t, err, persistence.ErrBlobNotFound, "the collectable payload behind the poison was reached")

	exists, err := index.Store().Exists(ctx, location)
	require.NoError(t, err)
	require.False(t, exists)
}

// A payload that fails evaluation is retried a bounded number of times and then left alone,
// rather than being re-attempted and re-reported for ever.
func TestBlobQuarantineBoundsRetries(t *testing.T) {
	quarantine := newBlobQuarantine()

	candidate := &persistence.BlobCandidate{Kind: "beacon_state", Network: "net", DedupKey: "bad"}
	key := blobQuarantineKey(candidate)

	require.False(t, quarantine.quarantined(key))

	for attempt := 1; attempt < maxBlobCollectAttempts; attempt++ {
		require.False(t, quarantine.fail(key))
		require.False(t, quarantine.quarantined(key))
	}

	require.True(t, quarantine.fail(key), "the last attempt is reported exactly once")
	require.True(t, quarantine.quarantined(key))

	quarantine.forget(key)
	require.False(t, quarantine.quarantined(key), "a payload that evaluates cleanly gets its allowance back")
}

// Rows that own their object are not always one to one with it: a copy stored because it
// disagreed with the canonical payload is addressed by its content alone, so every node that
// served those bytes wrote the same object. Purging one of them must not empty the location
// the others still advertise.
func TestRetentionKeepsAnObjectSiblingRowsStillPointAt(t *testing.T) {
	ctx := context.Background()

	index, cleanup, err := NewMockIndexer(ctx, expireImmediately())
	require.NoError(t, err)

	defer func() { require.NoError(t, cleanup()) }()

	network := generateRandomString(6)
	location := "bad_blocks/" + generateRandomString(8) + ".ssz"

	saveObject(ctx, t, index, location)

	for _, node := range []string{"node-a", "node-b"} {
		_, cerr := index.CreateBeaconBadBlock(ctx, &pindexer.CreateBeaconBadBlockRequest{
			Node:                 wrapperspb.String(node),
			Slot:                 wrapperspb.UInt64(900),
			Epoch:                wrapperspb.UInt64(28),
			BlockRoot:            wrapperspb.String("root-shared"),
			FetchedAt:            timestamppb.New(time.Now().Add(-time.Hour)),
			BeaconImplementation: wrapperspb.String("impl"),
			NodeVersion:          wrapperspb.String("v1"),
			Location:             wrapperspb.String(location),
			Network:              wrapperspb.String(network),
		})
		require.NoError(t, cerr)
	}

	expiring, err := index.db.ListExpiringBeaconBadBlocks(ctx, time.Now(), 10)
	require.NoError(t, err)
	require.Len(t, expiring, 2)

	// One page, one of the two rows: exactly what a page boundary inside the group looks like.
	_, err = index.purgePage(ctx, persistence.KindBeaconBadBlock, expiring[:1])
	require.NoError(t, err)

	exists, err := index.Store().Exists(ctx, location)
	require.NoError(t, err)
	require.True(t, exists, "the surviving row still advertises this location")

	_, err = index.purgePage(ctx, persistence.KindBeaconBadBlock, expiring[1:])
	require.NoError(t, err)

	exists, err = index.Store().Exists(ctx, location)
	require.NoError(t, err)
	require.False(t, exists, "the last row takes the object with it")
}

// One page of blocks must not be able to hold the whole cycle: the rows that do not get their
// turn inside the budget keep their objects and are offered again next time.
func TestArchivingBeforePurgeIsBoundedPerPage(t *testing.T) {
	ctx := context.Background()

	config := expireImmediately()
	config.PermanentStore.Blocks.Enabled = true

	index, cleanup, err := NewMockIndexer(ctx, config)
	require.NoError(t, err)

	defer func() { require.NoError(t, cleanup()) }()

	// The permanent store is never started, so nothing ever reports a block as processed.
	index.archiveBudget = 50 * time.Millisecond

	rows := make([]*persistence.ExpiringArtifact, 0, 20)
	for i := 0; i < 20; i++ {
		rows = append(rows, &persistence.ExpiringArtifact{
			ID:         generateRandomString(10),
			Location:   "blocks/" + generateRandomString(8) + ".ssz",
			Network:    "net",
			Identifier: "root-" + strconv.Itoa(i),
			Slot:       int64(i),
		})
	}

	start := time.Now()

	archived, err := index.archiveBlocksBeforePurge(ctx, rows)
	require.NoError(t, err)

	require.Less(t, time.Since(start), 5*time.Second, "the page has one budget, not one per row")
	require.NotEmpty(t, archived)
	require.Less(t, len(archived), len(rows), "the rows that did not get their turn keep their objects")
}

// The divergence log is written by every node on every mismatch, so it needs a bound of its own.
func TestPayloadDivergenceRetention(t *testing.T) {
	ctx := context.Background()

	index, cleanup, err := NewMockIndexer(ctx, expireImmediately())
	require.NoError(t, err)

	defer func() { require.NoError(t, cleanup()) }()

	network := generateRandomString(6)

	_, err = index.CreatePayloadDivergence(ctx, &pindexer.CreatePayloadDivergenceRequest{
		Network:      wrapperspb.String(network),
		Kind:         wrapperspb.String(persistence.KindBeaconState),
		Node:         wrapperspb.String("node-a"),
		DedupKey:     wrapperspb.String("1000/root-e"),
		ExpectedHash: wrapperspb.String(generateRandomContentHash()),
		ActualHash:   wrapperspb.String(generateRandomContentHash()),
		Slot:         wrapperspb.UInt64(1000),
		Identifier:   wrapperspb.String("root-e"),
		Severity:     wrapperspb.String("alarm"),
	})
	require.NoError(t, err)

	index.config.Retention.PayloadDivergences = human.Duration{Duration: time.Hour}
	require.NoError(t, index.purgePayloadDivergences(ctx))

	listed, err := index.db.ListPayloadDivergence(ctx, &persistence.PayloadDivergenceFilter{}, &persistence.PaginationCursor{Limit: 10})
	require.NoError(t, err)
	require.Len(t, listed, 1, "a fresh divergence is inside its window")

	index.config.Retention.PayloadDivergences = human.Duration{}
	require.NoError(t, index.purgePayloadDivergences(ctx))

	listed, err = index.db.ListPayloadDivergence(ctx, &persistence.PayloadDivergenceFilter{}, &persistence.PaginationCursor{Limit: 10})
	require.NoError(t, err)
	require.Empty(t, listed)
}
