package agent

import (
	"context"
	goerrors "errors"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	dropReasonAttemptsExhausted  = "attempts_exhausted"
	dropReasonCircuitOpen        = "circuit_open"
	dropReasonDivergent          = "divergent"
	dropReasonEmptySlot          = "empty_slot"
	dropReasonIndexerUnavailable = "indexer_unavailable"
	dropReasonNotAvailable       = "not_available"
	dropReasonSlotMismatch       = "slot_mismatch"
	dropReasonStale              = "stale"
	dropReasonStoreUnavailable   = "store_unavailable"
	dropReasonUnsupported        = "unsupported"

	defaultAttemptBudget = 1
)

var (
	// errItemNotAvailable marks a fetch that failed because the artifact simply
	// is not there. It consumes an attempt but says nothing about the node.
	errItemNotAvailable = goerrors.New("item not available")

	// errItemStale marks an item that has aged past the window in which its
	// artifact can still be served, so attempting it is a guaranteed failure.
	errItemStale = goerrors.New("item is past its capture window")

	// errPayloadDivergent marks an item whose node served bytes that did not
	// match the stored payload, and whose re-fetch then failed. The divergence
	// record was already written, which was the point; retrying would only
	// write it again.
	errPayloadDivergent = goerrors.New("payload diverged from the stored copy")

	// errIndexerUnavailable marks a failure that belongs to the indexer rather
	// than to the node: a lookup or record RPC failed. Like a store outage it
	// hits every node at once, so it is never held against the node.
	errIndexerUnavailable = goerrors.New("indexer unavailable")

	// errSlotEmpty marks a slot the node holds no block for, whether it said so or answered with an earlier block.
	errSlotEmpty = goerrors.New("slot has no block")

	// errSlotMismatch marks a payload asked for by slot whose own slot field names another one.
	errSlotMismatch = goerrors.New("payload belongs to another slot")
)

// attemptBudgets bounds how many times a single queued item may be attempted.
// The numbers follow how long each artifact stays fetchable and how expensive a
// second try is, not how badly the item is wanted.
var attemptBudgets = map[Queue]int{
	BeaconStateQueue:              2,
	BeaconBlockQueue:              3,
	ExecutionPayloadEnvelopeQueue: 2,
	ExecutionBlockTraceQueue:      2,
}

func attemptBudget(kind Queue) int {
	if budget, ok := attemptBudgets[kind]; ok {
		return budget
	}

	return defaultAttemptBudget
}

// allowArtifact reports whether the node is currently believed able to serve
// this artifact. Queueing work for a node that has stopped serving it only
// delays whatever is behind it.
func (s *agent) allowArtifact(kind Queue) bool {
	if s.breaker.Allow(kind) {
		return true
	}

	s.metrics.IncrementItemDropped(kind, s.Config.Name, dropReasonCircuitOpen)

	return false
}

// runQueueItem runs one queued item under its attempt budget and the node's
// circuit breaker. Failures never escape: an item that runs out of attempts is
// dropped and counted, because a missed artifact is routine rather than an
// error condition. A non-zero return is the delay before runWorker retries an item set aside.
func (s *agent) runQueueItem(ctx context.Context, kind Queue, logCtx logrus.FieldLogger, retry retryPolicy, fn func(context.Context) error) time.Duration {
	if !s.allowArtifact(kind) {
		logCtx.Debug("Skipping item, node is not currently serving this artifact")

		return 0
	}

	var (
		err        error
		class      = failureTransient
		retryAfter = time.Duration(0)
		reason     = dropReasonAttemptsExhausted
	)

	for attempt := 1; attempt <= attemptBudget(kind); attempt++ {
		err = fn(ctx)
		if err == nil {
			s.breaker.RecordSuccess(kind)

			return 0
		}

		if ctx.Err() != nil {
			return 0
		}

		if goerrors.Is(err, errItemStale) {
			reason = dropReasonStale

			break
		}

		// Final for this slot on this node: another read would get the same answer.
		if goerrors.Is(err, errSlotEmpty) {
			reason = dropReasonEmptySlot

			break
		}

		if goerrors.Is(err, errSlotMismatch) {
			reason = dropReasonSlotMismatch

			break
		}

		if goerrors.Is(err, errItemNotAvailable) {
			reason = dropReasonNotAvailable

			if retry.revealBy.IsZero() {
				continue
			}

			// An immediate retry lands in the same moment that missed the reveal, so spread them across its window.
			if delay, ok := retry.untilRevealed(s.revealBackoff, time.Now()); ok {
				return s.retryItem(kind, logCtx, reason, err, delay)
			}

			break
		}

		if goerrors.Is(err, errPayloadDivergent) {
			reason = dropReasonDivergent

			break
		}

		if goerrors.Is(err, errStoreUnavailable) || goerrors.Is(err, errIndexerUnavailable) {
			// Shared by every node, so the node is not read again now; the item waits the outage out.
			reason = dropReasonStoreUnavailable
			if goerrors.Is(err, errIndexerUnavailable) {
				reason = dropReasonIndexerUnavailable
			}

			if delay, ok := s.outageBackoff.after(retry.retries); ok {
				return s.retryItem(kind, logCtx, reason, err, delay)
			}

			break
		}

		class, retryAfter = classifyFailure(kind, err)
		if class == failurePermanent {
			reason = dropReasonUnsupported

			break
		}

		reason = dropReasonAttemptsExhausted
	}

	// A stale or absent artifact is a property of the chain, not of the node, so
	// neither outcome is held against it. Nor is a divergence: the node answered
	// perfectly well, it just answered differently, and pausing it would stop
	// collecting the very evidence that makes the finding useful. Nor is a store
	// or indexer outage, either of which would otherwise pause every node in the
	// fleet at once. Nor is an empty slot, or a node answering one with another slot's payload.
	if reason != dropReasonStale &&
		reason != dropReasonEmptySlot &&
		reason != dropReasonSlotMismatch &&
		reason != dropReasonNotAvailable &&
		reason != dropReasonDivergent &&
		reason != dropReasonStoreUnavailable &&
		reason != dropReasonIndexerUnavailable {
		if s.breaker.RecordFailure(kind, class, retryAfter) {
			s.metrics.IncrementArtifactUnsupported(kind, s.Config.Name)

			logCtx.
				WithError(err).
				WithField("cooldown", s.breaker.cooldownFor(retryAfter).String()).
				Warn("Node appears unable to serve this artifact, pausing it")
		}
	}

	s.metrics.IncrementItemDropped(kind, s.Config.Name, reason)

	logCtx.
		WithError(err).
		WithField("reason", reason).
		Debug("Dropping queue item")

	return 0
}
