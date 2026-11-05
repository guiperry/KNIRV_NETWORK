package livesearch

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/net/html"
	"golang.org/x/net/publicsuffix"
)

// EvidenceProvider is what a source of live evidence offers the retriever: search for
// documents by an already-minimized query, fetch one, and (for the recheck) report its
// current revision. Wikipedia, arXiv and the metasearch provider all implement it.
type EvidenceProvider = Provider

// MetasearchName is the Source of pages found through the metasearch engine.
const MetasearchName = "metasearch"

// MetasearchConfig configures the provider for a self-hosted metasearch engine with a
// SearXNG-compatible JSON API: GET {Endpoint}/search?q=...&format=json returns
// {"results":[{"url","title","content","engine"}]}. The engine is a separate process the
// operator runs; this provider only speaks to it.
type MetasearchConfig struct {
	// Endpoint is "http://127.0.0.1:8888" (loopback only) or "unix:///path/to.sock".
	// Anything else is refused: the engine sees every minimized query, so it must be local.
	Endpoint string
	// Engines is the allow-list passed as `engines=` (empty: the engine's defaults).
	Engines []string
	// AllowDomains, when set, keeps only results whose registered domain is listed.
	AllowDomains []string
	// DenyDomains drops results from these registered domains.
	DenyDomains []string
	Timeout     time.Duration
	// Pages fetches result pages; nil builds the default client (any public host, https
	// only, HTML and plain text, 1 MiB, one request per second per host).
	Pages *Client
	Now   func() time.Time
}

// Metasearch is the Provider backed by a local metasearch engine and the SSRF-safe page
// fetcher. Its revision for a page is a hash of the page's extracted text, since a web
// page has no revision id.
type Metasearch struct {
	cfg   MetasearchConfig
	hc    *http.Client
	base  string
	pages *Client
	// allowHTTP lets Search keep http result URLs; only tests set it, against loopback servers.
	allowHTTP bool
}

// ErrMetasearchEndpoint reports an endpoint that is not loopback or a Unix socket.
var ErrMetasearchEndpoint = errors.New("livesearch: the metasearch endpoint must be a loopback address or a unix socket")

// NewWebPageClient is the default guarded client for result pages.
func NewWebPageClient(userAgent string, deny []string) *Client {
	return NewClient(Policy{
		AnyPublicHost: true, DenyHosts: deny, UserAgent: userAgent,
		ContentTypes: []string{"text/html", "application/xhtml+xml", "text/plain"},
		Accept:       "text/html, application/xhtml+xml, text/plain;q=0.8",
		Timeout:      15 * time.Second, MaxBytes: 1 << 20,
		BreakerFailures: 4, Cooldown: 5 * time.Minute,
	})
}

// NewMetasearch validates the endpoint and builds the provider.
func NewMetasearch(cfg MetasearchConfig) (*Metasearch, error) {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 15 * time.Second
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	m := &Metasearch{cfg: cfg}
	u, err := url.Parse(strings.TrimSpace(cfg.Endpoint))
	if err != nil || cfg.Endpoint == "" {
		return nil, fmt.Errorf("%w: %q", ErrMetasearchEndpoint, cfg.Endpoint)
	}
	switch u.Scheme {
	case "unix":
		sock := u.Path
		if sock == "" {
			return nil, fmt.Errorf("%w: %q", ErrMetasearchEndpoint, cfg.Endpoint)
		}
		m.base = "http://metasearch"
		m.hc = &http.Client{Timeout: cfg.Timeout, Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return (&net.Dialer{}).DialContext(ctx, "unix", sock)
			}, Proxy: nil}}
	case "http":
		host := u.Hostname()
		if ip := net.ParseIP(host); (ip == nil || !ip.IsLoopback()) && host != "localhost" {
			return nil, fmt.Errorf("%w: %q", ErrMetasearchEndpoint, cfg.Endpoint)
		}
		m.base = strings.TrimRight(u.Scheme+"://"+u.Host, "/")
		m.hc = &http.Client{Timeout: cfg.Timeout, Transport: &http.Transport{Proxy: nil},
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	default:
		return nil, fmt.Errorf("%w: %q", ErrMetasearchEndpoint, cfg.Endpoint)
	}
	m.pages = cfg.Pages
	if m.pages == nil {
		m.pages = NewWebPageClient("", nil)
	}
	return m, nil
}

func (m *Metasearch) Name() string { return MetasearchName }

type searxResult struct {
	URL     string `json:"url"`
	Title   string `json:"title"`
	Content string `json:"content"`
	Engine  string `json:"engine"`
}

// Search asks the engine. query is already minimized; it is the only thing sent.
func (m *Metasearch) Search(ctx context.Context, query string, n int) ([]Ref, error) {
	q := url.Values{"q": {query}, "format": {"json"}, "language": {"en"}, "safesearch": {"1"}, "pageno": {"1"}}
	if len(m.cfg.Engines) > 0 {
		q.Set("engines", strings.Join(m.cfg.Engines, ","))
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, m.base+"/search?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	res, err := m.hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("metasearch: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("metasearch: engine answered HTTP %d", res.StatusCode)
	}
	var body bytes.Buffer
	if _, err := body.ReadFrom(http.MaxBytesReader(nil, res.Body, 4<<20)); err != nil {
		return nil, fmt.Errorf("metasearch: %w", err)
	}
	var out struct {
		Results []searxResult `json:"results"`
	}
	if err := json.Unmarshal(body.Bytes(), &out); err != nil {
		return nil, fmt.Errorf("metasearch: malformed answer: %w", err)
	}
	var refs []Ref
	seen := map[string]bool{}
	for _, r := range out.Results {
		u, err := url.Parse(strings.TrimSpace(r.URL))
		// The engine is trusted to find pages, not to decide where we connect: every
		// result is vetted as if it came from anywhere.
		if err != nil || (u.Scheme != "https" && !(m.allowHTTP && u.Scheme == "http")) || u.User != nil || u.Host == "" {
			continue
		}
		dom := RegisteredDomain(u.Hostname())
		if m.allowHTTP && dom == "" {
			dom = u.Hostname()
		}
		if dom == "" || !domainAllowed(dom, m.cfg.AllowDomains, m.cfg.DenyDomains) {
			continue
		}
		u.Fragment = ""
		key := u.String()
		if seen[key] {
			continue
		}
		seen[key] = true
		title := strings.TrimSpace(r.Title)
		if title == "" {
			title = u.Host
		}
		refs = append(refs, Ref{Source: MetasearchName, ID: key, Title: title, URL: key})
		if len(refs) >= n {
			break
		}
	}
	return refs, nil
}

func domainAllowed(dom string, allow, deny []string) bool {
	for _, d := range deny {
		if strings.EqualFold(d, dom) {
			return false
		}
	}
	if len(allow) == 0 {
		return true
	}
	for _, a := range allow {
		if strings.EqualFold(a, dom) {
			return true
		}
	}
	return false
}

// Fetch reads one result page through the guarded client and extracts its text.
func (m *Metasearch) Fetch(ctx context.Context, r Ref) (Page, error) {
	body, err := m.pages.Get(ctx, r.URL)
	if err != nil {
		return Page{}, err
	}
	title, text := ExtractText(body)
	if strings.TrimSpace(text) == "" {
		return Page{}, fmt.Errorf("metasearch: %s has no readable text", r.URL)
	}
	if title == "" {
		title = r.Title
	}
	pg := Page{Ref: Ref{Source: MetasearchName, ID: r.ID, Title: title, URL: r.URL}, Text: text,
		License: "unknown", FetchedAt: m.cfg.Now().UTC().Format(time.RFC3339)}
	if len(pg.Text) > maxPageChars {
		pg.Text, pg.Truncated = pg.Text[:maxPageChars], true
	}
	pg.Revision = ContentRevision(pg.Text)
	return pg, nil
}

// Revisions refetches each page and reports the hash of its text; a page that cannot be
// read is absent from the map, which the retriever reports as "not found on recheck".
func (m *Metasearch) Revisions(ctx context.Context, refs []Ref) (map[string]string, error) {
	out := map[string]string{}
	var firstErr error
	for _, r := range refs {
		pg, err := m.Fetch(ctx, r)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		out[r.ID] = pg.Revision
	}
	if len(out) == 0 && firstErr != nil {
		return nil, firstErr
	}
	return out, nil
}

// ContentRevision is the revision id of a web page: a short hash of its normalized text.
func ContentRevision(text string) string {
	sum := sha256.Sum256([]byte(strings.Join(strings.Fields(text), " ")))
	return "sha256:" + hex.EncodeToString(sum[:8])
}

// RegisteredDomain is the effective TLD plus one label ("news.bbc.co.uk" is "bbc.co.uk"),
// the unit of independence for the corroboration rule. It is "" for an IP address, a
// single-label name or a name the public-suffix list does not cover.
func RegisteredDomain(host string) string {
	host = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(host), "."))
	if host == "" || net.ParseIP(host) != nil {
		return ""
	}
	d, err := publicsuffix.EffectiveTLDPlusOne(host)
	if err != nil {
		return ""
	}
	return d
}

// IndependentDomains counts the distinct registered domains among urls.
func IndependentDomains(urls []string) int {
	seen := map[string]bool{}
	for _, raw := range urls {
		if u, err := url.Parse(raw); err == nil {
			if d := RegisteredDomain(u.Hostname()); d != "" {
				seen[d] = true
			}
		}
	}
	return len(seen)
}

// ---- readable text from HTML ----

var skipElements = map[string]bool{"script": true, "style": true, "noscript": true, "template": true, "nav": true,
	"header": true, "footer": true, "aside": true, "form": true, "iframe": true, "svg": true, "canvas": true,
	"button": true, "select": true, "option": true, "menu": true, "dialog": true}

var blockElements = map[string]bool{"p": true, "div": true, "li": true, "br": true, "tr": true, "h1": true, "h2": true,
	"h3": true, "h4": true, "h5": true, "h6": true, "section": true, "article": true, "main": true, "blockquote": true,
	"pre": true, "table": true, "ul": true, "ol": true, "dd": true, "dt": true, "figcaption": true}

// ExtractText returns the page title and its readable text: script, style, navigation,
// footers, forms and similar are dropped; text inside <main> or <article> is preferred
// when the page has one. The result is plain text, never markup, and short runs of
// boilerplate (a few words) are dropped so a menu is not mistaken for content.
func ExtractText(doc []byte) (title, text string) {
	if !utf8.Valid(doc) {
		doc = bytes.ToValidUTF8(doc, []byte("�"))
	}
	root, err := html.Parse(bytes.NewReader(doc))
	if err != nil {
		return "", ""
	}
	var findTitle func(*html.Node)
	findTitle = func(n *html.Node) {
		if title != "" {
			return
		}
		if n.Type == html.ElementNode && n.Data == "title" && n.FirstChild != nil {
			title = strings.Join(strings.Fields(n.FirstChild.Data), " ")
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findTitle(c)
		}
	}
	findTitle(root)

	scope := root
	var findMain func(*html.Node) *html.Node
	findMain = func(n *html.Node) *html.Node {
		if n.Type == html.ElementNode && (n.Data == "main" || n.Data == "article") {
			return n
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if f := findMain(c); f != nil {
				return f
			}
		}
		return nil
	}
	if m := findMain(root); m != nil {
		scope = m
	}
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && skipElements[n.Data] {
			return
		}
		if n.Type == html.ElementNode && n.Data == "head" {
			return
		}
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
			b.WriteByte(' ')
		}
		if n.Type == html.ElementNode && blockElements[n.Data] {
			b.WriteByte('\n')
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
		if n.Type == html.ElementNode && blockElements[n.Data] {
			b.WriteByte('\n')
		}
	}
	walk(scope)
	var lines []string
	for _, ln := range strings.Split(b.String(), "\n") {
		ln = strings.Join(strings.Fields(ln), " ")
		if len(strings.Fields(ln)) >= 4 {
			lines = append(lines, ln)
		}
	}
	text = strings.Join(lines, "\n")
	if text == "" && scope == root {
		// A plain-text document parses as one text node.
		text = strings.Join(strings.Fields(b.String()), " ")
	}
	return title, text
}
