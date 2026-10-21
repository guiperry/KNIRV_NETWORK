// Package livesearch is the Factuality Slice's live evidence path (plan §5.3): official
// Wikipedia and arXiv APIs, a retriever that treats the evidence index as a cache with a
// 30-day recheck, query minimization, and the guarded HTTP client every outward call
// goes through. It replaces a corpus download: KNIRVSERVER fetches what a claim needs,
// pins the revision it read, and keeps the text so the verdict can be re-checked.
package livesearch

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
	"syscall"
	"time"
)

// Errors a caller can tell apart: a degraded result needs to say why.
var (
	ErrHostNotAllowed = errors.New("livesearch: host is not on the allow-list")
	ErrBreakerOpen    = errors.New("livesearch: provider circuit breaker is open")
	ErrTooLarge       = errors.New("livesearch: response exceeds the size limit")
)

// Policy bounds every outward request.
type Policy struct {
	// Hosts are the only hosts that may be contacted, by exact name. Redirects are
	// followed only to another allowed host.
	Hosts []string
	// AllowInsecure permits http (tests against a loopback server). Production is https only.
	AllowInsecure bool
	// AllowPrivate permits private, loopback and link-local addresses (tests only).
	AllowPrivate bool
	UserAgent    string
	Timeout      time.Duration
	MaxBytes     int64
	// MinInterval per host between request starts (the rate limit).
	MinInterval map[string]time.Duration
	// Breaker: this many consecutive failures open it for Cooldown.
	BreakerFailures int
	Cooldown        time.Duration
	Now             func() time.Time
}

func (p *Policy) now() time.Time {
	if p.Now != nil {
		return p.Now()
	}
	return time.Now()
}

// Client is a guarded HTTP client.
type Client struct {
	pol  Policy
	http *http.Client

	mu       sync.Mutex
	next     map[string]time.Time // earliest start time per host
	failures map[string]int
	openTill map[string]time.Time
}

// NewClient builds the client. The dialer refuses private addresses at connect time,
// after DNS, so a hostname that resolves to an internal address cannot be used to reach
// the local network (the SSRF case).
func NewClient(p Policy) *Client {
	if p.Timeout <= 0 {
		p.Timeout = 20 * time.Second
	}
	if p.MaxBytes <= 0 {
		p.MaxBytes = 8 << 20
	}
	if p.BreakerFailures <= 0 {
		p.BreakerFailures = 4
	}
	if p.Cooldown <= 0 {
		p.Cooldown = 2 * time.Minute
	}
	if p.UserAgent == "" {
		p.UserAgent = "KNIRV-Factuality/1.0 (+https://knirv.com; evidence lookup)"
	}
	c := &Client{pol: p, next: map[string]time.Time{}, failures: map[string]int{}, openTill: map[string]time.Time{}}
	d := &net.Dialer{Timeout: 10 * time.Second}
	if !p.AllowPrivate {
		d.Control = func(_, address string, _ syscall.RawConn) error {
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				return err
			}
			if ip := net.ParseIP(host); ip == nil || isPrivate(ip) {
				return fmt.Errorf("livesearch: refusing to connect to non-public address %s", host)
			}
			return nil
		}
	}
	c.http = &http.Client{
		Timeout:   p.Timeout,
		Transport: &http.Transport{DialContext: d.DialContext, TLSHandshakeTimeout: 10 * time.Second, ResponseHeaderTimeout: p.Timeout},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 4 {
				return errors.New("livesearch: too many redirects")
			}
			return c.check(req.URL)
		},
	}
	return c
}

func isPrivate(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsUnspecified() || ip.IsMulticast() || ip.Equal(net.IPv4bcast)
}

// check enforces scheme and host before any connection is made.
func (c *Client) check(u *url.URL) error {
	if u.Scheme != "https" && !(u.Scheme == "http" && c.pol.AllowInsecure) {
		return fmt.Errorf("%w: scheme %q", ErrHostNotAllowed, u.Scheme)
	}
	if u.User != nil {
		return fmt.Errorf("%w: credentials in URL", ErrHostNotAllowed)
	}
	host := u.Host
	for _, h := range c.pol.Hosts {
		if host == h {
			return nil
		}
	}
	return fmt.Errorf("%w: %s", ErrHostNotAllowed, host)
}

// Get fetches a URL under the policy and returns the (size-capped) body.
func (c *Client) Get(ctx context.Context, rawURL string) ([]byte, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	if err := c.check(u); err != nil {
		return nil, err
	}
	if err := c.admit(ctx, u.Host); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.pol.UserAgent)
	req.Header.Set("Accept", "application/json, application/atom+xml, text/plain;q=0.5")
	res, err := c.http.Do(req)
	if err != nil {
		c.record(u.Host, false)
		return nil, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, c.pol.MaxBytes+1))
	if err != nil {
		c.record(u.Host, false)
		return nil, err
	}
	if int64(len(body)) > c.pol.MaxBytes {
		return nil, ErrTooLarge
	}
	if res.StatusCode == http.StatusTooManyRequests || res.StatusCode >= 500 {
		c.record(u.Host, false)
		return nil, fmt.Errorf("livesearch: %s answered HTTP %d", u.Host, res.StatusCode)
	}
	if res.StatusCode != http.StatusOK {
		// A 4xx is the request's fault, not the provider's health.
		c.record(u.Host, true)
		return nil, fmt.Errorf("livesearch: %s answered HTTP %d", u.Host, res.StatusCode)
	}
	c.record(u.Host, true)
	return body, nil
}

// admit applies the circuit breaker and the per-host rate limit, waiting if needed.
func (c *Client) admit(ctx context.Context, host string) error {
	c.mu.Lock()
	if t := c.openTill[host]; c.pol.now().Before(t) {
		c.mu.Unlock()
		return fmt.Errorf("%w: %s until %s", ErrBreakerOpen, host, t.Format(time.RFC3339))
	}
	wait := time.Duration(0)
	if iv := c.pol.MinInterval[host]; iv > 0 {
		now := c.pol.now()
		start := c.next[host]
		if start.Before(now) {
			start = now
		}
		wait = start.Sub(now)
		c.next[host] = start.Add(iv)
	}
	c.mu.Unlock()
	if wait > 0 {
		select {
		case <-time.After(wait):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

func (c *Client) record(host string, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if ok {
		c.failures[host] = 0
		return
	}
	c.failures[host]++
	if c.failures[host] >= c.pol.BreakerFailures {
		c.openTill[host] = c.pol.now().Add(c.pol.Cooldown)
		c.failures[host] = 0
	}
}
