package factslice

import (
	"container/list"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ClaimCache keeps finished claim results for a while. It is small and in memory: a
// restart empties it, which only costs time.
type ClaimCache struct {
	mu  sync.Mutex
	cap int
	ttl time.Duration
	now func() time.Time
	ll  *list.List
	m   map[string]*list.Element
}

type cacheEntry struct {
	key string
	at  time.Time
	r   ClaimResult
}

// NewClaimCache holds at most capacity results for ttl each.
func NewClaimCache(capacity int, ttl time.Duration) *ClaimCache {
	if capacity <= 0 {
		capacity = 1024
	}
	if ttl <= 0 {
		ttl = time.Hour
	}
	return &ClaimCache{cap: capacity, ttl: ttl, now: time.Now, ll: list.New(), m: map[string]*list.Element{}}
}

// Get returns a copy of the stored result.
func (c *ClaimCache) Get(key string) (ClaimResult, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	el, ok := c.m[key]
	if !ok {
		return ClaimResult{}, false
	}
	en := el.Value.(*cacheEntry)
	if c.now().Sub(en.at) > c.ttl {
		c.ll.Remove(el)
		delete(c.m, key)
		return ClaimResult{}, false
	}
	c.ll.MoveToFront(el)
	return en.r, true
}

// Put stores a result.
func (c *ClaimCache) Put(key string, r ClaimResult) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.m[key]; ok {
		el.Value = &cacheEntry{key, c.now(), r}
		c.ll.MoveToFront(el)
		return
	}
	c.m[key] = c.ll.PushFront(&cacheEntry{key, c.now(), r})
	for c.ll.Len() > c.cap {
		last := c.ll.Back()
		c.ll.Remove(last)
		delete(c.m, last.Value.(*cacheEntry).key)
	}
}

// Len is the number of results held.
func (c *ClaimCache) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ll.Len()
}

// cacheKey is the claim hash plus everything the result depends on: the index
// version, the judge and adapter scope, the thresholds and calibration state, the
// caller's tenant (user-tier evidence is tenant-scoped) and the retrieval options.
func (e *Engine) cacheKey(c Claim, req Request) string {
	th := e.thresholds()
	parts := []string{
		"v1", e.Index.Version(), e.CacheScope, req.Tenant, strconv.Itoa(req.K),
		strconv.FormatBool(e.Cal.IsFitted()), strconv.FormatFloat(th.Accepted, 'g', -1, 64), strconv.FormatFloat(th.Committed, 'g', -1, 64),
		strconv.FormatFloat(th.Attested, 'g', -1, 64), strconv.FormatFloat(th.Uncommitted, 'g', -1, 64),
		strconv.FormatFloat(th.Discouraged, 'g', -1, 64), strconv.FormatFloat(th.Refused, 'g', -1, 64),
		strconv.FormatBool(e.Witness != nil), req.Mode, c.Text, c.Subject, c.Property, c.Value, string(c.Type),
	}
	for _, t := range req.Tiers {
		parts = append(parts, string(t))
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(sum[:])
}
