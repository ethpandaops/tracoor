package agent

import (
	"bytes"
	"context"
	goerrors "errors"
	"fmt"
	"path"
	"time"

	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/tracoor/pkg/compression"
	"github.com/ethpandaops/tracoor/pkg/proto/tracoor/indexer"
	"github.com/ethpandaops/tracoor/pkg/store"
	"github.com/google/uuid"
	"golang.org/x/sync/singleflight"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const (
	// divergenceSeverityAlarm marks a mismatch on bytes the spec pins down, so
	// two nodes disagreeing about them is somebody's bug.
	divergenceSeverityAlarm = "alarm"

	// divergenceSeverityNotice marks a mismatch where the dedup key is a guess
	// rather than a promise, so a difference may be nobody's fault.
	divergenceSeverityNotice = "notice"

	// blobStateReady is the only blob state that may be linked to. Anything
	// else is being collected and its object may be gone by the time we look.
	blobStateReady = "ready"

	// contentHashSuffixLength is how much of the content hash is carried in a
	// location. It only has to separate payloads that disagree under one dedup
	// key, which is a handful of candidates rather than a global namespace.
	contentHashSuffixLength = 8
)

// dedupFlight collapses the compress-and-upload work for one dedup key. In
// single mode every agent in the process reacts to the same event for the same
// slot, so misses arrive together; the key is globally unique, so a
// package-level group carries no configuration and cannot leak between agents.
var dedupFlight singleflight.Group

// dedupTarget names the payload an artifact is expected to contain and where a
// copy of it lives. The dedup key is known before a single byte is fetched,
// which is the property the whole path rests on.
type dedupTarget struct {
	kind     store.DataType
	queue    Queue
	network  string
	dedupKey string

	// directory and identity compose the location. The node is deliberately
	// absent: the object is addressed by what determines its bytes.
	directory string
	identity  string
	extension string

	// slot and identifier are what a human reads on a divergence record. Slot
	// is zero for the execution kinds.
	slot       uint64
	identifier string

	severity string

	save   saveFunc
	remove func(ctx context.Context, location string) error
}

func (t *dedupTarget) flightKey() string {
	return fmt.Sprintf("%s/%s/%s", t.kind, t.network, t.dedupKey)
}

// finalLocation is where a payload with this content hash belongs. Two nodes
// that disagree under one dedup key land on different objects, so neither can
// overwrite the evidence the other left.
func (t *dedupTarget) finalLocation(contentHash string) string {
	suffix := contentHash
	if len(suffix) > contentHashSuffixLength {
		suffix = suffix[:contentHashSuffixLength]
	}

	return path.Join(t.directory, fmt.Sprintf("%s-%s%s", t.identity, suffix, t.extension))
}

// stagingLocation is where a payload is written while its content hash is still
// unknown. It is unique per node and per attempt, so a concurrent upload of the
// same key never shares it.
func (t *dedupTarget) stagingLocation(node string) string {
	return path.Join(t.directory, fmt.Sprintf(".tmp-%s-%s", node, uuid.NewString()))
}

// payloadFetcher produces a fresh read of one artifact from this node. Every
// call reads the node again: a consumed body can never be reused, and a payload
// somebody else already stored is still read here in full.
type payloadFetcher interface {
	// Hash reads the payload once and reports what it hashed to.
	Hash(ctx context.Context) (streamResult, error)
	// Upload reads the payload once, compressing and storing it at location.
	Upload(ctx context.Context, location string) (string, streamResult, error)
}

// streamFetcher carries a payload that can be read straight off the wire.
type streamFetcher struct {
	open       func(ctx context.Context) (*payloadSource, error)
	compressor *compression.Compressor
	save       saveFunc
}

func (f *streamFetcher) Hash(ctx context.Context) (streamResult, error) {
	src, err := f.open(ctx)
	if err != nil {
		return streamResult{}, err
	}

	return hashSource(src)
}

func (f *streamFetcher) Upload(ctx context.Context, location string) (string, streamResult, error) {
	src, err := f.open(ctx)
	if err != nil {
		return "", streamResult{}, err
	}

	return streamSource(ctx, f.compressor, src, f.save, location)
}

// bufferedFetcher carries a payload that has to be held whole before it can be
// used, which today is the execution trace: its bytes arrive inside a JSON-RPC
// envelope that has to be parsed before the result can be seen.
type bufferedFetcher struct {
	fetch      func(ctx context.Context) ([]byte, error)
	compressor *compression.Compressor
	save       saveFunc
}

func (f *bufferedFetcher) Hash(ctx context.Context) (streamResult, error) {
	data, err := f.fetch(ctx)
	if err != nil {
		return streamResult{}, err
	}

	return hashSource(sourceFromBytes(data))
}

func (f *bufferedFetcher) Upload(ctx context.Context, location string) (string, streamResult, error) {
	data, err := f.fetch(ctx)
	if err != nil {
		return "", streamResult{}, err
	}

	result, err := hashSource(sourceFromBytes(data))
	if err != nil {
		return "", streamResult{}, err
	}

	compressed, err := f.compressor.Compress(&data, compression.Default)
	if err != nil {
		return "", streamResult{}, fmt.Errorf("failed to compress payload: %w", err)
	}

	// The raw copy has served its purpose, so only the compressed one is held
	// for the duration of the store write.
	data = nil

	saved, err := f.save(ctx, &store.SaveParams{
		Data:            bytes.NewReader(compressed),
		Location:        location,
		ContentEncoding: compression.Default.ContentEncoding,
	})
	if err != nil {
		return "", streamResult{}, err
	}

	result.CompressedSize = int64(len(compressed))

	return saved, result, nil
}

// dedupOutcome is everything an artifact row needs to say about the bytes this
// node served: what they hashed to, where a copy of them lives, and whether
// they were compared against a payload somebody else had already stored.
type dedupOutcome struct {
	Location    string
	ContentHash string
	VerifiedAt  time.Time
	// ContentMatchedAt is unset unless the hash was compared against an
	// existing payload and matched.
	ContentMatchedAt *time.Time
	RawSize          int64
	CompressedSize   int64
}

// contentMatchedAt is nil unless the payload was compared against one that was
// already stored and matched it, which is what separates a row that agrees with
// the fleet from one that is simply the first of its kind.
func contentMatchedAt(outcome *dedupOutcome) *timestamppb.Timestamp {
	if outcome.ContentMatchedAt == nil {
		return nil
	}

	return timestamppb.New(*outcome.ContentMatchedAt)
}

// ownUpload is a payload this call read from its own node and stored.
type ownUpload struct {
	Location string
	Result   streamResult
}

// blobClaim is the outcome of resolving a dedup key: the payload that is
// canonical for it, plus this call's own upload when it was the one that
// stored it.
type blobClaim struct {
	blob *indexer.Blob
	own  *ownUpload
}

// indexDeduplicated runs one artifact through verified dedup and hands the
// result to create, which writes the row.
//
// Every node is read in full and hashed no matter what: only the compress and
// upload work is shared. Recording that a node served bytes nobody read from it
// would be fabricated agreement, which is the one thing this path exists to
// prevent.
func (s *agent) indexDeduplicated(
	ctx context.Context,
	target *dedupTarget,
	fetcher payloadFetcher,
	create func(ctx context.Context, outcome *dedupOutcome) error,
) error {
	outcome, err := s.resolveOutcome(ctx, target, fetcher)
	if err != nil {
		return err
	}

	err = create(ctx, outcome)

	switch status.Code(err) { //nolint:exhaustive // only these codes change what happens next.
	case codes.OK, codes.AlreadyExists:
		// AlreadyExists means another path indexed this row first. The work was
		// not wasted either way: the payload was read from this node in full.
		s.recordIndexed(target, outcome)

		return nil
	case codes.FailedPrecondition:
		// The payload we linked to went away underneath us. Store it again from
		// a fresh read rather than pointing a row at bytes that may be gone.
		s.log.
			WithField("kind", string(target.kind)).
			WithField("dedup_key", target.dedupKey).
			Debug("Stored payload disappeared while indexing, re-storing it")

		healed, herr := s.claimByUploading(ctx, target, fetcher)
		if herr != nil {
			return herr
		}

		outcome, herr = s.settleOwnUpload(ctx, target, healed)
		if herr != nil {
			return herr
		}

		if cerr := create(ctx, outcome); cerr != nil && status.Code(cerr) != codes.AlreadyExists {
			return cerr
		}

		s.recordIndexed(target, outcome)

		return nil
	default:
		return err
	}
}

// recordIndexed accounts for an artifact that made it all the way to a row.
func (s *agent) recordIndexed(target *dedupTarget, outcome *dedupOutcome) {
	s.metrics.IncrementItemExported(target.queue, s.Config.Name)

	s.log.
		WithField("kind", string(target.kind)).
		WithField("dedup_key", target.dedupKey).
		WithField("location", outcome.Location).
		WithField("content_hash", outcome.ContentHash).
		WithField("raw_size", outcome.RawSize).
		WithField("compressed_size", outcome.CompressedSize).
		Debug("Indexed artifact")
}

// resolveOutcome reads this node's copy of the payload and decides what its row
// should point at.
func (s *agent) resolveOutcome(ctx context.Context, target *dedupTarget, fetcher payloadFetcher) (*dedupOutcome, error) {
	claim, err := s.resolveBlob(ctx, target, fetcher)
	if err != nil {
		return nil, err
	}

	if claim.own != nil {
		return s.settleOwnUpload(ctx, target, claim)
	}

	return s.verifyAgainstBlob(ctx, target, fetcher, claim.blob)
}

// resolveBlob finds the payload that is canonical for this dedup key, storing
// it when nobody has. The returned claim carries an upload only when this call
// is the one that performed it: a caller that waited on somebody else's upload
// still has its own node to read.
func (s *agent) resolveBlob(ctx context.Context, target *dedupTarget, fetcher payloadFetcher) (*blobClaim, error) {
	led := false

	value, err, _ := dedupFlight.Do(target.flightKey(), func() (any, error) {
		led = true

		return s.lookupOrClaim(ctx, target, fetcher)
	})
	if err != nil {
		if led {
			return nil, err
		}

		// The leader's failure describes the leader's node, not this one. A
		// follower that inherited it would spend an attempt, and blame a
		// circuit breaker, for a node it never asked.
		return s.lookupOrClaim(ctx, target, fetcher)
	}

	claim, ok := value.(*blobClaim)
	if !ok {
		return nil, fmt.Errorf("unexpected dedup result of type %T", value)
	}

	if !led {
		// Somebody else's upload is not this node's evidence.
		return &blobClaim{blob: claim.blob}, nil
	}

	return claim, nil
}

// lookupOrClaim asks whether the payload for this dedup key is already stored
// and stores it when it is not.
func (s *agent) lookupOrClaim(ctx context.Context, target *dedupTarget, fetcher payloadFetcher) (*blobClaim, error) {
	blob, found, err := s.getBlob(ctx, target)
	if err != nil {
		return nil, err
	}

	if found {
		return &blobClaim{blob: blob}, nil
	}

	return s.claimByUploading(ctx, target, fetcher)
}

// claimByUploading reads the payload from this node, stores it, and claims the
// dedup key for it. The claim can be lost to another agent that got there
// first, in which case the response says whose bytes are canonical.
func (s *agent) claimByUploading(ctx context.Context, target *dedupTarget, fetcher payloadFetcher) (*blobClaim, error) {
	own, err := s.uploadPayload(ctx, target, fetcher)
	if err != nil {
		return nil, err
	}

	rsp, err := s.indexer.CreateBlob(ctx, &indexer.CreateBlobRequest{
		Kind:            wrapperspb.String(string(target.kind)),
		Network:         wrapperspb.String(target.network),
		DedupKey:        wrapperspb.String(target.dedupKey),
		ContentHash:     wrapperspb.String(own.Result.ContentHash),
		Location:        wrapperspb.String(own.Location),
		ContentEncoding: wrapperspb.String(compression.Default.ContentEncoding),
		RawSize:         wrapperspb.Int64(own.Result.RawSize),
		CompressedSize:  wrapperspb.Int64(own.Result.CompressedSize),
	})
	if err != nil {
		// The object stays where it is. Removing it would destroy a payload
		// other rows may already point at if the call did land and only its
		// response was lost.
		return nil, fmt.Errorf("failed to record payload: %w", err)
	}

	if rsp.GetBlob() == nil {
		return nil, fmt.Errorf("indexer returned no payload for dedup key %s", target.dedupKey)
	}

	return &blobClaim{blob: rsp.GetBlob(), own: own}, nil
}

// settleOwnUpload decides what a row should point at when this node did the
// storing. Winning outright is the common case; the rest is what happens when
// another agent claimed the same key first.
func (s *agent) settleOwnUpload(ctx context.Context, target *dedupTarget, claim *blobClaim) (*dedupOutcome, error) {
	var (
		now      = time.Now()
		ours     = claim.own
		winner   = claim.blob
		outcome  = &dedupOutcome{ContentHash: ours.Result.ContentHash, VerifiedAt: now, RawSize: ours.Result.RawSize, CompressedSize: ours.Result.CompressedSize}
		wonHash  = winner.GetContentHash().GetValue() == ours.Result.ContentHash
		location = winner.GetLocation().GetValue()
	)

	if state := winner.GetState().GetValue(); state != "" && state != blobStateReady {
		// The winning row is on its way out, so its object may already be gone.
		// Ours is the copy that certainly exists.
		outcome.Location = ours.Location

		s.metrics.IncrementBlobCreated(target.queue, s.Config.Name)

		return outcome, nil
	}

	if !wonHash {
		// Two nodes disagree about what belongs under one dedup key. Our copy
		// is already at its own hash-suffixed location, so both survive.
		s.recordDivergence(ctx, target, winner.GetContentHash().GetValue(), ours.Result.ContentHash, 1, ours.Location)

		outcome.Location = ours.Location

		return outcome, nil
	}

	outcome.Location = location

	if location == ours.Location {
		s.metrics.IncrementBlobCreated(target.queue, s.Config.Name)

		return outcome, nil
	}

	// Somebody else stored identical bytes under a different location — usually
	// a concurrent agent, but after a tombstone resurrect the recorded location
	// may have no object behind it yet. Publish ours there before linking: the
	// bytes are identical, so overwriting an existing object is a no-op, and it
	// guarantees the location a row is about to reference actually resolves.
	if err := s.store.Copy(ctx, &store.CopyParams{Source: ours.Location, Destination: location}); err != nil {
		s.log.
			WithField("source", ours.Location).
			WithField("destination", location).
			WithError(err).
			Warn("Failed to publish payload at the recorded blob location; keeping own copy")

		outcome.Location = ours.Location

		s.metrics.IncrementBlobCreated(target.queue, s.Config.Name)

		return outcome, nil
	}

	// Ours is redundant now that the recorded location is backed. Only remove
	// it once the locations are known to differ: identical bytes usually
	// produce the identical path, and deleting that would take the canonical
	// object with it.
	if err := target.remove(ctx, ours.Location); err != nil {
		s.log.
			WithField("location", ours.Location).
			WithError(err).
			Warn("Failed to remove a redundant copy of a stored payload")
	}

	outcome.ContentMatchedAt = &now

	s.metrics.IncrementBlobReused(target.queue, s.Config.Name)

	return outcome, nil
}

// verifyAgainstBlob reads this node's copy of a payload somebody else has
// already stored, and compares. The bytes are discarded as they are read, so
// the cost is one streamed hash rather than a second copy of the payload.
func (s *agent) verifyAgainstBlob(
	ctx context.Context,
	target *dedupTarget,
	fetcher payloadFetcher,
	blob *indexer.Blob,
) (*dedupOutcome, error) {
	result, err := s.readPayloadOnce(ctx, target, fetcher.Hash)
	if err != nil {
		return nil, err
	}

	now := time.Now()

	if result.ContentHash == blob.GetContentHash().GetValue() {
		s.metrics.IncrementBlobReused(target.queue, s.Config.Name)

		return &dedupOutcome{
			Location:         blob.GetLocation().GetValue(),
			ContentHash:      result.ContentHash,
			VerifiedAt:       now,
			ContentMatchedAt: &now,
			RawSize:          result.RawSize,
		}, nil
	}

	// The record of the mismatch is the deliverable, so it is written before
	// anything is attempted with the payload itself.
	s.recordDivergence(ctx, target, blob.GetContentHash().GetValue(), result.ContentHash, 1, "")

	own, err := s.uploadPayload(ctx, target, fetcher)
	if err != nil {
		s.log.
			WithField("kind", string(target.kind)).
			WithField("dedup_key", target.dedupKey).
			WithError(err).
			Debug("Failed to re-fetch a diverging payload, the divergence record stands on its own")

		return nil, fmt.Errorf("%w: %w", errPayloadDivergent, err)
	}

	s.recordDivergence(ctx, target, blob.GetContentHash().GetValue(), own.Result.ContentHash, 2, own.Location)

	return &dedupOutcome{
		Location:       own.Location,
		ContentHash:    own.Result.ContentHash,
		VerifiedAt:     time.Now(),
		RawSize:        own.Result.RawSize,
		CompressedSize: own.Result.CompressedSize,
	}, nil
}

// uploadPayload stores one read of the payload. The destination is not known
// until the content hash is, so the bytes land on a staging location first and
// are copied into place once they can be named.
func (s *agent) uploadPayload(ctx context.Context, target *dedupTarget, fetcher payloadFetcher) (*ownUpload, error) {
	staged := target.stagingLocation(s.Config.Name)

	var saved string

	result, err := s.readPayloadOnce(ctx, target, func(ctx context.Context) (streamResult, error) {
		location, uploaded, uerr := fetcher.Upload(ctx, staged)
		saved = location

		return uploaded, uerr
	})
	if err != nil {
		return nil, err
	}

	final := target.finalLocation(result.ContentHash)

	if err := s.store.Copy(ctx, &store.CopyParams{Source: saved, Destination: final}); err != nil {
		// Nothing points at the staged object, so it is safe to take with us.
		if rerr := target.remove(ctx, saved); rerr != nil {
			s.log.WithField("location", saved).WithError(rerr).Debug("Failed to remove a staged payload")
		}

		return nil, fmt.Errorf("failed to publish payload: %w", err)
	}

	if err := target.remove(ctx, saved); err != nil {
		s.log.WithField("location", saved).WithError(err).Warn("Failed to remove a staged payload")
	}

	return &ownUpload{Location: final, Result: result}, nil
}

// readPayloadOnce runs one read of the node and accounts for how it went. A
// payload that did not arrive in full is counted and refused: nothing derived
// from a truncated body may reach a decision.
func (s *agent) readPayloadOnce(
	ctx context.Context,
	target *dedupTarget,
	read func(ctx context.Context) (streamResult, error),
) (streamResult, error) {
	result, err := read(ctx)
	if err != nil {
		if goerrors.Is(err, errIncompleteTransfer) {
			s.metrics.IncrementTransferIncomplete(target.queue, s.Config.Name)
		}

		return streamResult{}, err
	}

	s.metrics.IncrementPayloadVerified(target.queue, s.Config.Name)

	return result, nil
}

// getBlob reports whether the payload for this dedup key is already stored. A
// payload that is being collected is treated as absent, because its object may
// be gone by the time a row points at it.
func (s *agent) getBlob(ctx context.Context, target *dedupTarget) (*indexer.Blob, bool, error) {
	rsp, err := s.indexer.GetBlob(ctx, &indexer.GetBlobRequest{
		Kind:     wrapperspb.String(string(target.kind)),
		Network:  wrapperspb.String(target.network),
		DedupKey: wrapperspb.String(target.dedupKey),
	})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, false, nil
		}

		return nil, false, fmt.Errorf("failed to look up payload: %w", err)
	}

	blob := rsp.GetBlob()
	if blob == nil || blob.GetContentHash().GetValue() == "" {
		return nil, false, nil
	}

	if state := blob.GetState().GetValue(); state != "" && state != blobStateReady {
		return nil, false, nil
	}

	return blob, true, nil
}

// recordDivergence writes the finding. It is best effort in the sense that a
// failure to write it must not also cost the payload, but it is attempted
// before anything else is done with the diverging bytes.
func (s *agent) recordDivergence(
	ctx context.Context,
	target *dedupTarget,
	expected, actual string,
	attempt int32,
	location string,
) {
	s.metrics.IncrementPayloadMismatch(target.queue, s.Config.Name)

	s.log.
		WithField("kind", string(target.kind)).
		WithField("dedup_key", target.dedupKey).
		WithField("expected_hash", expected).
		WithField("actual_hash", actual).
		WithField("attempt", attempt).
		Warn("Node served a payload that does not match the one already stored")

	if _, err := s.indexer.CreatePayloadDivergence(ctx, &indexer.CreatePayloadDivergenceRequest{
		ObservedAt:   timestamppb.New(time.Now()),
		Network:      wrapperspb.String(target.network),
		Kind:         wrapperspb.String(string(target.kind)),
		Node:         wrapperspb.String(s.Config.Name),
		DedupKey:     wrapperspb.String(target.dedupKey),
		ExpectedHash: wrapperspb.String(expected),
		ActualHash:   wrapperspb.String(actual),
		Slot:         wrapperspb.UInt64(target.slot),
		Identifier:   wrapperspb.String(target.identifier),
		Attempt:      wrapperspb.Int32(attempt),
		Severity:     wrapperspb.String(target.severity),
		Location:     wrapperspb.String(location),
	}); err != nil {
		s.log.
			WithField("kind", string(target.kind)).
			WithField("dedup_key", target.dedupKey).
			WithError(err).
			Error("Failed to record a payload divergence")
	}
}

// beaconStateDedupKey and its siblings name the bytes an artifact should
// contain. The indexer derives the same keys from the columns of a stored row,
// so these formats are part of the contract between them.
func beaconStateDedupKey(slot phase0.Slot, stateRoot string) string {
	return fmt.Sprintf("%d/%s", slot, stateRoot)
}

func beaconBlockDedupKey(slot phase0.Slot, blockRoot string) string {
	return fmt.Sprintf("%d/%s", slot, blockRoot)
}

func executionPayloadEnvelopeDedupKey(slot phase0.Slot, blockRoot string) string {
	return fmt.Sprintf("%d/%s", slot, blockRoot)
}

// executionBlockTraceDedupKey has to carry the trace parameters as well as the
// block: they are per-agent configuration, and two agents that disagree about
// them produce different bytes for the same block from the same client.
func executionBlockTraceDedupKey(
	blockNumber uint64,
	blockHash string,
	implementation string,
	nodeVersion string,
	disableMemory, disableStack, disableStorage bool,
) string {
	return fmt.Sprintf(
		"%d/%s/%s/%s/%s%s%s",
		blockNumber,
		blockHash,
		implementation,
		nodeVersion,
		traceParameterFlag(disableMemory),
		traceParameterFlag(disableStack),
		traceParameterFlag(disableStorage),
	)
}

func traceParameterFlag(enabled bool) string {
	if enabled {
		return "1"
	}

	return "0"
}
