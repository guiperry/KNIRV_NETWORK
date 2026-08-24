package probes

import "time"

type KNIRVGraphProbe struct {
	url string
}

func NewKNIRVGraphProbe(url string) *KNIRVGraphProbe {
	return &KNIRVGraphProbe{url: url}
}

func (p *KNIRVGraphProbe) Name() string {
	return "knirvgraph"
}

func (p *KNIRVGraphProbe) Scrape() (*ProbeResult, error) {
	return scrapeURL(p.url)
}

type KNIRVGraphHealth struct {
	URL       string    `json:"url"`
	Status    string    `json:"status"`
	LastCheck time.Time `json:"last_check"`
}
