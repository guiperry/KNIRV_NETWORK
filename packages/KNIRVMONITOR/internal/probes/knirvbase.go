package probes

import "time"

type KNIRVBaseProbe struct {
	url string
}

func NewKNIRVBaseProbe(url string) *KNIRVBaseProbe {
	return &KNIRVBaseProbe{url: url}
}

func (p *KNIRVBaseProbe) Name() string {
	return "knirvbase"
}

func (p *KNIRVBaseProbe) Scrape() (*ProbeResult, error) {
	return scrapeURL(p.url)
}

type KNIRVBaseHealth struct {
	URL              string    `json:"url"`
	Status           string    `json:"status"`
	BlocksCommitted  int64     `json:"blocks_committed"`
	ErrorRate        float64   `json:"error_rate"`
	CacheHitRatio    float64   `json:"cache_hit_ratio"`
	ActiveConnections int64    `json:"active_connections"`
	LastCheck        time.Time `json:"last_check"`
}
