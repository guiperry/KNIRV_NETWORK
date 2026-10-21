package livesearch

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Ref points at one fetchable document.
type Ref struct {
	Source string // "wikipedia" or "arxiv"
	ID     string // canonical title (Wikipedia) or arXiv id without version
	Title  string
	URL    string
}

// Page is a fetched document pinned to the revision that was read.
type Page struct {
	Ref
	Revision  string // Wikipedia revid; arXiv version ("v2")
	Text      string
	License   string
	FetchedAt string
	Truncated bool
}

// Provider is one official source.
type Provider interface {
	Name() string
	// Search finds up to n documents for an already-minimized query.
	Search(ctx context.Context, query string, n int) ([]Ref, error)
	// Fetch reads one document and the revision it is at right now.
	Fetch(ctx context.Context, r Ref) (Page, error)
	// Revisions returns the current revision of each ref, keyed by Ref.ID, without
	// fetching text. This is the cheap 30-day recheck.
	Revisions(ctx context.Context, refs []Ref) (map[string]string, error)
}

const maxPageChars = 400_000

// ---- Wikipedia ----

// Wikipedia reads the MediaWiki action API.
type Wikipedia struct {
	API    string // e.g. https://en.wikipedia.org/w/api.php
	PageAt string // e.g. https://en.wikipedia.org/wiki/
	C      *Client
	Now    func() time.Time
}

// NewWikipedia targets en.wikipedia.org.
func NewWikipedia(c *Client) *Wikipedia {
	return &Wikipedia{API: "https://en.wikipedia.org/w/api.php", PageAt: "https://en.wikipedia.org/wiki/", C: c}
}

func (w *Wikipedia) Name() string { return "wikipedia" }

func (w *Wikipedia) now() time.Time {
	if w.Now != nil {
		return w.Now()
	}
	return time.Now()
}

func (w *Wikipedia) pageURL(title string) string {
	return w.PageAt + url.PathEscape(strings.ReplaceAll(title, " ", "_"))
}

func (w *Wikipedia) get(ctx context.Context, q url.Values, into any) error {
	q.Set("format", "json")
	q.Set("formatversion", "2")
	q.Set("origin", "") // not a browser call
	q.Del("origin")
	body, err := w.C.Get(ctx, w.API+"?"+q.Encode())
	if err != nil {
		return err
	}
	var env struct {
		Error *struct{ Code, Info string } `json:"error"`
	}
	if json.Unmarshal(body, &env) == nil && env.Error != nil {
		return fmt.Errorf("wikipedia: %s: %s", env.Error.Code, env.Error.Info)
	}
	return json.Unmarshal(body, into)
}

func (w *Wikipedia) Search(ctx context.Context, query string, n int) ([]Ref, error) {
	var out struct {
		Query struct {
			Search []struct {
				Title string `json:"title"`
			} `json:"search"`
		} `json:"query"`
	}
	err := w.get(ctx, url.Values{"action": {"query"}, "list": {"search"}, "srsearch": {query},
		"srlimit": {fmt.Sprint(n)}, "srnamespace": {"0"}, "srprop": {""}}, &out)
	if err != nil {
		return nil, err
	}
	var refs []Ref
	for _, s := range out.Query.Search {
		refs = append(refs, Ref{Source: "wikipedia", ID: s.Title, Title: s.Title, URL: w.pageURL(s.Title)})
	}
	return refs, nil
}

func (w *Wikipedia) Fetch(ctx context.Context, r Ref) (Page, error) {
	var out struct {
		Query struct {
			Pages []struct {
				Title     string `json:"title"`
				Missing   bool   `json:"missing"`
				Extract   string `json:"extract"`
				FullURL   string `json:"fullurl"`
				Revisions []struct {
					RevID int64 `json:"revid"`
				} `json:"revisions"`
			} `json:"pages"`
		} `json:"query"`
	}
	err := w.get(ctx, url.Values{"action": {"query"}, "prop": {"extracts|revisions|info"}, "explaintext": {"1"},
		"exsectionformat": {"plain"}, "redirects": {"1"}, "rvprop": {"ids"}, "inprop": {"url"}, "titles": {r.ID}}, &out)
	if err != nil {
		return Page{}, err
	}
	if len(out.Query.Pages) == 0 || out.Query.Pages[0].Missing || len(out.Query.Pages[0].Revisions) == 0 {
		return Page{}, fmt.Errorf("wikipedia: no page %q", r.ID)
	}
	p := out.Query.Pages[0]
	text := strings.TrimSpace(p.Extract)
	pg := Page{Ref: Ref{Source: "wikipedia", ID: p.Title, Title: p.Title, URL: w.pageURL(p.Title)},
		Revision: fmt.Sprint(p.Revisions[0].RevID), Text: text, License: "CC-BY-SA-4.0", FetchedAt: w.now().UTC().Format(time.RFC3339)}
	if len(pg.Text) > maxPageChars {
		pg.Text, pg.Truncated = pg.Text[:maxPageChars], true
	}
	if pg.Text == "" {
		return Page{}, fmt.Errorf("wikipedia: %q has no text", r.ID)
	}
	return pg, nil
}

func (w *Wikipedia) Revisions(ctx context.Context, refs []Ref) (map[string]string, error) {
	out := map[string]string{}
	for i := 0; i < len(refs); i += 50 {
		end := i + 50
		if end > len(refs) {
			end = len(refs)
		}
		var titles []string
		for _, r := range refs[i:end] {
			titles = append(titles, r.ID)
		}
		var resp struct {
			Query struct {
				Pages []struct {
					Title     string `json:"title"`
					Missing   bool   `json:"missing"`
					Revisions []struct {
						RevID int64 `json:"revid"`
					} `json:"revisions"`
				} `json:"pages"`
			} `json:"query"`
		}
		if err := w.get(ctx, url.Values{"action": {"query"}, "prop": {"revisions"}, "rvprop": {"ids"}, "titles": {strings.Join(titles, "|")}}, &resp); err != nil {
			return nil, err
		}
		for _, p := range resp.Query.Pages {
			if !p.Missing && len(p.Revisions) > 0 {
				out[p.Title] = fmt.Sprint(p.Revisions[0].RevID)
			}
		}
	}
	return out, nil
}

// ---- arXiv ----

// Arxiv reads the arXiv Atom API. It indexes the title and abstract, which is what the
// API serves; it does not download PDFs.
type Arxiv struct {
	API string // e.g. https://export.arxiv.org/api/query
	C   *Client
	Now func() time.Time

	mu    sync.Mutex
	cache map[string]arxivEntry // entries returned by Search, reused by Fetch
}

// NewArxiv targets export.arxiv.org.
func NewArxiv(c *Client) *Arxiv {
	return &Arxiv{API: "https://export.arxiv.org/api/query", C: c, cache: map[string]arxivEntry{}}
}

func (a *Arxiv) Name() string { return "arxiv" }

type arxivEntry struct {
	ID      string `xml:"id"`
	Title   string `xml:"title"`
	Summary string `xml:"summary"`
	Updated string `xml:"updated"`
	Authors []struct {
		Name string `xml:"name"`
	} `xml:"author"`
}

type arxivFeed struct {
	Entries []arxivEntry `xml:"entry"`
}

// splitID turns "http://arxiv.org/abs/2101.00001v2" into ("2101.00001", "v2").
func splitID(abs string) (id, ver string) {
	s := abs[strings.LastIndex(abs, "/abs/")+len("/abs/"):]
	if i := strings.LastIndex(s, "v"); i > 0 && strings.Trim(s[i+1:], "0123456789") == "" && s[i+1:] != "" {
		return s[:i], s[i:]
	}
	return s, ""
}

func (a *Arxiv) query(ctx context.Context, q url.Values) ([]arxivEntry, error) {
	body, err := a.C.Get(ctx, a.API+"?"+q.Encode())
	if err != nil {
		return nil, err
	}
	var f arxivFeed
	if err := xml.Unmarshal(body, &f); err != nil {
		return nil, fmt.Errorf("arxiv: bad Atom feed: %w", err)
	}
	return f.Entries, nil
}

func clean(s string) string { return strings.Join(strings.Fields(s), " ") }

func (a *Arxiv) Search(ctx context.Context, query string, n int) ([]Ref, error) {
	var terms []string
	for _, w := range strings.Fields(query) {
		terms = append(terms, "all:"+w)
	}
	ents, err := a.query(ctx, url.Values{"search_query": {strings.Join(terms, " AND ")}, "start": {"0"},
		"max_results": {fmt.Sprint(n)}, "sortBy": {"relevance"}})
	if err != nil {
		return nil, err
	}
	var refs []Ref
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, e := range ents {
		id, _ := splitID(e.ID)
		a.cache[id] = e
		refs = append(refs, Ref{Source: "arxiv", ID: id, Title: clean(e.Title), URL: "https://arxiv.org/abs/" + id})
	}
	return refs, nil
}

func (a *Arxiv) now() time.Time {
	if a.Now != nil {
		return a.Now()
	}
	return time.Now()
}

func (a *Arxiv) Fetch(ctx context.Context, r Ref) (Page, error) {
	a.mu.Lock()
	e, ok := a.cache[r.ID]
	a.mu.Unlock()
	if !ok {
		ents, err := a.query(ctx, url.Values{"id_list": {r.ID}})
		if err != nil {
			return Page{}, err
		}
		if len(ents) == 0 {
			return Page{}, fmt.Errorf("arxiv: no paper %q", r.ID)
		}
		e = ents[0]
	}
	id, ver := splitID(e.ID)
	var names []string
	for _, au := range e.Authors {
		names = append(names, clean(au.Name))
	}
	text := clean(e.Title) + "\n\n" + strings.TrimSpace(e.Summary)
	if len(names) > 0 {
		text += "\n\nAuthors: " + strings.Join(names, ", ")
	}
	return Page{Ref: Ref{Source: "arxiv", ID: id, Title: clean(e.Title), URL: "https://arxiv.org/abs/" + id}, Revision: ver, Text: text,
		License: "arXiv metadata (CC0)", FetchedAt: a.now().UTC().Format(time.RFC3339)}, nil
}

func (a *Arxiv) Revisions(ctx context.Context, refs []Ref) (map[string]string, error) {
	out := map[string]string{}
	for i := 0; i < len(refs); i += 50 {
		end := i + 50
		if end > len(refs) {
			end = len(refs)
		}
		var ids []string
		for _, r := range refs[i:end] {
			ids = append(ids, r.ID)
		}
		ents, err := a.query(ctx, url.Values{"id_list": {strings.Join(ids, ",")}, "max_results": {fmt.Sprint(len(ids))}})
		if err != nil {
			return nil, err
		}
		for _, e := range ents {
			id, ver := splitID(e.ID)
			out[id] = ver
		}
	}
	return out, nil
}
