package teacher

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// entry is one cached answer.
type entry struct {
	Key  string     `json:"key"`
	Text string     `json:"text"`
	Tok  int        `json:"tokens"`
	Prov Provenance `json:"prov"`
}

// Cache is an append-only JSONL file of answers keyed by model, request and sample,
// so a rerun is a no-op and a killed job resumes where it stopped. A nil *Cache
// caches nothing.
type Cache struct {
	mu   sync.Mutex
	f    *os.File
	byID map[string]entry
}

// OpenCache loads (or creates) the cache file.
func OpenCache(path string) (*Cache, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	c := &Cache{byID: map[string]entry{}}
	if f, err := os.Open(path); err == nil {
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 0, 1<<20), 64<<20)
		for sc.Scan() {
			var e entry
			if json.Unmarshal(sc.Bytes(), &e) == nil && e.Key != "" {
				c.byID[e.Key] = e
			}
		}
		f.Close()
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	c.f = f
	return c, nil
}

// Close flushes and closes the file.
func (c *Cache) Close() error {
	if c == nil || c.f == nil {
		return nil
	}
	return c.f.Close()
}

// Len is the number of cached answers.
func (c *Cache) Len() int {
	if c == nil {
		return 0
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.byID)
}

// Sample returns sample n of the request: from the cache when present, else from the
// teacher (and then cached). cached tells which.
func (c *Cache) Sample(ctx context.Context, cl *Client, r Request, n int) (rep Reply, cached bool, err error) {
	model := cl.catalogKey(ctx)
	key := sampleKey(model, r, n)
	if c != nil {
		c.mu.Lock()
		e, ok := c.byID[key]
		c.mu.Unlock()
		if ok {
			return Reply{Text: e.Text, CompletionTokens: e.Tok, Prov: e.Prov}, true, nil
		}
	}
	rep, err = cl.Chat(ctx, r)
	if err != nil {
		return Reply{}, false, err
	}
	if c != nil {
		e := entry{Key: key, Text: rep.Text, Tok: rep.CompletionTokens, Prov: rep.Prov}
		b, _ := json.Marshal(e)
		c.mu.Lock()
		c.byID[key] = e
		_, werr := c.f.Write(append(b, '\n'))
		c.mu.Unlock()
		if werr != nil {
			return rep, false, fmt.Errorf("write teacher cache: %w", werr)
		}
	}
	return rep, false, nil
}

// catalogKey is the model identity a cache key is built from: the catalog model when
// known, else a placeholder (still stable for one deployment).
func (c *Client) catalogKey(ctx context.Context) string {
	if c.catalog == "" {
		if st, err := c.Status(ctx); err == nil {
			c.catalog = catalogOf(st)
		}
	}
	return c.catalog
}
