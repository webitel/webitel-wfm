package cache

import (
	"context"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/VictoriaMetrics/fastcache"
	"go.opentelemetry.io/otel/trace"
)

// TODO: Inmemory distributed cache.
//  	 Invalidations and sync via rabbitmq or http.
//  	 Enable when service replica count > 1.
//  	 See: golang/groupcache etc..

// TODO: Metrics
// TODO: Tracing
// TODO: Logging

type Mode int

// Cache modes.
const (
	Split Mode = iota
	Switching
	Whole
)

type Manager interface {
	Get(ctx context.Context, k []byte) []byte
	Set(ctx context.Context, k []byte, v []byte)
	Remove(ctx context.Context, k []byte) error
	Has(k []byte) bool

	Stop()
	Reset()
	UpdateStats(fcs *fastcache.Stats)
}

// Cache is a cache for working set entries.
//
// The cache evicts inactive entries after the given expireDuration.
// Recently accessed entries survive expireDuration.
type Cache struct {
	curr atomic.Pointer[fastcache.Cache]
	prev atomic.Pointer[fastcache.Cache]

	// csHistory holds cache stats history
	csHistory fastcache.Stats

	// mode indicates whether to use only curr and skip prev.
	//
	// This flag is set to switching if curr is filled for more than 50% space.
	// In this case using prev would result in RAM waste,
	// it is better to use only curr cache with doubled size.
	// After the process of switching, this flag will be set to whole.
	mode atomic.Uint32

	// maxBytes must be smaller than the available RAM size for the app, since the cache holds data in memory.
	// If maxBytes is less than 32MB, then the minimum cache capacity is 32MB.
	maxBytes int

	// Items in the previous caches are removed when the percent of requests it serves
	// becomes lower than prevCacheRemovalPercent value.
	// Higher values reduce memory usage at the cost of higher CPU usage.
	prevCacheRemovalPercent float64

	// Items are removed from in-memory caches after they aren't accessed for cacheExpireDuration.
	// Lower values may reduce memory usage at the cost of higher CPU usage.
	cacheExpireDuration time.Duration

	// mu serializes access to curr, prev and mode
	// in expirationWatcher, prevCacheWatcher and cacheSizeWatcher.
	mu sync.Mutex

	wg     sync.WaitGroup
	stopCh chan struct{}

	tracer *Tracer
}

// New creates new cache with the given maxBytes capacity in bytes and *cacheExpireDuration expiration.
//
// Stop must be called on the returned cache when it is no longer needed.
func New(maxBytes int) (*Cache, error) {
	curr := fastcache.New(maxBytes / 2)
	prev := fastcache.New(1024)

	c := &Cache{
		maxBytes: maxBytes,
		stopCh:   make(chan struct{}),
		tracer:   NewTracer(),
	}

	c.curr.Store(curr)
	c.prev.Store(prev)
	c.mode.Store(uint32(Split))

	return c, nil
}

func (c *Cache) Get(ctx context.Context, k []byte) []byte {
	var err error
	if c.tracer != nil {
		ctx = c.tracer.Start(ctx, "cache.Get", string(k))
		defer c.tracer.End(ctx, err)
	}

	curr := c.curr.Load()
	result := curr.Get(nil, k)
	if len(result) > 0 {
		trace.SpanFromContext(ctx).AddEvent("fast path - the entry is found in the current cache")
		return result
	}

	if c.mode.Load() == uint32(Whole) {
		trace.SpanFromContext(ctx).AddEvent("nothing found in the current cache")
		return result
	}

	// Search for the entry in the previous cache.
	prev := c.prev.Load()
	result = prev.Get(nil, k)
	if len(result) == 0 {
		trace.SpanFromContext(ctx).AddEvent("nothing found in the previous cache")
		return result
	}

	// Cache the found entry in the current cache.
	curr.Set(k, result)
	trace.SpanFromContext(ctx).AddEvent("cache the found entry in the current cache")

	return result
}

// Has verifies whether the cache contains the given key.
func (c *Cache) Has(k []byte) bool {
	curr := c.curr.Load()
	if curr.Has(k) {
		return true
	}

	if c.mode.Load() == uint32(Whole) {
		return false
	}

	prev := c.prev.Load()
	if !prev.Has(k) {
		return false
	}

	// Cache the found entry in the current cache.
	b := prev.Get(nil, k)
	curr.Set(k, b)

	return true
}

// Set sets the given value for the given key.
func (c *Cache) Set(ctx context.Context, k []byte, v []byte) {
	var err error
	if c.tracer != nil {
		ctx = c.tracer.Start(ctx, "cache.Set", string(k))
		defer c.tracer.End(ctx, err)
	}

	curr := c.curr.Load()
	curr.Set(k, v)
}

func (c *Cache) Remove(ctx context.Context, k []byte) error {
	var err error
	if c.tracer != nil {
		ctx = c.tracer.Start(ctx, "cache.Remove", string(k))
		defer c.tracer.End(ctx, err)
	}

	curr := c.curr.Load()
	curr.Del(k)

	prev := c.prev.Load()
	if prev.Has(k) {
		prev.Del(k)
	}

	return nil
}

// Stop stops the cache.
//
// The cache cannot be used after the Stop call.
func (c *Cache) Stop() {
	close(c.stopCh)
	c.wg.Wait()

	c.Reset()
}

// Reset resets the cache.
func (c *Cache) Reset() {
	var cs fastcache.Stats
	prev := c.prev.Load()
	prev.UpdateStats(&cs)
	prev.Reset()
	curr := c.curr.Load()
	curr.UpdateStats(&cs)
	updateCacheStatsHistory(&c.csHistory, &cs)
	curr.Reset()
	// Reset the mode to `split` in the hope the working set size becomes smaller after the reset.
	c.mode.Store(uint32(Split))
}

// UpdateStats updates fcs with cache stats.
func (c *Cache) UpdateStats(fcs *fastcache.Stats) {
	updateCacheStatsHistory(fcs, &c.csHistory)

	var cs fastcache.Stats
	curr := c.curr.Load()
	curr.UpdateStats(&cs)
	updateCacheStats(fcs, &cs)

	prev := c.prev.Load()
	cs.Reset()
	prev.UpdateStats(&cs)
	updateCacheStats(fcs, &cs)
}

func (c *Cache) expirationWatcher(expireDuration time.Duration) {
	expireDuration = addJitterToDuration(expireDuration)
	t := time.NewTicker(expireDuration)
	defer t.Stop()
	for {
		select {
		case <-c.stopCh:
			return
		case <-t.C:
		}

		c.mu.Lock()
		if c.mode.Load() != uint32(Split) {
			// Stop the expirationWatcher on non-split mode.
			c.mu.Unlock()
			return
		}

		// Reset prev cache and swap it with the curr cache.
		prev := c.prev.Load()
		curr := c.curr.Load()
		c.prev.Store(curr)
		var cs fastcache.Stats
		prev.UpdateStats(&cs)
		updateCacheStatsHistory(&c.csHistory, &cs)
		prev.Reset()
		c.curr.Store(prev)
		c.mu.Unlock()
	}
}

func (c *Cache) prevCacheWatcher() {
	p := c.prevCacheRemovalPercent / 100
	if p <= 0 {
		// There is no need in removing the previous cache.
		return
	}

	minCurrRequests := uint64(1 / p)

	// Watch for the usage of the prev cache and drop it whenever it receives
	// less than prevCacheRemovalPercent requests comparing to the curr cache during the last 60 seconds.
	checkInterval := addJitterToDuration(time.Second * 60)
	t := time.NewTicker(checkInterval)
	defer t.Stop()
	prevGetCalls := uint64(0)
	currGetCalls := uint64(0)
	for {
		select {
		case <-c.stopCh:
			return
		case <-t.C:
		}

		c.mu.Lock()
		if c.mode.Load() != uint32(Split) {
			// Do nothing in non-split mode.
			c.mu.Unlock()

			return
		}

		prev := c.prev.Load()
		curr := c.curr.Load()

		var csCurr, csPrev fastcache.Stats
		curr.UpdateStats(&csCurr)
		prev.UpdateStats(&csPrev)

		currRequests := csCurr.GetCalls
		if currRequests >= currGetCalls {
			currRequests -= currGetCalls
		}

		prevRequests := csPrev.GetCalls
		if prevRequests >= prevGetCalls {
			prevRequests -= prevGetCalls
		}

		currGetCalls = csCurr.GetCalls
		prevGetCalls = csPrev.GetCalls
		if currRequests >= minCurrRequests && float64(prevRequests)/float64(currRequests) < p {
			// The majority of requests are served from the curr cache,
			// so the prev cache can be deleted in order to free up memory.
			if csPrev.EntriesCount > 0 {
				updateCacheStatsHistory(&c.csHistory, &csPrev)
				prev.Reset()
			}
		}

		c.mu.Unlock()
	}
}

func (c *Cache) cacheSizeWatcher() {
	checkInterval := addJitterToDuration(time.Millisecond * 1500)
	t := time.NewTicker(checkInterval)
	defer t.Stop()

	var maxBytesSize uint64
	for {
		select {
		case <-c.stopCh:
			return
		case <-t.C:
		}

		if c.mode.Load() != uint32(Split) {
			continue
		}

		var cs fastcache.Stats
		curr := c.curr.Load()
		curr.UpdateStats(&cs)
		if cs.BytesSize >= uint64(0.9*float64(cs.MaxBytesSize)) {
			maxBytesSize = cs.MaxBytesSize

			break
		}
	}

	// curr cache size exceeds 90% of its capacity. It is better
	// to double the size of curr cache and stop using prev cache,
	// since this will result in higher summary cache capacity.
	//
	// Do this in the following steps:
	// 1) switch to mode=switching
	// 2) move curr cache to prev
	// 3) create curr cache with doubled size
	// 4) wait until curr cache size exceeds maxBytesSize, i.e. it is populated with new data
	// 5) switch to mode=whole
	// 6) drop prev cache

	c.mu.Lock()
	c.mode.Store(uint32(Switching))
	prev := c.prev.Load()
	curr := c.curr.Load()
	c.prev.Store(curr)

	var cs fastcache.Stats
	prev.UpdateStats(&cs)
	updateCacheStatsHistory(&c.csHistory, &cs)
	prev.Reset()

	// use c.maxBytes instead of maxBytesSize*2 for creating new cache, since otherwise the created cache
	// couldn't be loaded from file with c.maxBytes limit after saving with maxBytesSize*2 limit.
	c.curr.Store(fastcache.New(c.maxBytes))
	c.mu.Unlock()

	for {
		select {
		case <-c.stopCh:
			return
		case <-t.C:
		}

		var cs fastcache.Stats
		curr := c.curr.Load()
		curr.UpdateStats(&cs)
		if cs.BytesSize >= maxBytesSize {
			break
		}
	}

	c.mu.Lock()
	c.mode.Store(uint32(Whole))
	prev = c.prev.Load()
	c.prev.Store(fastcache.New(1024))
	cs.Reset()
	prev.UpdateStats(&cs)
	updateCacheStatsHistory(&c.csHistory, &cs)
	prev.Reset()
	c.mu.Unlock()
}

func updateCacheStats(dst, src *fastcache.Stats) {
	dst.GetCalls += src.GetCalls
	dst.SetCalls += src.SetCalls
	dst.Misses += src.Misses
	dst.Collisions += src.Collisions
	dst.Corruptions += src.Corruptions
	dst.EntriesCount += src.EntriesCount
	dst.BytesSize += src.BytesSize
	dst.MaxBytesSize += src.MaxBytesSize
}

func updateCacheStatsHistory(dst, src *fastcache.Stats) {
	atomic.AddUint64(&dst.GetCalls, atomic.LoadUint64(&src.GetCalls))
	atomic.AddUint64(&dst.SetCalls, atomic.LoadUint64(&src.SetCalls))
	atomic.AddUint64(&dst.Misses, atomic.LoadUint64(&src.Misses))
	atomic.AddUint64(&dst.Collisions, atomic.LoadUint64(&src.Collisions))
	atomic.AddUint64(&dst.Corruptions, atomic.LoadUint64(&src.Corruptions))

	// Do not add EntriesCount, BytesSize and MaxBytesSize, since these metrics
	// are calculated from c.curr and c.prev caches.
}

// addJitterToDuration adds up to 10% random jitter to d and returns the resulting duration.
//
// The maximum jitter is limited by 10 seconds.
func addJitterToDuration(d time.Duration) time.Duration {
	dv := d / 10
	if dv > 10*time.Second {
		dv = 10 * time.Second
	}

	p := float64(rand.Uint32()) / (1 << 32)

	return d + time.Duration(p*float64(dv))
}
