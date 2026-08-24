package probes

import "time"

type KNIRVChainProbe struct {
	url string
}

func NewKNIRVChainProbe(url string) *KNIRVChainProbe {
	return &KNIRVChainProbe{url: url}
}

func (p *KNIRVChainProbe) Name() string {
	return "knirvchain"
}

func (p *KNIRVChainProbe) Scrape() (*ProbeResult, error) {
	return scrapeURL(p.url)
}

type KNIRVChainHealth struct {
	URL       string    `json:"url"`
	Status    string    `json:"status"`
	LastCheck time.Time `json:"last_check"`
}
