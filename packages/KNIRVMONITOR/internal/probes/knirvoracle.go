package probes

import "time"

type KNIRVOracleProbe struct {
	url string
}

func NewKNIRVOracleProbe(url string) *KNIRVOracleProbe {
	return &KNIRVOracleProbe{url: url}
}

func (p *KNIRVOracleProbe) Name() string {
	return "knirvoracle"
}

func (p *KNIRVOracleProbe) Scrape() (*ProbeResult, error) {
	return scrapeURL(p.url)
}

type KNIRVOracleHealth struct {
	URL       string    `json:"url"`
	Status    string    `json:"status"`
	LastCheck time.Time `json:"last_check"`
}
