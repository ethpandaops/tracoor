package agent

import (
	"context"
	goerrors "errors"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	dropReasonAttemptsExhausted = "attempts_exhausted"
	dropReasonCircuitOpen       = "circuit_open"
	dropReasonDivergent         = "divergent"
	dropReasonNotAvailable      = "not_available"
	dropReasonStale             = "stale"
	dropReasonStoreUnavailable  = "store_unavailable"
	dropReasonUnsupported       = "unsupported"

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
// error condition.
func (s *agent) runQueueItem(ctx context.Context, kind Queue, logCtx logrus.FieldLogger, fn func(context.Context) error) {
	if !s.allowArtifact(kind) {
		logCtx.Debug("Skipping item, node is not currently serving this artifact")

		return
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

			return
		}

		if ctx.Err() != nil {
			return
		}

		if goerrors.Is(err, errItemStale) {
			reason = dropReasonStale

			break
		}

		if goerrors.Is(err, errItemNotAvailable) {
			reason = dropReasonNotAvailable

			continue
		}

		if goerrors.Is(err, errPayloadDivergent) {
			reason = dropReasonDivergent

			break
		}

		if goerrors.Is(err, errStoreUnavailable) {
			// The store could not take the payload. Retrying would re-read the
			// whole artifact from a node that has done nothing wrong, and a
			// store that is down is down for every node at once.
			reason = dropReasonStoreUnavailable

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
	// outage, which would otherwise pause every node in the fleet at once.
	if reason != dropReasonStale &&
		reason != dropReasonNotAvailable &&
		reason != dropReasonDivergent &&
		reason != dropReasonStoreUnavailable {
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
}
