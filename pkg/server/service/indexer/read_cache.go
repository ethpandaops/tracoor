package indexer

import (
	"context"
	"crypto/sha256"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	readCacheHit    = "hit"
	readCacheMiss   = "miss"
	readCacheShared = "shared"
)

// readCache memoizes the responses of the filter-dropdown (ListUnique*Values) and Count*
// handlers for a short TTL. The web UI issues both on every page load, and each one is a
// database walk: the dropdowns a query per field, the counts an index scan over every row the
// filter matches. Uncached, a crawler or scraper replaying them turns straight into database
// CPU. The UI refetches counts once they are 6s old and filter values at 60s, so a TTL under
// 6s never shows a page an older answer than the UI would have kept on its own.
//
// Concurrent misses on one key share a single load, so a burst of identical requests costs
// one query rather than one per request. Only successful responses are kept. A load runs for
// as long as any caller still waits on it and is cancelled once they have all gone, as the
// query behind an uncached request would be.
type readCache struct {
	ttl     time.Duration
	entries *lru.Cache[string, readCacheEntry]
	metrics *Metrics

	mu    sync.Mutex
	loads map[string]*readLoad
}

type readCacheEntry struct {
	resp    proto.Message
	expires time.Time
}

// readLoad is one in-flight database read and the callers waiting on it. waiters is guarded
// by readCache.mu; val and err are written before done is closed and read only after it.
type readLoad struct {
	done    chan struct{}
	cancel  context.CancelFunc
	waiters int
	val     proto.Message
	err     error
}

// newReadCache returns nil when ttl or size is not positive, which disables caching: every
// read then goes straight to the database.
func newReadCache(ttl time.Duration, size int, metrics *Metrics) *readCache {
	if ttl <= 0 || size <= 0 {
		return nil
	}

	entries, err := lru.New[string, readCacheEntry](size)
	if err != nil {
		return nil
	}

	return &readCache{
		ttl:     ttl,
		entries: entries,
		metrics: metrics,
		loads:   make(map[string]*readLoad),
	}
}

// cachedRead serves req from the cache, or loads it once for every concurrent caller asking
// the same thing. Callers get their own clone: the cached message is shared and must never be
// handed out for mutation.
func cachedRead[Req, Resp proto.Message](
	ctx context.Context,
	c *readCache,
	req Req,
	load func(context.Context, Req) (Resp, error),
) (Resp, error) {
	if c == nil {
		return load(ctx, req)
	}

	key, err := readCacheKey(req)
	if err != nil {
		return load(ctx, req)
	}

	method := readCacheMethod(req)

	if cached, ok := c.get(key); ok {
		c.observe(method, readCacheHit)

		return cloneAs[Resp](cached)
	}

	l, started := c.join(ctx, key, func(loadCtx context.Context) (proto.Message, error) {
		return load(loadCtx, req)
	})

	if started {
		c.observe(method, readCacheMiss)
	} else {
		c.observe(method, readCacheShared)
	}

	select {
	case <-ctx.Done():
		c.leave(key, l)

		var zero Resp

		return zero, ctx.Err()
	case <-l.done:
	}

	if l.err != nil {
		var zero Resp

		return zero, l.err
	}

	return cloneAs[Resp](l.val)
}

func (c *readCache) get(key string) (proto.Message, bool) {
	entry, ok := c.entries.Get(key)
	if !ok || time.Now().After(entry.expires) {
		// An expired entry is left for the next load to overwrite: removing it here could
		// race that load and drop the fresh one.
		return nil, false
	}

	return entry.resp, true
}

// join adds the caller to the load in flight for key, starting one if there is none, and
// reports whether it started it.
func (c *readCache) join(ctx context.Context, key string, load func(context.Context) (proto.Message, error)) (*readLoad, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	l, ok := c.loads[key]
	if !ok {
		// The load belongs to every caller that joins it, not just this one, so it ignores
		// this caller's cancellation; leave() cancels it once the last caller has gone.
		loadCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))

		l = &readLoad{done: make(chan struct{}), cancel: cancel}
		c.loads[key] = l

		go c.run(loadCtx, key, l, load)
	}

	l.waiters++

	return l, !ok
}

func (c *readCache) run(ctx context.Context, key string, l *readLoad, load func(context.Context) (proto.Message, error)) {
	defer l.cancel()

	val, err := load(ctx)
	if err == nil {
		// Cached before the load is retired, so a caller arriving in between finds one or
		// the other.
		c.entries.Add(key, readCacheEntry{resp: val, expires: time.Now().Add(c.ttl)})
	}

	c.mu.Lock()
	if c.loads[key] == l {
		delete(c.loads, key)
	}
	c.mu.Unlock()

	l.val, l.err = val, err
	close(l.done)
}

// leave drops a caller that stopped waiting, and cancels the load when it was the last one.
func (c *readCache) leave(key string, l *readLoad) {
	c.mu.Lock()
	defer c.mu.Unlock()

	l.waiters--
	if l.waiters > 0 {
		return
	}

	l.cancel()

	// A caller arriving after this starts a fresh load instead of joining a cancelled one.
	if c.loads[key] == l {
		delete(c.loads, key)
	}
}

func (c *readCache) observe(method, result string) {
	if c.metrics == nil {
		return
	}

	c.metrics.readCache.WithLabelValues(method, result).Inc()
}

// readCacheKey identifies a request by its message type and a digest of its deterministic
// wire encoding, so any field that can change the answer changes the key. Every cached method
// has its own request type, so two methods never share a key.
//
// A ListUnique request's fields are sorted and deduplicated first. The response has one slot
// per field and the store skips repeats, so neither order nor repeats change the answer, and
// left in they would let one question fill the cache under many keys.
func readCacheKey(req proto.Message) (string, error) {
	msg := req.ProtoReflect()

	if fd := msg.Descriptor().Fields().ByName("fields"); fd != nil && fd.IsList() && fd.Kind() == protoreflect.EnumKind {
		msg = proto.Clone(req).ProtoReflect()
		list := msg.Mutable(fd).List()

		values := make([]protoreflect.EnumNumber, 0, list.Len())
		for idx := range list.Len() {
			values = append(values, list.Get(idx).Enum())
		}

		slices.Sort(values)
		values = slices.Compact(values)

		list.Truncate(0)

		for _, v := range values {
			list.Append(protoreflect.ValueOfEnum(v))
		}
	}

	raw, err := proto.MarshalOptions{Deterministic: true}.Marshal(msg.Interface())
	if err != nil {
		return "", err
	}

	sum := sha256.Sum256(raw)

	return string(msg.Descriptor().FullName()) + "\x00" + string(sum[:]), nil
}

// readCacheMethod names the handler for the metrics. Each request type is named after its
// method.
func readCacheMethod(req proto.Message) string {
	return strings.TrimSuffix(string(req.ProtoReflect().Descriptor().Name()), "Request")
}

func cloneAs[Resp proto.Message](msg proto.Message) (Resp, error) {
	resp, ok := proto.Clone(msg).(Resp)
	if !ok {
		var zero Resp

		// Unreachable while each request type is served by one handler: the key carries the
		// request type, so a load only ever produces that handler's response type.
		return zero, fmt.Errorf("read cache holds %T, not %T", msg, zero)
	}

	return resp, nil
}
