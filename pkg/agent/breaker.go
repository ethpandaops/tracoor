package agent

import (
	goerrors "errors"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/0xsequence/ethkit/ethrpc/jsonrpc"
	"github.com/ethpandaops/beacon/pkg/beacon/api"
)

const (
	// breakerFailureThreshold is how many consecutive failed items are needed
	// before a node is assumed to have stopped serving an artifact.
	breakerFailureThreshold = 3

	// breakerCooldown is how long a tripped pair is left alone before a single
	// probe is allowed through, so a node that gets upgraded recovers on its own.
	breakerCooldown = 5 * time.Minute

	// breakerMaxCooldown bounds a node-supplied Retry-After so one bad header
	// cannot silence an artifact indefinitely.
	breakerMaxCooldown = 30 * time.Minute

	// jsonRPCMethodNotFound is the JSON-RPC code an execution node returns for a
	// method it does not implement.
	jsonRPCMethodNotFound = -32601
)

type failureClass int

const (
	// failureTransient describes a failure that says nothing about whether the
	// node can serve the artifact at all.
	failureTransient failureClass = iota

	// failurePermanent describes a response that means "this node cannot serve
	// this artifact", so there is nothing to gain from further attempts.
	failurePermanent
)

// classifyFailure separates "this node cannot serve this artifact" from a
// transient failure, and extracts the node's own retry hint when it supplied
// one.
func classifyFailure(kind Queue, err error) (failureClass, time.Duration) {
	var rpcErr *jsonrpc.Error
	if goerrors.As(err, &rpcErr) && rpcErr.Code == jsonRPCMethodNotFound {
		return failurePermanent, 0
	}

	var statusErr *api.HTTPStatusError
	if !goerrors.As(err, &statusErr) {
		return failureTransient, 0
	}

	retryAfter := parseRetryAfter(statusErr.RetryAfter)

	switch statusErr.StatusCode {
	case http.StatusMethodNotAllowed, http.StatusNotImplemented:
		return failurePermanent, retryAfter
	case http.StatusNotFound:
		// On the state and block paths a 404 describes the slot rather than the
		// node: a state can be pruned and a slot can be empty while the endpoint
		// itself works perfectly.
		if kind == BeaconStateQueue || kind == BeaconBlockQueue {
			return failureTransient, retryAfter
		}

		return failurePermanent, retryAfter
	}

	// Everything else, 5xx included, is transient. Some clients answer a state
	// they cannot serve with a 500, which is a property of the request and not
	// of the node.
	return failureTransient, retryAfter
}

// parseRetryAfter reads either form of the Retry-After header. An absent,
// malformed or already-elapsed value yields zero, meaning "no hint".
func parseRetryAfter(value string) time.Duration {
	if value == "" {
		return 0
	}

	if seconds, err := strconv.Atoi(value); err == nil {
		if seconds <= 0 {
			return 0
		}

		return time.Duration(seconds) * time.Second
	}

	if at, err := http.ParseTime(value); err == nil {
		if d := time.Until(at); d > 0 {
			return d
		}
	}

	return 0
}

// breakerState tracks one artifact kind on one node. openUntil is zero while
// the pair is healthy.
type breakerState struct {
	failures  int
	openUntil time.Time
}

// circuitBreaker stops an agent asking its node for an artifact the node has
// repeatedly refused to serve. One agent covers one node, so the map key alone
// identifies a (node, artifact) pair.
type circuitBreaker struct {
	mu        sync.Mutex
	states    map[Queue]*breakerState
	threshold int
	cooldown  time.Duration
	now       func() time.Time
}

func newCircuitBreaker() *circuitBreaker {
	return &circuitBreaker{
		states:    make(map[Queue]*breakerState),
		threshold: breakerFailureThreshold,
		cooldown:  breakerCooldown,
		now:       time.Now,
	}
}

// Allow reports whether work for this artifact should be attempted. Once the
// cooldown has elapsed it allows probes through again, so the breaker can never
// latch permanently.
func (b *circuitBreaker) Allow(kind Queue) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	state, ok := b.states[kind]
	if !ok || state.openUntil.IsZero() {
		return true
	}

	return !b.now().Before(state.openUntil)
}

// RecordSuccess closes the breaker for this artifact.
func (b *circuitBreaker) RecordSuccess(kind Queue) {
	b.mu.Lock()
	defer b.mu.Unlock()

	delete(b.states, kind)
}

// RecordFailure accounts for one failed item and reports whether that tripped
// the breaker. A failed probe from the half-open window re-opens it straight
// away, as does a single permanently-classified failure.
func (b *circuitBreaker) RecordFailure(kind Queue, class failureClass, retryAfter time.Duration) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	state, ok := b.states[kind]
	if !ok {
		state = &breakerState{}
		b.states[kind] = state
	}

	halfOpen := !state.openUntil.IsZero() && !b.now().Before(state.openUntil)

	switch {
	case halfOpen:
	case class == failurePermanent:
		state.failures = b.threshold
	default:
		state.failures++
	}

	if !halfOpen && state.failures < b.threshold {
		return false
	}

	state.openUntil = b.now().Add(b.cooldownFor(retryAfter))

	return true
}

// cooldownFor prefers the node's own hint over the default, bounded so a
// hostile or mistaken header cannot silence an artifact for long.
func (b *circuitBreaker) cooldownFor(retryAfter time.Duration) time.Duration {
	if retryAfter <= 0 {
		return b.cooldown
	}

	if retryAfter > breakerMaxCooldown {
		return breakerMaxCooldown
	}

	return retryAfter
}
