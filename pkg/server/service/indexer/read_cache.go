package indexer

import (
	"context"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
	"golang.org/x/sync/singleflight"
	"google.golang.org/protobuf/proto"
)

const (
	readCacheHit    = "hit"
	readCacheMiss   = "miss"
	readCacheShared = "shared"

	// readCacheLoadTimeout bounds a shared load. It is detached from the caller that started
	// it, so something has to stop it if the database never answers.
	readCacheLoadTimeout = 30 * time.Second
)

// readCache memoizes the responses of the filter-dropdown (ListUnique*Values) and Count*
// handlers for a short TTL. The web UI issues both on every page load and neither needs to
// be fresher than the UI's own 60s staleTime, yet each one is a database walk: the dropdowns
// a query per field, the counts an index scan over every row the filter matches. Uncached,
// a crawler or scraper replaying them turns straight into database CPU.
//
// Concurrent misses on one key share a single load (singleflight), so a burst of identical
// requests costs one query rather than one per request. Only successful responses are kept.
type readCache struct {
	entries *expirable.LRU[string, proto.Message]
	group   singleflight.Group
	metrics *Metrics
}

// newReadCache returns nil when ttl is not positive, which disables caching: every read then
// goes straight to the database.
func newReadCache(ttl time.Duration, size int, metrics *Metrics) *readCache {
	if ttl <= 0 || size <= 0 {
		return nil
	}

	return &readCache{
		entries: expirable.NewLRU[string, proto.Message](size, nil, ttl),
		metrics: metrics,
	}
}

// cachedRead serves req from the cache, or loads it once for every concurrent caller asking
// the same thing. The key is the method plus the deterministic wire encoding of the request,
// so any field that changes the answer changes the key. Callers get their own clone: the
// cached message is shared and must never be handed out for mutation.
func cachedRead[Req, Resp proto.Message](
	ctx context.Context,
	c *readCache,
	method string,
	req Req,
	load func(context.Context, Req) (Resp, error),
) (Resp, error) {
	if c == nil {
		return load(ctx, req)
	}

	raw, err := proto.MarshalOptions{Deterministic: true}.Marshal(req)
	if err != nil {
		return load(ctx, req)
	}

	key := method + "\x00" + string(raw)

	if cached, ok := c.entries.Get(key); ok {
		if resp, ok := proto.Clone(cached).(Resp); ok {
			c.observe(method, readCacheHit)

			return resp, nil
		}
	}

	ch := c.group.DoChan(key, func() (any, error) {
		// The load outlives the caller that started it: other callers may be waiting on it,
		// and its result is cached for the ones after them.
		loadCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), readCacheLoadTimeout)
		defer cancel()

		resp, err := load(loadCtx, req)
		if err != nil {
			return nil, err
		}

		c.entries.Add(key, resp)

		return resp, nil
	})

	var zero Resp

	select {
	case <-ctx.Done():
		return zero, ctx.Err()
	case res := <-ch:
		if res.Shared {
			c.observe(method, readCacheShared)
		} else {
			c.observe(method, readCacheMiss)
		}

		if res.Err != nil {
			return zero, res.Err
		}

		// The key carries the method, so a load only ever produces its own method's type.
		msg, ok := res.Val.(proto.Message)
		if !ok {
			return load(ctx, req)
		}

		resp, ok := proto.Clone(msg).(Resp)
		if !ok {
			return load(ctx, req)
		}

		return resp, nil
	}
}

func (c *readCache) observe(method, result string) {
	if c.metrics == nil {
		return
	}

	c.metrics.readCache.WithLabelValues(method, result).Inc()
}
