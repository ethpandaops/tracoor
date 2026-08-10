package agent

import (
	"fmt"
	"sync"
)

// dropReasonDuplicate marks an item that was already queued or in flight when
// something asked for it again.
const dropReasonDuplicate = "duplicate"

// pendingItems is the set of items an agent has accepted but not yet finished.
//
// A reorg re-derives every slot between the reorg point and head across four
// artifact kinds, and the queues block on send, so without this the event
// handler stalls behind duplicates of work already queued. Claims arrive from
// beacon event callbacks and the scheduler; releases come from the queue
// workers, so the set is guarded.
type pendingItems struct {
	mu    sync.Mutex
	items map[string]struct{}
}

func newPendingItems() *pendingItems {
	return &pendingItems{items: make(map[string]struct{})}
}

// claim reserves key, reporting false if it is already queued or in flight.
func (p *pendingItems) claim(key string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.items[key]; exists {
		return false
	}

	p.items[key] = struct{}{}

	return true
}

// release gives up a claim. It runs whether the item succeeded or was dropped:
// the set tracks what is outstanding, not what worked.
func (p *pendingItems) release(key string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	delete(p.items, key)
}

// pendingKey names one queued item. The kind is part of the key because the
// same slot is queued independently for several artifacts.
func pendingKey(kind Queue, identifier string) string {
	return fmt.Sprintf("%s/%s", kind, identifier)
}

// claimQueueItem reserves an item for a queue, counting and reporting the
// duplicates it turns away.
func (s *agent) claimQueueItem(kind Queue, key string) bool {
	if s.pending.claim(key) {
		return true
	}

	s.metrics.IncrementItemDropped(kind, s.Config.Name, dropReasonDuplicate)

	return false
}

// releaseQueueItem is called by a queue worker once it has finished with an
// item, whatever the outcome, so the same work can be queued again later.
func (s *agent) releaseQueueItem(key string) {
	s.pending.release(key)
}
