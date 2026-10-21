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

// Enabled reads the deployment switch. Live lookup is on unless KNIRV_FLASH is 0/false/off;
// a claim's minimized search terms are the only thing that leaves the machine.
func Enabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("KNIRV_FLASH"))) {
	case "0", "false", "off", "no":
		return false
	}
	return true
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
	return NewRetriever(ix, indexDir, []Provider{NewWikipedia(c), NewArxiv(c)}, Config{Enabled: Enabled()})
}
