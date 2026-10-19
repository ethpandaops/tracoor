package agent

import (
	"context"
	"math"

	"github.com/ethpandaops/beacon/pkg/beacon/state"
	eth2v1 "github.com/ethpandaops/go-eth2-client/api/v1"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/sirupsen/logrus"
)

// farFutureEpoch is how a spec marks a fork that is not scheduled.
const farFutureEpoch = phase0.Epoch(math.MaxUint64)

// envelopeForkScheduled reports whether the spec schedules any fork that carries execution payload envelopes.
func envelopeForkScheduled(spec *state.Spec) bool {
	if spec == nil {
		return false
	}

	for _, name := range envelopeForks {
		fork, err := spec.ForkEpochs.GetByName(name)
		if err == nil && fork.Epoch != farFutureEpoch {
			return true
		}
	}

	return false
}

// watchExecutionPayloads opens the node's execution_payload stream once, and only on a node whose spec has envelopes.
func (s *agent) watchExecutionPayloads(ctx context.Context, spec *state.Spec) {
	if !s.Config.Ethereum.Features.GetFetchExecutionPayloadEnvelope() || !envelopeForkScheduled(spec) {
		return
	}

	s.payloadEventsMu.Lock()
	defer s.payloadEventsMu.Unlock()

	if s.payloadEventsSubscribed {
		return
	}

	err := s.node.Beacon().SubscribeExecutionPayloads(ctx, func(_ context.Context, event *eth2v1.ExecutionPayloadEvent) {
		s.onExecutionPayload(ctx, event)
	})
	if err != nil {
		s.log.WithError(err).Warn("Failed to subscribe to execution payload events, envelopes rely on block events alone")

		return
	}

	s.payloadEventsSubscribed = true

	s.log.Info("Subscribed to execution payload events")
}

// onExecutionPayload queues the envelope of a payload the node just imported, however late its builder revealed it.
func (s *agent) onExecutionPayload(ctx context.Context, event *eth2v1.ExecutionPayloadEvent) {
	logCtx := s.log.WithFields(logrus.Fields{
		"event_topic": "execution_payload",
		logKeySlot:    event.Slot,
		"root":        rootAsString(event.BlockRoot),
		logKeyPurpose: "execution_payload_envelope",
	})

	if ignore, err := s.node.ShouldIgnoreEventFromSlot(event.Slot); err != nil {
		logCtx.WithError(err).Error("Failed to check if the payload is too old to fetch")

		return
	} else if ignore {
		logCtx.Debug("Skipping queueing execution payload envelope as the payload is too old to fetch")

		return
	}

	// Off the stream's reader: the queue blocks when full.
	s.workers.Go(func() { s.enqueueExecutionPayloadEnvelope(ctx, event.Slot) })
}
