package livesearch

import (
	"os"
	"strings"
	"time"

	"knirvmodels/pkg/evidence"
)

// Production hosts. These two names are the whole outward surface of the live path.
const (
	WikipediaHost = "en.wikipedia.org"
	ArxivHost     = "export.arxiv.org"
)

// Enabled reads the deployment switch. Live lookup ("flash") is ON by default (owner
// decision O5, 2026-11-03) and is turned off with either of:
//
//	KNIRV_FLASH=0          (also false, off, no)
//	KNIRV_NO_FLASH=1       (also true, on, yes)
//
// Off wins if both are set. What leaves the machine when it is on is a claim's minimized
// search words only; KNIRV_METASEARCH_URL adds the self-hosted metasearch provider to the
// built-in Wikipedia and arXiv ones.
func Enabled() bool {
	if truthy(os.Getenv("KNIRV_NO_FLASH")) {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(os.Getenv("KNIRV_FLASH"))) {
	case "0", "false", "off", "no":
		return false
	}
	return true
}

func truthy(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "on", "yes":
		return true
	}
	return false
}

// Metasearch settings from the environment:
//
//	KNIRV_METASEARCH_URL       http://127.0.0.1:PORT or unix:///path (loopback or socket only)
//	KNIRV_METASEARCH_ENGINES   comma-separated engine allow-list passed to the engine
//	KNIRV_METASEARCH_ALLOW     comma-separated registered domains; only these may be read
//	KNIRV_METASEARCH_DENY      comma-separated registered domains that are never read
func metasearchFromEnv() (*Metasearch, error) {
	url := strings.TrimSpace(os.Getenv("KNIRV_METASEARCH_URL"))
	if url == "" {
		return nil, nil
	}
	list := func(k string) []string {
		var out []string
		for _, v := range strings.Split(os.Getenv(k), ",") {
			if v = strings.TrimSpace(v); v != "" {
				out = append(out, v)
			}
		}
		return out
	}
	deny := list("KNIRV_METASEARCH_DENY")
	return NewMetasearch(MetasearchConfig{
		Endpoint: url, Engines: list("KNIRV_METASEARCH_ENGINES"), AllowDomains: list("KNIRV_METASEARCH_ALLOW"), DenyDomains: deny,
		Pages: NewWebPageClient(strings.TrimSpace(os.Getenv("KNIRV_USER_AGENT")), deny),
	})
}

// NewProduction builds a retriever over Wikipedia and arXiv with the production policy:
// https only, the two hosts above, public addresses only, and the rate limits those
// services ask for (Wikimedia tolerates a few requests per second from an identified
// client; arXiv asks for one request every three seconds).
func NewProduction(ix *evidence.Index, indexDir string) (*Retriever, error) {
	ua := strings.TrimSpace(os.Getenv("KNIRV_USER_AGENT"))
	c := NewClient(Policy{
		Hosts:     []string{WikipediaHost, ArxivHost},
		UserAgent: ua,
		MinInterval: map[string]time.Duration{
			WikipediaHost: 250 * time.Millisecond,
			ArxivHost:     3100 * time.Millisecond,
		},
	})
	providers := []Provider{NewWikipedia(c), NewArxiv(c)}
	cfg := Config{Enabled: Enabled()}
	if ms, err := metasearchFromEnv(); err != nil {
		return nil, err
	} else if ms != nil {
		providers = append(providers, ms)
		// A web page has no revision id and changes faster than an encyclopedia entry:
		// provisional, it is rechecked daily rather than every 30 days.
		cfg.TTLBySource = map[string]time.Duration{MetasearchName: 24 * time.Hour}
	}
	return NewRetriever(ix, indexDir, providers, cfg)
}
