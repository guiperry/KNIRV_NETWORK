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
	"strings"
	"sync"
	"time"
)

// Errors a caller can tell apart: a degraded result needs to say why.
var (
	ErrHostNotAllowed = errors.New("livesearch: host is not on the allow-list")
	ErrBreakerOpen    = errors.New("livesearch: provider circuit breaker is open")
	ErrTooLarge       = errors.New("livesearch: response exceeds the size limit")
	ErrContentType    = errors.New("livesearch: content type is not allowed")
)

// Policy bounds every outward request.
type Policy struct {
	// Hosts are the only hosts that may be contacted, by exact name. Redirects are
	// followed only to another allowed host.
	Hosts []string
	// AnyPublicHost lets the client fetch any host on the public internet instead of an
	// allow-list: for pages a metasearch engine pointed at. It does not relax anything
	// else: https only, no credentials in the URL, and a connection is made only to an
	// address that is public after DNS (see Lookup), including after every redirect.
	AnyPublicHost bool
	// DenyHosts are refused even under AnyPublicHost (exact name, or a registered domain
	// such as "example.com" which also covers its subdomains).
	DenyHosts []string
	// ContentTypes, when set, are the only media types a response may have (the part
	// before any ";", lower case, e.g. "text/html"). A page of any other type is refused.
	ContentTypes []string
	// Accept overrides the Accept header.
	Accept string
	// Lookup resolves a host to addresses; nil uses the system resolver. Tests replace it
	// to model DNS rebinding.
	Lookup func(ctx context.Context, host string) ([]net.IP, error)
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
	dial := d.DialContext
	if !p.AllowPrivate {
		// Resolve, vet every address, then connect to a vetted address directly. The name
		// is never resolved a second time by the dialer, so a DNS answer that changes
		// between the check and the connection (rebinding) cannot reach an internal host.
		dial = func(ctx context.Context, network, address string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(address)
			if err != nil {
				return nil, err
			}
			ips, err := c.resolve(ctx, host)
			if err != nil {
				return nil, err
			}
			var last error
			for _, ip := range ips {
				conn, derr := d.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
				if derr == nil {
					return conn, nil
				}
				last = derr
			}
			return nil, last
		}
	}
	c.http = &http.Client{
		Timeout:   p.Timeout,
		Transport: &http.Transport{DialContext: dial, TLSHandshakeTimeout: 10 * time.Second, ResponseHeaderTimeout: p.Timeout, DisableCompression: false, Proxy: nil},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 4 {
				return errors.New("livesearch: too many redirects")
			}
			return c.check(req.URL)
		},
	}
	return c
}

// resolve returns the addresses of host, all of which must be public. An address
// literal is vetted as is; a name that has even one non-public address is refused.
func (c *Client) resolve(ctx context.Context, host string) ([]net.IP, error) {
	if ip := net.ParseIP(host); ip != nil {
		if isPrivate(ip) {
			return nil, fmt.Errorf("livesearch: refusing to connect to non-public address %s", host)
		}
		return []net.IP{ip}, nil
	}
	lookup := c.pol.Lookup
	if lookup == nil {
		lookup = func(ctx context.Context, h string) ([]net.IP, error) {
			addrs, err := net.DefaultResolver.LookupIPAddr(ctx, h)
			ips := make([]net.IP, 0, len(addrs))
			for _, a := range addrs {
				ips = append(ips, a.IP)
			}
			return ips, err
		}
	}
	ips, err := lookup(ctx, host)
	if err != nil {
		return nil, err
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("livesearch: %s has no address", host)
	}
	for _, ip := range ips {
		if isPrivate(ip) {
			return nil, fmt.Errorf("livesearch: refusing to connect to non-public address %s (%s)", ip, host)
		}
	}
	return ips, nil
}

var nonPublicNets = func() []*net.IPNet {
	var out []*net.IPNet
	for _, cidr := range []string{
		"0.0.0.0/8",       // "this" network
		"100.64.0.0/10",   // carrier-grade NAT
		"192.0.0.0/24",    // IETF protocol assignments
		"192.0.2.0/24",    // TEST-NET-1
		"198.18.0.0/15",   // benchmarking
		"198.51.100.0/24", // TEST-NET-2
		"203.0.113.0/24",  // TEST-NET-3
		"240.0.0.0/4",     // reserved
		"64:ff9b::/96",    // NAT64 (can embed a private IPv4)
		"2001:db8::/32",   // documentation
		"fec0::/10",       // deprecated site-local
	} {
		if _, n, err := net.ParseCIDR(cidr); err == nil {
			out = append(out, n)
		}
	}
	return out
}()

// isPrivate is true for any address a public web fetch must never reach: loopback,
// private, link-local (which covers the 169.254.169.254 cloud metadata address),
// unspecified, multicast, carrier-grade NAT, reserved and documentation ranges. An
// IPv4-mapped IPv6 address is judged as the IPv4 address it wraps.
func isPrivate(ip net.IP) bool {
	if v4 := ip.To4(); v4 != nil {
		ip = v4
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsInterfaceLocalMulticast() || ip.IsUnspecified() || ip.IsMulticast() || ip.Equal(net.IPv4bcast) {
		return true
	}
	for _, n := range nonPublicNets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
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
	if c.pol.AnyPublicHost {
		name := strings.ToLower(u.Hostname())
		if name == "" || strings.EqualFold(name, "localhost") || strings.HasSuffix(name, ".localhost") || strings.HasSuffix(name, ".local") || strings.HasSuffix(name, ".internal") {
			return fmt.Errorf("%w: %s", ErrHostNotAllowed, host)
		}
		if ip := net.ParseIP(name); ip != nil && isPrivate(ip) && !c.pol.AllowPrivate {
			return fmt.Errorf("%w: non-public address %s", ErrHostNotAllowed, host)
		}
		for _, d := range c.pol.DenyHosts {
			d = strings.ToLower(d)
			if name == d || strings.HasSuffix(name, "."+d) {
				return fmt.Errorf("%w: %s is denied", ErrHostNotAllowed, host)
			}
		}
		return nil
	}
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
	accept := c.pol.Accept
	if accept == "" {
		accept = "application/json, application/atom+xml, text/plain;q=0.5"
	}
	req.Header.Set("Accept", accept)
	req.Header.Set("Referer", "") // never tell a site where the request came from
	res, err := c.http.Do(req)
	if err != nil {
		c.record(u.Host, false)
		return nil, err
	}
	defer res.Body.Close()
	if len(c.pol.ContentTypes) > 0 {
		ct := strings.ToLower(strings.TrimSpace(strings.SplitN(res.Header.Get("Content-Type"), ";", 2)[0]))
		ok := false
		for _, allowed := range c.pol.ContentTypes {
			ok = ok || ct == allowed
		}
		if !ok {
			c.record(u.Host, true) // the server is healthy; the document is not one we read
			return nil, fmt.Errorf("%w: %q", ErrContentType, ct)
		}
	}
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
