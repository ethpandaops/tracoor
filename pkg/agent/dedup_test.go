package agent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	goerrors "errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ethpandaops/tracoor/pkg/compression"
	"github.com/ethpandaops/tracoor/pkg/proto/tracoor/indexer"
	"github.com/ethpandaops/tracoor/pkg/store"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestDedupKeys(t *testing.T) {
	tests := []struct {
		name string
		key  string
		want string
	}{
		{
			name: "beacon state is keyed on the slot and the state root",
			key:  beaconStateDedupKey(7, "0xaabb"),
			want: "7/0xaabb",
		},
		{
			name: "beacon block is keyed on the slot and the block root",
			key:  beaconBlockDedupKey(9, "0xccdd"),
			want: "9/0xccdd",
		},
		{
			name: "an envelope is keyed like the block it belongs to",
			key:  executionPayloadEnvelopeDedupKey(9, "0xccdd"),
			want: "9/0xccdd",
		},
		{
			name: "a trace carries the client, its version and the trace parameters",
			key:  executionBlockTraceDedupKey(42, "0xeeff", "geth", "geth/v1.14.0", false, true, false),
			want: "42/0xeeff/geth/geth/v1.14.0/010",
		},
		{
			name: "the trace parameters keep their order",
			key:  executionBlockTraceDedupKey(42, "0xeeff", "geth", "geth/v1.14.0", true, false, true),
			want: "42/0xeeff/geth/geth/v1.14.0/101",
		},
		{
			name: "every trace parameter off",
			key:  executionBlockTraceDedupKey(1, "0x00", "reth", "reth/v1", false, false, false),
			want: "1/0x00/reth/reth/v1/000",
		},
		{
			name: "every trace parameter on",
			key:  executionBlockTraceDedupKey(1, "0x00", "reth", "reth/v1", true, true, true),
			want: "1/0x00/reth/reth/v1/111",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, test.key)
		})
	}
}

func TestDedupLocations(t *testing.T) {
	target := &dedupTarget{
		directory: "beacon_states/testnet/slots/12",
		identity:  "0xroot",
		extension: ".ssz",
	}

	hash := strings.Repeat("a", 64)

	require.Equal(t, "beacon_states/testnet/slots/12/0xroot-aaaaaaaa.ssz", target.finalLocation(hash))

	staged := target.stagingLocation("node-a")
	require.True(t, strings.HasPrefix(staged, "beacon_states/testnet/slots/12/.tmp-node-a-"))
	require.NotEqual(t, staged, target.stagingLocation("node-a"), "two attempts must never share a staging location")
}

// fakeIndexer answers the three calls the dedup path makes and records what it
// was asked. Everything else is left unimplemented: a test that reaches for it
// is testing something this fake has no opinion about.
type fakeIndexer struct {
	indexerClient

	mu sync.Mutex

	blobs       map[string]*indexer.Blob
	divergences []*indexer.CreatePayloadDivergenceRequest
	getBlobs    int
	createBlobs int

	// onCreateBlob replaces the default first-writer-wins behaviour, which is
	// how a lost race is staged.
	onCreateBlob func(req *indexer.CreateBlobRequest) (*indexer.Blob, error)
}

func newFakeIndexer() *fakeIndexer {
	return &fakeIndexer{blobs: make(map[string]*indexer.Blob)}
}

func (f *fakeIndexer) GetBlob(_ context.Context, req *indexer.GetBlobRequest) (*indexer.GetBlobResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.getBlobs++

	blob, ok := f.blobs[req.GetDedupKey().GetValue()]
	if !ok {
		return nil, status.Error(codes.NotFound, "no such blob")
	}

	return &indexer.GetBlobResponse{Blob: blob}, nil
}

func (f *fakeIndexer) CreateBlob(_ context.Context, req *indexer.CreateBlobRequest) (*indexer.CreateBlobResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.createBlobs++

	if f.onCreateBlob != nil {
		blob, err := f.onCreateBlob(req)
		if err != nil {
			return nil, err
		}

		return &indexer.CreateBlobResponse{Blob: blob}, nil
	}

	key := req.GetDedupKey().GetValue()

	if existing, ok := f.blobs[key]; ok {
		return &indexer.CreateBlobResponse{Blob: existing}, nil
	}

	blob := &indexer.Blob{
		Kind:        req.GetKind(),
		Network:     req.GetNetwork(),
		DedupKey:    req.GetDedupKey(),
		ContentHash: req.GetContentHash(),
		Location:    req.GetLocation(),
		State:       wrapperspb.String(blobStateReady),
	}

	f.blobs[key] = blob

	return &indexer.CreateBlobResponse{Blob: blob}, nil
}

func (f *fakeIndexer) CreatePayloadDivergence(
	_ context.Context,
	req *indexer.CreatePayloadDivergenceRequest,
) (*indexer.CreatePayloadDivergenceResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.divergences = append(f.divergences, req)

	return &indexer.CreatePayloadDivergenceResponse{Id: wrapperspb.String("divergence")}, nil
}

func (f *fakeIndexer) setBlob(key, contentHash, location string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.blobs[key] = &indexer.Blob{
		DedupKey:    wrapperspb.String(key),
		ContentHash: wrapperspb.String(contentHash),
		Location:    wrapperspb.String(location),
		State:       wrapperspb.String(blobStateReady),
	}
}

func (f *fakeIndexer) deleteBlob(key string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	delete(f.blobs, key)
}

func (f *fakeIndexer) recorded() []*indexer.CreatePayloadDivergenceRequest {
	f.mu.Lock()
	defer f.mu.Unlock()

	return append([]*indexer.CreatePayloadDivergenceRequest(nil), f.divergences...)
}

// testFetcher serves canned payloads and counts how many times the node was
// read, which is what the dedup path's promises are made of.
type testFetcher struct {
	compressor *compression.Compressor
	save       saveFunc

	mu sync.Mutex

	data []byte
	// contentLength overrides what the payload claims to be, which is how a
	// truncated transfer is staged.
	contentLength int64
	uploads       int
	hashes        int
	uploadErr     error
	beforeUpload  func()
}

func (f *testFetcher) source() *payloadSource {
	src := sourceFromBytes(f.data)
	if f.contentLength != 0 {
		src.contentLength = f.contentLength
	}

	return src
}

func (f *testFetcher) Hash(context.Context) (streamResult, error) {
	f.mu.Lock()
	f.hashes++
	src := f.source()
	f.mu.Unlock()

	return hashSource(src)
}

func (f *testFetcher) Upload(ctx context.Context, location string) (string, streamResult, error) {
	f.mu.Lock()
	f.uploads++
	before := f.beforeUpload
	err := f.uploadErr
	src := f.source()
	f.mu.Unlock()

	if before != nil {
		before()
	}

	if err != nil {
		return "", streamResult{}, err
	}

	return streamSource(ctx, f.compressor, src, f.save, location)
}

func (f *testFetcher) counts() (uploads, hashes int) {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.uploads, f.hashes
}

type dedupFixture struct {
	agent   *agent
	indexer *fakeIndexer
	target  *dedupTarget
	fetcher *testFetcher
	base    string
}

func newDedupFixture(t *testing.T, name string) *dedupFixture {
	t.Helper()

	// The metrics are a process-wide singleton and the singleflight group is
	// package level, so each fixture takes a name of its own.
	name += "-" + uuid.NewString()[:8]

	log := logrus.New()
	log.SetOutput(io.Discard)

	fsStore, base := newTestFSStore(t)
	fake := newFakeIndexer()

	s := &agent{
		Config:     &Config{Name: name},
		log:        log,
		metrics:    GetMetricsInstance(namespace),
		breaker:    newCircuitBreaker(),
		store:      fsStore,
		indexer:    fake,
		compressor: compression.NewCompressor(),
	}

	// The dedup key is unique per test so the package-level singleflight group
	// cannot carry one test's work into another.
	target := &dedupTarget{
		kind:       store.BeaconStateDataType,
		queue:      BeaconStateQueue,
		network:    "testnet",
		dedupKey:   "1/0xroot-" + name,
		directory:  "beacon_states/testnet/slots/1",
		identity:   "0xroot",
		extension:  ".ssz",
		slot:       1,
		identifier: "0xroot",
		severity:   divergenceSeverityAlarm,
		save:       fsStore.SaveBeaconState,
		remove:     fsStore.DeleteBeaconState,
	}

	return &dedupFixture{
		agent:   s,
		indexer: fake,
		target:  target,
		fetcher: &testFetcher{compressor: s.compressor, save: fsStore.SaveBeaconState},
		base:    base,
	}
}

func (f *dedupFixture) withPayload(data []byte) *dedupFixture {
	f.fetcher.data = data

	return f
}

func (f *dedupFixture) run(t *testing.T, create func(ctx context.Context, outcome *dedupOutcome) error) error {
	t.Helper()

	return f.agent.indexDeduplicated(context.Background(), f.target, f.fetcher, create)
}

// storedObjects lists every published object, so a test can assert that a
// staging copy was cleaned up and that nothing was left behind by a failure.
func storedObjects(t *testing.T, base string) []string {
	t.Helper()

	var found []string

	err := filepath.WalkDir(base, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !entry.IsDir() {
			rel, rerr := filepath.Rel(base, path)
			if rerr != nil {
				return rerr
			}

			found = append(found, filepath.ToSlash(rel))
		}

		return nil
	})
	require.NoError(t, err)

	return found
}

func hashOf(data []byte) string {
	sum := sha256.Sum256(data)

	return hex.EncodeToString(sum[:])
}

func TestDedupMissStoresThePayloadAndLinksToIt(t *testing.T) {
	data := payload(64 * 1024)
	f := newDedupFixture(t, "miss").withPayload(data)

	var outcome *dedupOutcome

	require.NoError(t, f.run(t, func(_ context.Context, o *dedupOutcome) error {
		outcome = o

		return nil
	}))

	require.Equal(t, hashOf(data), outcome.ContentHash)
	require.Equal(t, f.target.finalLocation(hashOf(data)), outcome.Location)
	require.False(t, outcome.VerifiedAt.IsZero())
	require.Nil(t, outcome.ContentMatchedAt, "the first writer had nothing to compare against")

	uploads, hashes := f.fetcher.counts()
	require.Equal(t, 1, uploads)
	require.Zero(t, hashes, "the payload was hashed by the same pass that stored it")

	require.Equal(t, 1, f.indexer.createBlobs)
	require.Empty(t, f.indexer.recorded())

	require.Equal(t, []string{outcome.Location}, storedObjects(t, f.base), "the staging copy must not survive")
	require.Equal(t, float64(1), testutil.ToFloat64(f.agent.metrics.blobCreated.WithLabelValues(string(BeaconStateQueue), f.agent.Config.Name)))
	require.Equal(t, float64(1), testutil.ToFloat64(f.agent.metrics.payloadVerified.WithLabelValues(string(BeaconStateQueue), f.agent.Config.Name)))
	require.Equal(t, float64(1), testutil.ToFloat64(f.agent.metrics.itemExported.WithLabelValues(string(BeaconStateQueue), f.agent.Config.Name)))
	require.Equal(t, float64(len(data)), testutil.ToFloat64(f.agent.metrics.fetchBytes.WithLabelValues(string(BeaconStateQueue), f.agent.Config.Name)))

	stored := testutil.ToFloat64(f.agent.metrics.storedBytes.WithLabelValues(string(BeaconStateQueue), f.agent.Config.Name))
	require.Positive(t, stored)
	require.Less(t, stored, float64(len(data)), "the stored bytes are the compressed ones")
}

func TestDedupHitLinksToTheStoredPayloadWhenTheHashMatches(t *testing.T) {
	data := payload(32 * 1024)
	f := newDedupFixture(t, "hit-match").withPayload(data)

	f.indexer.setBlob(f.target.dedupKey, hashOf(data), "somebody/elses/object.ssz")

	var outcome *dedupOutcome

	require.NoError(t, f.run(t, func(_ context.Context, o *dedupOutcome) error {
		outcome = o

		return nil
	}))

	require.Equal(t, "somebody/elses/object.ssz", outcome.Location)
	require.Equal(t, hashOf(data), outcome.ContentHash)
	require.NotNil(t, outcome.ContentMatchedAt, "the hash was compared against a stored payload and matched")

	uploads, hashes := f.fetcher.counts()
	require.Equal(t, 1, hashes, "this node is still read in full")
	require.Zero(t, uploads, "the payload was already stored")

	require.Empty(t, storedObjects(t, f.base))
	require.Equal(t, float64(1), testutil.ToFloat64(f.agent.metrics.blobReused.WithLabelValues(string(BeaconStateQueue), f.agent.Config.Name)))
	require.Equal(t, float64(len(data)), testutil.ToFloat64(f.agent.metrics.fetchBytes.WithLabelValues(string(BeaconStateQueue), f.agent.Config.Name)))
	require.Zero(t, testutil.ToFloat64(f.agent.metrics.storedBytes.WithLabelValues(string(BeaconStateQueue), f.agent.Config.Name)),
		"a payload that was only hashed stored nothing, which is the gap dedup saved")
}

func TestDedupHitRecordsADivergenceAndStoresTheDivergentCopy(t *testing.T) {
	data := payload(16 * 1024)
	f := newDedupFixture(t, "hit-mismatch").withPayload(data)

	f.indexer.setBlob(f.target.dedupKey, hashOf([]byte("something else entirely")), "somebody/elses/object.ssz")

	var outcome *dedupOutcome

	require.NoError(t, f.run(t, func(_ context.Context, o *dedupOutcome) error {
		outcome = o

		return nil
	}))

	require.Equal(t, f.target.finalLocation(hashOf(data)), outcome.Location, "the divergent copy keeps its own location")
	require.Nil(t, outcome.ContentMatchedAt, "a divergent row must never claim a match")

	uploads, hashes := f.fetcher.counts()
	require.Equal(t, 1, hashes)
	require.Equal(t, 1, uploads, "exactly one re-fetch")

	divergences := f.indexer.recorded()
	require.Len(t, divergences, 2)
	require.Equal(t, int32(1), divergences[0].GetAttempt().GetValue())
	require.Empty(t, divergences[0].GetLocation().GetValue(), "nothing was stored when the mismatch was first seen")
	require.Equal(t, hashOf(data), divergences[0].GetActualHash().GetValue())
	require.Equal(t, divergenceSeverityAlarm, divergences[0].GetSeverity().GetValue())
	require.Equal(t, int32(2), divergences[1].GetAttempt().GetValue())
	require.Equal(t, outcome.Location, divergences[1].GetLocation().GetValue())

	require.Equal(t, float64(2), testutil.ToFloat64(f.agent.metrics.payloadMismatch.WithLabelValues(string(BeaconStateQueue), f.agent.Config.Name)))
}

func TestDedupKeepsTheDivergenceWhenTheRefetchFails(t *testing.T) {
	data := payload(4096)
	f := newDedupFixture(t, "refetch-failed").withPayload(data)

	f.indexer.setBlob(f.target.dedupKey, hashOf([]byte("other bytes")), "somebody/elses/object.ssz")
	f.fetcher.uploadErr = goerrors.New("node stopped answering")

	created := 0

	err := f.run(t, func(context.Context, *dedupOutcome) error {
		created++

		return nil
	})
	require.ErrorIs(t, err, errPayloadDivergent)
	require.Zero(t, created, "no row may point at a payload that was never stored")

	divergences := f.indexer.recorded()
	require.Len(t, divergences, 1, "the record of the mismatch is the deliverable")
	require.Equal(t, int32(1), divergences[0].GetAttempt().GetValue())
}

func TestDedupRaceLoserWithIdenticalBytesLinksToTheWinner(t *testing.T) {
	data := payload(8192)
	f := newDedupFixture(t, "race-same-hash").withPayload(data)

	f.indexer.onCreateBlob = func(req *indexer.CreateBlobRequest) (*indexer.Blob, error) {
		return &indexer.Blob{
			DedupKey:    req.GetDedupKey(),
			ContentHash: req.GetContentHash(),
			Location:    wrapperspb.String("somebody/elses/identical.ssz"),
			State:       wrapperspb.String(blobStateReady),
		}, nil
	}

	var outcome *dedupOutcome

	require.NoError(t, f.run(t, func(_ context.Context, o *dedupOutcome) error {
		outcome = o

		return nil
	}))

	require.Equal(t, "somebody/elses/identical.ssz", outcome.Location)
	require.NotNil(t, outcome.ContentMatchedAt)
	require.Empty(t, f.indexer.recorded(), "identical bytes are not a divergence")
	require.Equal(t, []string{"somebody/elses/identical.ssz"}, storedObjects(t, f.base),
		"the winner's location is backed by our bytes and the redundant copy is removed")
	require.Equal(t, float64(1), testutil.ToFloat64(f.agent.metrics.blobReused.WithLabelValues(string(BeaconStateQueue), f.agent.Config.Name)))
}

func TestDedupRaceLoserWithDifferentBytesRecordsADivergence(t *testing.T) {
	data := payload(8192)
	f := newDedupFixture(t, "race-other-hash").withPayload(data)

	winnerHash := hashOf([]byte("the other agent's bytes"))

	f.indexer.onCreateBlob = func(req *indexer.CreateBlobRequest) (*indexer.Blob, error) {
		return &indexer.Blob{
			DedupKey:    req.GetDedupKey(),
			ContentHash: wrapperspb.String(winnerHash),
			Location:    wrapperspb.String("somebody/elses/different.ssz"),
			State:       wrapperspb.String(blobStateReady),
		}, nil
	}

	var outcome *dedupOutcome

	require.NoError(t, f.run(t, func(_ context.Context, o *dedupOutcome) error {
		outcome = o

		return nil
	}))

	require.Equal(t, f.target.finalLocation(hashOf(data)), outcome.Location, "our bytes stay where we put them")
	require.Nil(t, outcome.ContentMatchedAt)
	require.Equal(t, []string{outcome.Location}, storedObjects(t, f.base), "the divergent copy is the evidence and must survive")

	divergences := f.indexer.recorded()
	require.Len(t, divergences, 1)
	require.Equal(t, winnerHash, divergences[0].GetExpectedHash().GetValue())
	require.Equal(t, hashOf(data), divergences[0].GetActualHash().GetValue())
	require.Equal(t, int32(1), divergences[0].GetAttempt().GetValue())
	require.Equal(t, outcome.Location, divergences[0].GetLocation().GetValue())
}

func TestDedupTreatsAnAlreadyIndexedRowAsSuccess(t *testing.T) {
	data := payload(2048)
	f := newDedupFixture(t, "already-exists").withPayload(data)

	require.NoError(t, f.run(t, func(context.Context, *dedupOutcome) error {
		return status.Error(codes.AlreadyExists, "row already indexed")
	}))

	require.Equal(t, float64(1), testutil.ToFloat64(f.agent.metrics.itemExported.WithLabelValues(string(BeaconStateQueue), f.agent.Config.Name)))
}

func TestDedupSelfHealsOnceWhenTheStoredPayloadIsGone(t *testing.T) {
	data := payload(4096)
	f := newDedupFixture(t, "self-heal").withPayload(data)

	// The blob is found, so the first pass hashes only and links; the indexer
	// then reports that the payload went away underneath it.
	f.indexer.setBlob(f.target.dedupKey, hashOf(data), "collected/object.ssz")

	var (
		attempts  int
		locations []string
	)

	require.NoError(t, f.run(t, func(_ context.Context, o *dedupOutcome) error {
		attempts++

		locations = append(locations, o.Location)

		if attempts == 1 {
			// The row could not be linked because the payload it named has
			// been collected, so the indexer no longer holds it either.
			f.indexer.deleteBlob(f.target.dedupKey)

			return status.Error(codes.FailedPrecondition, "blob is gone")
		}

		return nil
	}))

	require.Equal(t, 2, attempts, "the row is written again, once, from a fresh read")
	require.Equal(t, "collected/object.ssz", locations[0])
	require.Equal(t, f.target.finalLocation(hashOf(data)), locations[1], "the healed row points at bytes that exist")

	uploads, hashes := f.fetcher.counts()
	require.Equal(t, 1, hashes)
	require.Equal(t, 1, uploads, "a consumed body is never reused")

	require.Equal(t, []string{locations[1]}, storedObjects(t, f.base))
}

func TestDedupRecordsNothingWhenTheTransferIsIncomplete(t *testing.T) {
	data := payload(4096)
	f := newDedupFixture(t, "incomplete").withPayload(data)

	f.indexer.setBlob(f.target.dedupKey, hashOf(data), "somebody/elses/object.ssz")

	// The payload arrives short of what it promised, which hashes perfectly
	// well and must therefore never be hashed into a decision.
	f.fetcher.contentLength = int64(len(data)) + 1

	created := 0

	err := f.run(t, func(context.Context, *dedupOutcome) error {
		created++

		return nil
	})
	require.ErrorIs(t, err, errIncompleteTransfer)
	require.Zero(t, created)
	require.Empty(t, f.indexer.recorded(), "a short read is not a divergence")
	require.Empty(t, storedObjects(t, f.base))

	require.Equal(t, float64(1), testutil.ToFloat64(f.agent.metrics.transferIncomplete.WithLabelValues(string(BeaconStateQueue), f.agent.Config.Name)))
	require.Zero(t, testutil.ToFloat64(f.agent.metrics.payloadVerified.WithLabelValues(string(BeaconStateQueue), f.agent.Config.Name)))
}

func TestDedupCollapsesConcurrentUploadsButNotReads(t *testing.T) {
	data := payload(16 * 1024)

	leader := newDedupFixture(t, "flight-leader").withPayload(data)

	follower := newDedupFixture(t, "flight-follower").withPayload(data)
	// Both agents are after the same payload, and in single mode they share
	// the process. Everything else about them is their own.
	follower.target.dedupKey = leader.target.dedupKey
	follower.agent.indexer = leader.indexer
	follower.target.save = leader.target.save
	follower.target.remove = leader.target.remove
	follower.fetcher.save = leader.target.save

	var (
		uploading = make(chan struct{})
		release   = make(chan struct{})
	)

	leader.fetcher.beforeUpload = func() {
		close(uploading)
		<-release
	}

	done := make(chan error, 2)

	go func() {
		done <- leader.run(t, func(context.Context, *dedupOutcome) error { return nil })
	}()

	<-uploading

	go func() {
		done <- follower.run(t, func(context.Context, *dedupOutcome) error { return nil })
	}()

	// The follower has no way to announce that it has joined the flight, so it
	// is given a moment to get there before the leader is let go.
	time.Sleep(100 * time.Millisecond)

	close(release)

	require.NoError(t, <-done)
	require.NoError(t, <-done)

	leaderUploads, leaderHashes := leader.fetcher.counts()
	followerUploads, followerHashes := follower.fetcher.counts()

	require.Equal(t, 1, leaderUploads)
	require.Zero(t, leaderHashes)
	require.Zero(t, followerUploads, "only one agent compresses and uploads the payload")
	require.Equal(t, 1, followerHashes, "the follower still reads its own node in full")

	require.Equal(t, 1, leader.indexer.getBlobs, "the follower waited on the leader instead of looking up itself")
	require.Equal(t, 1, leader.indexer.createBlobs)
}
