package agent

import (
	"fmt"
	"sync"
)

const (
	// dropReasonDuplicate marks an item that was already queued or in flight
	// when something asked for it again, and whose repeat carried nothing new.
	dropReasonDuplicate = "duplicate"

	// dropReasonRedoRequested marks a colliding enqueue that was folded into
	// the claim it collided with rather than lost: the worker holding the claim
	// re-enqueues the item once when it releases it.
	dropReasonRedoRequested = "redo_requested"
)

// pendingItems is the set of items an agent has accepted but not yet finished.
//
// A reorg re-derives every slot between the reorg point and head across four
// artifact kinds, and the queues block on send, so without this the event
// handler stalls behind duplicates of work already queued. Claims arrive from
// beacon event callbacks and the scheduler; releases come from the queue
// workers, so the set is guarded.
type pendingItems struct {
	mu sync.Mutex
	// items maps each claim to whether a colliding claim asked for the item to
	// be run again once the current holder releases it.
	items map[string]bool
}

func newPendingItems() *pendingItems {
	return &pendingItems{items: make(map[string]bool)}
}

// claim reserves key, reporting false if it is already queued or in flight.
// redo is what a collision means for the caller's item: a reorg re-derivation
// colliding with an in-flight fetch names newer bytes than the holder resolved,
// so the item must run again once the claim is released; a scheduled scan
// colliding with itself carries nothing the pending item does not.
func (p *pendingItems) claim(key string, redo bool) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.items[key]; exists {
		if redo {
			p.items[key] = true
		}

		return false
	}

	p.items[key] = false

	return true
}

// release gives up a claim, reporting whether a colliding claim asked for the
// item to be run again. It runs whether the item succeeded or was dropped: the
// set tracks what is outstanding, not what worked.
func (p *pendingItems) release(key string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	redo := p.items[key]

	delete(p.items, key)

	return redo
}

// pendingKey names one queued item. The kind is part of the key because the
// same slot is queued independently for several artifacts.
func pendingKey(kind Queue, identifier string) string {
	return fmt.Sprintf("%s/%s", kind, identifier)
}

// claimQueueItem reserves an item for a queue. A collision is counted rather
// than lost silently: as a redo when the pending item will be re-enqueued on
// release, as a duplicate when the repeat carried nothing new.
func (s *agent) claimQueueItem(kind Queue, key string, redo bool) bool {
	if s.pending.claim(key, redo) {
		return true
	}

	reason := dropReasonDuplicate
	if redo {
		reason = dropReasonRedoRequested
	}

	s.metrics.IncrementItemDropped(kind, s.Config.Name, reason)

	return false
}

// releaseQueueItem is called by a queue worker once it has finished with an
// item, whatever the outcome, so the same work can be queued again later. It
// reports whether a colliding enqueue asked for the item to be run again.
func (s *agent) releaseQueueItem(key string) bool {
	return s.pending.release(key)
}
